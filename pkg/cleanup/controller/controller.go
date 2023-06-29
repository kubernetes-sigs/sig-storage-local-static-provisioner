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

	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/cleanup/deleter"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"
)

type EntryPair struct {
	nodeName string
	pv       *v1.PersistentVolume
}

type CleanupController struct {
	client kubernetes.Interface

	// entryQueue is a rate-limited delayed queue. PVs and their corresponding Node are added
	// as an EntryPair to this queue when the Node is deleted. The queue's delay allows to wait
	// for the Node to come back up before cleaning up resources.
	entryQueue workqueue.RateLimitingInterface

	nodeLister       corelisters.NodeLister
	nodeListerSynced cache.InformerSynced

	pvLister       corelisters.PersistentVolumeLister
	pvListerSynced cache.InformerSynced

	eventRecorder record.EventRecorder
	broadcaster   record.EventBroadcaster

	// storageClassName is the name of the StorageClass that PVs and PVCs
	// must belong to in order to be cleaned up.
	storageClassName string

	// delay is the amount of time to wait after Node deletion to cleanup resources.
	delay time.Duration

	// stalePVDiscoveryInterval is how often to scan for and delete PVs with affinity to a deleted Node.
	stalePVDiscoveryInterval time.Duration
}

func NewCleanupController(client kubernetes.Interface, pvInformer coreinformers.PersistentVolumeInformer, nodeInformer coreinformers.NodeInformer, storageClassName string, delay time.Duration, stalePVDiscoveryInterval time.Duration) *CleanupController {
	broadcaster := record.NewBroadcaster()
	eventRecorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: fmt.Sprintf("cleanup-controller")})

	controller := &CleanupController{
		client:           client,
		storageClassName: storageClassName,
		// Delayed queue with rate limiting
		entryQueue: workqueue.NewRateLimitingQueueWithConfig(
			workqueue.DefaultControllerRateLimiter(),
			workqueue.RateLimitingQueueConfig{
				Name: "stale",
			}),
		nodeLister:               nodeInformer.Lister(),
		nodeListerSynced:         nodeInformer.Informer().HasSynced,
		pvLister:                 pvInformer.Lister(),
		pvListerSynced:           pvInformer.Informer().HasSynced,
		eventRecorder:            eventRecorder,
		broadcaster:              broadcaster,
		delay:                    delay,
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
	defer c.entryQueue.ShutDown()

	klog.Info("Starting to Run CleanupController")

	c.broadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: c.client.CoreV1().Events(v1.NamespaceAll)})
	defer c.broadcaster.Shutdown()

	klog.Info("Waiting for informer caches to sync")
	// Wait for the caches to be synced before starting workers
	if ok := cache.WaitForCacheSync(ctx.Done(), c.nodeListerSynced, c.pvListerSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Infof("Starting workers, count: &d", workers)
	// Launch workers to process
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	klog.Info("Started workers")

	// Look for stale PVs and start timers for resource cleanup
	c.startCleanupTimersIfNeeded()

	// Run Deleter and block until channel is closed
	deleter := deleter.NewDeleter(c.client, c.pvLister, c.nodeLister, c.storageClassName)
	deleter.Run(ctx, c.stalePVDiscoveryInterval)

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
	obj, shutdown := c.entryQueue.Get()
	if shutdown {
		return false
	}
	defer c.entryQueue.Done(obj)

	entryPair, ok := obj.(EntryPair)
	if !ok {
		// Item is invalid so we can forget it.
		c.entryQueue.Forget(obj)
		klog.Errorf("expected EntryPair in workqueue but got %+v", obj)
		return true
	}

	err := c.syncHandler(ctx, entryPair)
	if err != nil {
		// An error occurred so re-add the item to the queue to work on later (has backoff to avoid
		// hot-looping).
		c.entryQueue.AddRateLimited(obj)
		klog.Errorf("error syncing %q: %w, requeuing", entryPair.pv.Name, err)
		return true
	}

	c.entryQueue.Forget(obj)
	return true
}

func (c *CleanupController) syncHandler(ctx context.Context, entryPair EntryPair) error {
	nodeExists, err := common.NodeExists(c.nodeLister, entryPair.nodeName)
	if err != nil {
		return err
	}

	if nodeExists {
		// Node is back up so cleanup is not needed.
		return nil
	}

	pvClaimRef := entryPair.pv.Spec.ClaimRef
	if pvClaimRef == nil {
		return nil
	}

	err = c.client.CoreV1().PersistentVolumeClaims(pvClaimRef.Namespace).Delete(ctx, pvClaimRef.Name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// The PVC could already be deleted by some other process, in
			// which case we stop processing.
			klog.Infof("PVC %q in queue no longer exists", pvClaimRef.Name)
			return nil
		}
		return err
	}

	klog.Infof("Deleted PVC %q that pointed to Node %q", pvClaimRef.Name, entryPair.nodeName)
	return nil
}

func (c *CleanupController) nodeDeleted(obj interface{}) {
	c.startCleanupTimersIfNeeded()
}

// startCleanupTimersIfNeeded enqueues any local PVs
// with a given StorageClass and a NodeAffinity to a deleted Node.
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

		entry := &EntryPair{nodeName, pv}

		shouldEnqueue, err := c.shouldEnqueueEntry(entry)
		if err != nil {
			klog.Errorf("error determining whether to enqueue entry with pv %q: %w", pv.Name, err)
			continue
		}

		if shouldEnqueue {
			klog.Infof("Starting timer for resource deletion, resource:%s, timer duration: %s", pv.Spec.ClaimRef, c.delay.String())
			c.eventRecorder.Event(pv.Spec.ClaimRef, v1.EventTypeWarning, "ReferencedNodeDeleted", fmt.Sprintf("PVC is tied to a deleted Node. PVC will be cleaned up in %s if the Node doesn't come back", c.delay.String()))

			c.entryQueue.AddAfter(*entry, c.delay)
		}
	}
}

// shouldEnqueuePV checks if a PV should be enqueued to the entryQueue.
// The PV must be a local PV, have a given StorageClass, have a NodeAffinity
// to a deleted Node, and have a PVC bound to it (otherwise there's nothing to clean up).
func (c *CleanupController) shouldEnqueueEntry(entry *EntryPair) (bool, error) {
	if !common.IsLocalPVWithStorageClass(entry.pv, c.storageClassName) || entry.pv.Spec.ClaimRef == nil {
		return false, nil
	}

	exists, err := common.NodeExists(c.nodeLister, entry.nodeName)
	return !exists && err == nil, err
}
