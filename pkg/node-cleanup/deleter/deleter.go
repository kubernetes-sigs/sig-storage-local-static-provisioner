/*
Copyright 2023 The Kubernetes Authors.

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

package deleter

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"
	cleanupmetrics "sigs.k8s.io/sig-storage-local-static-provisioner/pkg/metrics/node-cleanup"
)

// Deleter handles cleanup of local PVs with an affinity to a deleted Node.
// Only PVs with a StorageClass listed in the storageClassNames will be considered for cleanup.
type Deleter struct {
	client            kubernetes.Interface
	pvLister          corelisters.PersistentVolumeLister
	nodeLister        corelisters.NodeLister
	storageClassNames []string
	namespacesToWatch []string

	// When recreatePvc is true, PVCs are recreated after deleting them
	recreatePvc bool
}

// NewDeleter creates a Deleter object to handle the deletion of local PVs
// that have an affinity to a deleted Node and have a StorageClass listed in storageClassNames.
func NewDeleter(client kubernetes.Interface, pvLister corelisters.PersistentVolumeLister, nodeLister corelisters.NodeLister, storageClassNames []string, recreatePvc bool, namespacesToWatch []string) *Deleter {
	return &Deleter{
		client:            client,
		pvLister:          pvLister,
		nodeLister:        nodeLister,
		storageClassNames: storageClassNames,
		recreatePvc:       recreatePvc,
		namespacesToWatch: namespacesToWatch,
	}
}

// Run will delete stale PVs on a given interval until the given context is done.
func (d *Deleter) Run(ctx context.Context, discoveryInterval time.Duration) {
	for {
		select {
		case <-ctx.Done():
			klog.Info("Deleter stopped")
			return
		default:
			d.DeletePVs(ctx)
			time.Sleep(discoveryInterval)
		}
	}
}

// DeletePVs will scan through PVs and delete those that are
// local PVs with a StorageClass listed in storageClassNames and have an affinity to a deleted Node.
func (d *Deleter) DeletePVs(ctx context.Context) {
	pvs, err := d.pvLister.List(labels.Everything())
	if err != nil {
		klog.Errorf("error listing pvs: %s", err.Error())
		return
	}

	for _, pv := range pvs {
		if !common.IsLocalPVWithStorageClass(pv, d.storageClassNames) {
			// Either isn't a local PV or doesn't have matching storage class.
			continue
		}

		referencesDeletedNode, err := d.referencesNonExistentNode(pv)
		if err != nil {
			klog.Errorf("error determining if pv %q references deleted node: %w", pv.Name, err)
			continue
		}
		if !referencesDeletedNode {
			// PV's node is up so PV is not stale
			continue
		}

		phase := pv.Status.Phase
		reclaimPolicy := pv.Spec.PersistentVolumeReclaimPolicy
		// PV is a stale object since it references a deleted Node.
		// Therefore it can safely be deleted in the two following cases.
		isReleasedWithDeleteReclaim := phase == v1.VolumeReleased && reclaimPolicy == v1.PersistentVolumeReclaimDelete
		isAvailable := phase == v1.VolumeAvailable
		if isReleasedWithDeleteReclaim || isAvailable {
			klog.Infof("Attempting to delete PV that has NodeAffinity to deleted Node, pv: %s", pv.Name)
			if err = d.deletePV(ctx, pv.Name); err != nil {
				cleanupmetrics.PersistentVolumeDeleteFailedTotal.WithLabelValues(string(phase)).Inc()
				klog.Errorf("Error deleting PV: %s", pv.Name)
				continue
			}
			// TODO: Cache successful deletion to avoid multiple delete calls
			// when there is a short sync period
			cleanupmetrics.PersistentVolumeDeleteTotal.WithLabelValues(string(phase)).Inc()
		}
	}

	if d.recreatePvc {
		klog.Infof("PVC recreation is turned on, checking pending pods with non existent PVCs...")
		err = d.recreatePVC(ctx)
		if err != nil {
			cleanupmetrics.PersistentVolumeClaimRecreateFailedTotal.Inc()
			klog.Errorf("failed to recreated pvc: %v", err)
		}
	}
}

// referencesNonExistentNode returns true if the local PV has a NodeAffinity to
// a deleted Node. An error is returned if the local PV's NodeAffinity
// does not have the form:
//
//	nodeAffinity:
//	  required:
//	    nodeSelectorTerms:
//	    - matchExpressions:
//	      - key: kubernetes.io/hostname
//	        operator: In
//	        values:
//	        - <node1>
func (d *Deleter) referencesNonExistentNode(localPV *v1.PersistentVolume) (bool, error) {
	nodeName, ok := common.NodeAttachedToLocalPV(localPV)
	if !ok {
		return false, fmt.Errorf("Error retrieving node")
	}

	exists, err := common.NodeExists(d.nodeLister, nodeName)
	return !exists && err == nil, err
}

func (d *Deleter) deletePV(ctx context.Context, pvName string) error {
	err := d.client.CoreV1().PersistentVolumes().Delete(ctx, pvName, metav1.DeleteOptions{})
	if err != nil && errors.IsNotFound(err) {
		klog.Warningf("PV %q no longer exists", pvName)
		return nil
	}
	return err
}

// recreatePVC recreates the PVC with the given name and namespace
// and returns nil if the operation was successful or if the PVC already exists
func (d *Deleter) recreatePVC(ctx context.Context) error {
	// list pods by namespace
	for _, namespace := range d.namespacesToWatch {
		klog.Infof("Looking for non existent PVCs keeping pods pending in namespace %q", namespace)
		podsInNamespace, err := d.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			klog.Errorf("failed to list pods in namespace %q: %v", namespace, err)
			continue
		}

		// recreate PVCs for each pod
		for _, pod := range podsInNamespace.Items {
			// check if the pod is in the `namespacesToWatch` list by doing a for loop over `d.namespacesToWatch`
			in_watched_namespace := false
			for _, namespace := range d.namespacesToWatch {
				in_watched_namespace = in_watched_namespace || pod.Namespace == namespace
			}
			if !in_watched_namespace {
				continue
			}
			// check if the pod is pending
			if pod.Status.Phase != v1.PodPending {
				continue
			}

			// check if the pod has a PVC claim with a storage class in `d.storageClassNames`
			has_pvc_to_recreate := false
			managed_storage_class := ""
			for _, volume := range pod.Spec.Volumes {
				if volume.PersistentVolumeClaim != nil {
					_, err := d.client.CoreV1().PersistentVolumeClaims(pod.Namespace).Get(ctx, volume.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
					if err == nil {
						continue
					}

					// check if the PVC name is in the `d.storageClassNames` list
					for _, storageClassName := range d.storageClassNames {
						supposedClaimName := fmt.Sprintf("%s-%s", storageClassName, pod.Name)
						if supposedClaimName == volume.PersistentVolumeClaim.ClaimName {
							has_pvc_to_recreate = true
							managed_storage_class = storageClassName
						}
					}
				}
			}

			if has_pvc_to_recreate {
				klog.Infof("Pod %q in namespace %q is in pending phase and has a missing PVC from managed storage class %q, it will be deleted so that the PVC is recreated automatically", pod.Name, pod.Namespace, managed_storage_class)

				err = d.client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
				if err != nil && errors.IsNotFound(err) {
					klog.Warningf("Pod %q in namespace %q no longer exists", pod.Name, pod.Namespace)
					continue
				} else if err != nil {
					cleanupmetrics.PersistentVolumeClaimRecreateFailedTotal.Inc()
					klog.Errorf("failed to delete pod %q in namespace %q: %v", pod.Name, pod.Namespace, err)
				} else {
					cleanupmetrics.PersistentVolumeClaimRecreateSuccessTotal.Inc()
					klog.Infof("Pod %q in namespace %q deleted successfully, PVC will be recreated shortly after", pod.Name, pod.Namespace)
				}
			}
		}
	}

	return nil
}
