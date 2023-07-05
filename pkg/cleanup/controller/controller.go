/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"
)

type CleanupController struct {
	client kubernetes.Interface

	// pvQueue is a rate-limited delayed queue. PVs names are added
	// to this queue when its corresponding Node is deleted. The queue's delay allows to wait
	// for the Node to come back up before cleaning up resources.
	pvQueue workqueue.RateLimitingInterface

	nodeLister       corelisters.NodeLister
	nodeListerSynced cache.InformerSynced

	pvLister       corelisters.PersistentVolumeLister
	pvListerSynced cache.InformerSynced

	pvcLister       corelisters.PersistentVolumeClaimLister
	pvcListerSynced cache.InformerSynced

	eventRecorder record.EventRecorder
	broadcaster   record.EventBroadcaster

	// storageClassNames is the list StorageClasses that PVs and PVCs
	// can belong to in order to be eligible for cleanup
	storageClassNames []string

	// pvcDeletionDelay is the amount of time to wait after Node deletion to cleanup resources.
	pvcDeletionDelay time.Duration

	// stalePVDiscoveryInterval is how often to scan for and delete PVs with affinity to a deleted Node.
	stalePVDiscoveryInterval time.Duration
}

func NewCleanupController(client kubernetes.Interface, pvInformer coreinformers.PersistentVolumeInformer, pvcInformer coreinformers.PersistentVolumeClaimInformer, nodeInformer coreinformers.NodeInformer, storageClassNames []string, pvcDeletionDelay time.Duration, stalePVDiscoveryInterval time.Duration) *CleanupController {
	broadcaster := record.NewBroadcaster()
	eventRecorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: fmt.Sprintf("cleanup-controller")})

	controller := &CleanupController{
		client:            client,
		storageClassNames: storageClassNames,
		// Delayed queue with rate limiting
		pvQueue: workqueue.NewRateLimitingQueueWithConfig(
			workqueue.DefaultControllerRateLimiter(),
			workqueue.RateLimitingQueueConfig{
				Name: "stalePVQueue",
			}),
		nodeLister:               nodeInformer.Lister(),
		nodeListerSynced:         nodeInformer.Informer().HasSynced,
		pvLister:                 pvInformer.Lister(),
		pvListerSynced:           pvInformer.Informer().HasSynced,
		pvcLister:                pvcInformer.Lister(),
		pvcListerSynced:          pvcInformer.Informer().HasSynced,
		eventRecorder:            eventRecorder,
		broadcaster:              broadcaster,
		pvcDeletionDelay:         pvcDeletionDelay,
		stalePVDiscoveryInterval: stalePVDiscoveryInterval,
	}

	// Set up event handler for when Nodes are deleted
	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: controller.nodeDeleted,
	})

	return controller
}

// Run will start worker threads that try to process items off of the entryQueue, as well
// as syncing informer caches and continuously running the Deleter.
// It will block until the context gives a Done signal, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *CleanupController) Run(ctx context.Context, workers int) error {
	defer utilruntime.HandleCrash()
	defer c.pvQueue.ShutDown()

	klog.Info("Starting to Run CleanupController")

	c.broadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: c.client.CoreV1().Events(v1.NamespaceAll)})
	defer c.broadcaster.Shutdown()

	klog.Info("Waiting for informer caches to sync")
	// Wait for the caches to be synced before starting workers
	if ok := cache.WaitForCacheSync(ctx.Done(), c.nodeListerSynced, c.pvListerSynced, c.pvcListerSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Infof("Starting workers, count: %d", workers)
	// Launch workers to process
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	klog.Info("Started workers")

	// Look for stale PVs and start timers for resource cleanup
	c.startCleanupTimersIfNeeded()

	<-ctx.Done()
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the entryQueue.
func (c *CleanupController) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

// processNextWorkItem will read a single work item off the entryQueue and
// attempt to process it, by calling the syncHandler.
func (c *CleanupController) processNextWorkItem(ctx context.Context) bool {
	key, shutdown := c.pvQueue.Get()
	if shutdown {
		return false
	}
	defer c.pvQueue.Done(key)

	pvName, ok := key.(string)
	if !ok {
		// Item is invalid so we can forget it.
		c.pvQueue.Forget(key)
		klog.Errorf("expected string in workqueue but got %+v", key)
		return true
	}

	err := c.syncHandler(ctx, pvName)
	if err != nil {
		// An error occurred so re-add the item to the queue to work on later (has backoff to avoid
		// hot-looping).
		c.pvQueue.AddRateLimited(key)
		klog.Errorf("error syncing %q: %w, requeuing", pvName, err)
		return true
	}

	c.pvQueue.Forget(key)
	return true
}

// syncHandler processes a PV by deleting the PVC bound to if it's
// associated Node is gone.
func (c *CleanupController) syncHandler(ctx context.Context, pvName string) error {
	pv, err := c.pvLister.Get(pvName)
	if err != nil {
		if errors.IsNotFound(err) {
			// PV was deleted in the meantime, ignore.
			klog.Infof("PV %q in queue no longer exists", pvName)
			return nil
		}
		return err
	}

	nodeName, ok := common.NodeAttachedToLocalPV(pv)
	if !ok {
		// For whatever reason the PV isn't formatted properly so we will
		// never be able to get its corresponding Node, so ignore.
		klog.Errorf("error getting node attached to pv: %s", pv)
		return nil
	}

	nodeExists, err := common.NodeExists(c.nodeLister, nodeName)
	if err != nil {
		return err
	}
	// Check that the node the PV/PVC reference is still deleted
	if nodeExists {
		return nil
	}

	pvClaimRef := pv.Spec.ClaimRef
	if pvClaimRef == nil {
		return nil
	}

	pvc, err := c.pvcLister.PersistentVolumeClaims(pvClaimRef.Namespace).Get(pvClaimRef.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			// PVC was deleted in the meantime, ignore.
			klog.Infof("PVC %q in namespace %q no longer exists", pvClaimRef.Name, pvClaimRef.Namespace)
			return nil
		}
		return err
	}
	// Check that the PVC we're about to delete still points back to the PV that enqueued it.
	if pvc.Spec.VolumeName != pv.Name {
		klog.Infof("PVC %q no longer references PV %q so will not be cleaned up", pvc.Name, pv.Name)
		return nil
	}

	err = c.deletePVC(ctx, pvc)
	if err != nil {
		klog.Errorf("failed to delete pvc %q in namespace &q: %w", pvClaimRef.Name, pvClaimRef.Namespace, err)
		return err
	}

	klog.Infof("Deleted PVC %q that pointed to Node %q", pvClaimRef.Name, nodeName)
	return nil
}

func (c *CleanupController) nodeDeleted(obj interface{}) {
	c.startCleanupTimersIfNeeded()
}

// startCleanupTimersIfNeeded enqueues any local PVs
// with a NodeAffinity to a deleted Node and a StorageClass listed in storageClassNames.
func (c *CleanupController) startCleanupTimersIfNeeded() {
	pvs, err := c.pvLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("error listing pvs: %w", err)
		return
	}

	for _, pv := range pvs {
		nodeName, ok := common.NodeAttachedToLocalPV(pv)
		if !ok {
			klog.Errorf("error getting node attached to pv: %s", pv)
			continue
		}

		shouldEnqueue, err := c.shouldEnqueueEntry(pv, nodeName)
		if err != nil {
			klog.Errorf("error determining whether to enqueue entry with pv %q: %w", pv.Name, err)
			continue
		}

		if shouldEnqueue {
			klog.Infof("Starting timer for resource deletion, resource:%s, timer duration: %s", pv.Spec.ClaimRef, c.pvcDeletionDelay.String())
			c.eventRecorder.Event(pv.Spec.ClaimRef, v1.EventTypeWarning, "ReferencedNodeDeleted", fmt.Sprintf("PVC is tied to a deleted Node. PVC will be cleaned up in %s if the Node doesn't come back", c.pvcDeletionDelay.String()))

			c.pvQueue.AddAfter(pv.Name, c.pvcDeletionDelay)
		}
	}
}

// shouldEnqueuePV checks if a PV should be enqueued to the entryQueue.
// The PV must be a local PV, have a StorageClass present in the list of storageClassNames, have a NodeAffinity
// to a deleted Node, and have a PVC bound to it (otherwise there's nothing to clean up).
func (c *CleanupController) shouldEnqueueEntry(pv *v1.PersistentVolume, nodeName string) (bool, error) {
	if !common.IsLocalPVWithStorageClass(pv, c.storageClassNames) || pv.Spec.ClaimRef == nil {
		return false, nil
	}

	exists, err := common.NodeExists(c.nodeLister, nodeName)
	return !exists && err == nil, err
}

// deletePVC deletes the PVC with the given name and namespace
// and returns nil if the operation was successful or if the PVC doesn't exist
func (c *CleanupController) deletePVC(ctx context.Context, pvc *v1.PersistentVolumeClaim) error {
	err := c.client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(ctx, pvc.Name, metav1.DeleteOptions{})
	if err != nil && errors.IsNotFound(err) {
		// The PVC could already be deleted by some other process
		klog.Infof("PVC %q in namespace %q no longer exists", pvc.Name, pvc.Namespace)
		return nil
	}
	return err
}
