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
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/metrics"
	cleanupmetrics "sigs.k8s.io/sig-storage-local-static-provisioner/pkg/metrics/node-cleanup"
)

// Deleter handles cleanup of local PVs with an affinity to a deleted Node.
// Only PVs with a StorageClass listed in the storageClassNames will be considered for cleanup.
type Deleter struct {
	client            kubernetes.Interface
	pvLister          corelisters.PersistentVolumeLister
	nodeLister        corelisters.NodeLister
	storageClassNames []string
}

// NewDeleter creates a Deleter object to handle the deletion of local PVs
// that have an affinity to a deleted Node and have a StorageClass listed in storageClassNames.
func NewDeleter(client kubernetes.Interface, pvLister corelisters.PersistentVolumeLister, nodeLister corelisters.NodeLister, storageClassNames []string) *Deleter {
	return &Deleter{
		client:            client,
		pvLister:          pvLister,
		nodeLister:        nodeLister,
		storageClassNames: storageClassNames,
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
				cleanupmetrics.PersistentVolumeDeleteFailedTotal.WithLabelValues(string(phase), string(reclaimPolicy)).Inc()
				klog.Errorf("Error deleting PV: %s", pv.Name)
				continue
			}
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
	cleanupmetrics.APIServerRequestsTotal.WithLabelValues(metrics.APIServerRequestDelete).Inc()
	err := d.client.CoreV1().PersistentVolumes().Delete(ctx, pvName, metav1.DeleteOptions{})
	if err != nil && errors.IsNotFound(err) {
		klog.Warningf("PV %q no longer exists", pvName)
		return nil
	}
	return err
}
