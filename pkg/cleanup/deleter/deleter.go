package deleter

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"
)

// Deleter handles cleanup of local PVs with an affinity to a deleted Node.
// Only PVs with the passed-in storageClassName will be considered for cleanup.
type Deleter struct {
	client           kubernetes.Interface
	pvLister         corelisters.PersistentVolumeLister
	nodeLister       corelisters.NodeLister
	storageClassName string
}

// NewDeleter creates a Deleter object to handle the deletion of local PVs
// that use a given StorageClass and have an affinity to a deleted Node.
func NewDeleter(client kubernetes.Interface, pvLister corelisters.PersistentVolumeLister, nodeLister corelisters.NodeLister, storageClassName string) *Deleter {
	return &Deleter{
		client:           client,
		pvLister:         pvLister,
		nodeLister:       nodeLister,
		storageClassName: storageClassName,
	}
}

// Delete PVs will scan through PVs and delete those that are
// local PVs with a given StorageClass and have an affinity to a deleted Node.
func (d *Deleter) DeletePVs() {
	pvs, err := d.pvLister.List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("error listing pvs: %s", err.Error()))
		return
	}

	for _, pv := range pvs {
		if !common.IsLocalPVWithStorageClass(pv, d.storageClassName) {
			// Either isn't a local PV or doesn't have matching storage class.
			continue
		}

		referencesDeletedNode, err := d.referencesNonExistentNode(pv)
		if err != nil {
			utilruntime.HandleError(err)
			continue
		}
		if !referencesDeletedNode {
			// PV's node is up so PV is not stale
			continue
		}

		// PV is a stale object since it references a deleted Node.
		// Therefore it can safely be deleted in the two following cases.
		isReleasedWithDeleteReclaim := pv.Status.Phase == v1.VolumeReleased && pv.Spec.PersistentVolumeReclaimPolicy == v1.PersistentVolumeReclaimDelete
		isAvailable := pv.Status.Phase == v1.VolumeAvailable
		if isReleasedWithDeleteReclaim || isAvailable {
			klog.Infof("Deleting PV that has NodeAffinity to deleted Node, pv: %s", pv.Name)
			if err = d.deletePV(pv.Name); err != nil {
				klog.Errorf("Error deleting PV: %s", pv.Name)
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

func (d *Deleter) deletePV(pvName string) error {
	err := d.client.CoreV1().PersistentVolumes().Delete(context.TODO(), pvName, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("PV '%s' no longer exists", pvName))
			return nil
		}
	}
	return err
}
