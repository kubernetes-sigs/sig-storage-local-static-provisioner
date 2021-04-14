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

package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/metrics"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	storagev1listers "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/cache"
	esUtil "sigs.k8s.io/sig-storage-lib-external-provisioner/v6/util"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/deleter"
)

// Discoverer finds available volumes and creates PVs for them
// It looks for volumes in the directories specified in the discoveryMap
type Discoverer struct {
	*common.RuntimeConfig
	Labels map[string]string
	// ProcTable is a reference to running processes so that we can prevent PV from being created while
	// it is being cleaned
	CleanupTracker  *deleter.CleanupStatusTracker
	nodeAffinityAnn string
	nodeAffinity    *v1.VolumeNodeAffinity
	classLister     storagev1listers.StorageClassLister
	ownerReference  *metav1.OwnerReference

	Readyz *readyzCheck
}

type readyzCheck struct {
	ready     bool
	readySync sync.RWMutex
}

// Check returns an error if the discovery state is not ready
func (d *readyzCheck) Check(_ *http.Request) error {
	d.readySync.RLock()
	defer d.readySync.RUnlock()
	if d.ready {
		return nil
	}
	return errors.New("discovererNotReady")
}

// Name returns the name of this ReadyzCheck
func (d *readyzCheck) Name() string {
	return "DiscovererReadyzCheck"
}

// NewDiscoverer creates a Discoverer object that will scan through
// the configured directories and create local PVs for any new directories found
func NewDiscoverer(config *common.RuntimeConfig, cleanupTracker *deleter.CleanupStatusTracker) (*Discoverer, error) {
	sharedInformer := config.InformerFactory.Storage().V1().StorageClasses()
	sharedInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		// We don't need an actual event handler for StorageClasses,
		// but we must pass a non-nil one to cache.NewInformer()
		AddFunc:    nil,
		UpdateFunc: nil,
		DeleteFunc: nil,
	})

	labelMap := make(map[string]string)
	for _, labelName := range config.NodeLabelsForPV {
		labelVal, ok := config.Node.Labels[labelName]
		if ok {
			labelMap[labelName] = labelVal
		}
	}

	// Also add any additional labels configured for the PVs
	for labelName, labelValue := range config.LabelsForPV {
		labelMap[labelName] = labelValue
	}

	// Generate owner reference
	ownerRef, err := generateOwnerReference(config.Node)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate owner reference: %v", err)
	}

	if config.UseAlphaAPI {
		nodeAffinity, err := generateNodeAffinity(config.Node)
		if err != nil {
			return nil, fmt.Errorf("Failed to generate node affinity: %v", err)
		}
		tmpAnnotations := map[string]string{}
		err = StorageNodeAffinityToAlphaAnnotation(tmpAnnotations, nodeAffinity)
		if err != nil {
			return nil, fmt.Errorf("Failed to convert node affinity to alpha annotation: %v", err)
		}
		return &Discoverer{
			RuntimeConfig:   config,
			Labels:          labelMap,
			CleanupTracker:  cleanupTracker,
			classLister:     sharedInformer.Lister(),
			nodeAffinityAnn: tmpAnnotations[common.AlphaStorageNodeAffinityAnnotation],
			ownerReference:  ownerRef,
			Readyz:          &readyzCheck{},
		}, nil
	}

	volumeNodeAffinity, err := generateVolumeNodeAffinity(config.Node)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate volume node affinity: %v", err)
	}

	return &Discoverer{
		RuntimeConfig:  config,
		Labels:         labelMap,
		CleanupTracker: cleanupTracker,
		classLister:    sharedInformer.Lister(),
		nodeAffinity:   volumeNodeAffinity,
		ownerReference: ownerRef,
		Readyz:         &readyzCheck{},
	}, nil
}

func generateOwnerReference(node *v1.Node) (*metav1.OwnerReference, error) {
	if node.GetName() == "" {
		return nil, fmt.Errorf("Node does not have name")
	}

	if node.GetUID() == "" {
		return nil, fmt.Errorf("Node does not have UID")
	}

	return &metav1.OwnerReference{
		Kind:       "Node",
		APIVersion: "v1",
		Name:       node.GetName(),
		UID:        node.UID,
	}, nil
}

func generateNodeAffinity(node *v1.Node) (*v1.NodeAffinity, error) {
	if node.Labels == nil {
		return nil, fmt.Errorf("Node does not have labels")
	}
	nodeValue, found := node.Labels[common.NodeLabelKey]
	if !found {
		return nil, fmt.Errorf("Node does not have expected label %s", common.NodeLabelKey)
	}

	return &v1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
			NodeSelectorTerms: []v1.NodeSelectorTerm{
				{
					MatchExpressions: []v1.NodeSelectorRequirement{
						{
							Key:      common.NodeLabelKey,
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{nodeValue},
						},
					},
				},
			},
		},
	}, nil
}

func generateVolumeNodeAffinity(node *v1.Node) (*v1.VolumeNodeAffinity, error) {
	if node.Labels == nil {
		return nil, fmt.Errorf("Node does not have labels")
	}
	nodeValue, found := node.Labels[common.NodeLabelKey]
	if !found {
		return nil, fmt.Errorf("Node does not have expected label %s", common.NodeLabelKey)
	}

	return &v1.VolumeNodeAffinity{
		Required: &v1.NodeSelector{
			NodeSelectorTerms: []v1.NodeSelectorTerm{
				{
					MatchExpressions: []v1.NodeSelectorRequirement{
						{
							Key:      common.NodeLabelKey,
							Operator: v1.NodeSelectorOpIn,
							Values:   []string{nodeValue},
						},
					},
				},
			},
		},
	}, nil
}

// DiscoverLocalVolumes reads the configured discovery paths, and creates PVs for the new volumes
func (d *Discoverer) DiscoverLocalVolumes() {
	readyz := true
	for class, config := range d.DiscoveryMap {
		err := d.discoverVolumesAtPath(class, config)
		if err != nil {
			klog.Errorf("Failed to discover local volumes: %v", err)
			readyz = false
		}
	}
	d.Readyz.readySync.Lock()
	d.Readyz.ready = readyz
	d.Readyz.readySync.Unlock()
}

func (d *Discoverer) getReclaimPolicyFromStorageClass(name string) (v1.PersistentVolumeReclaimPolicy, error) {
	class, err := d.classLister.Get(name)
	if err != nil {
		return "", err
	}
	if class.ReclaimPolicy != nil {
		return *class.ReclaimPolicy, nil
	}
	return v1.PersistentVolumeReclaimDelete, nil
}

func (d *Discoverer) getMountOptionsFromStorageClass(name string) ([]string, error) {
	class, err := d.classLister.Get(name)
	if err != nil {
		return nil, err
	}

	return class.MountOptions, nil
}

func (d *Discoverer) discoverVolumesAtPath(class string, config common.MountConfig) error {
	klog.V(7).Infof("Discovering volumes at hostpath %q, mount path %q for storage class %q", config.HostDir, config.MountDir, class)

	reclaimPolicy, err := d.getReclaimPolicyFromStorageClass(class)
	if err != nil {
		return fmt.Errorf("failed to get ReclaimPolicy from storage class %q: %v", class, err)
	}

	if reclaimPolicy != v1.PersistentVolumeReclaimRetain && reclaimPolicy != v1.PersistentVolumeReclaimDelete {
		return fmt.Errorf("unsupported ReclaimPolicy %q from storage class %q, supported policy are Retain and Delete", reclaimPolicy, class)
	}

	files, err := d.VolUtil.ReadDir(config.MountDir)
	if err != nil {
		return fmt.Errorf("error reading directory: %v", err)
	}

	// Retrieve list of mount points to iterate through discovered paths (aka files) below
	mountPoints, err := d.RuntimeConfig.Mounter.List()
	if err != nil {
		return fmt.Errorf("error retrieving mountpoints: %v", err)
	}
	// Put mount points into set for faster checks below
	type empty struct{}
	mountPointMap := make(map[string]empty)
	for _, mp := range mountPoints {
		mountPointMap[mp.Path] = empty{}
	}

	var discoErrors []error
	var totalCapacityBlockBytes, totalCapacityFSBytes int64
	for _, file := range files {
		if config.NamePattern != "" {
			matched, err := filepath.Match(config.NamePattern, file)
			if err != nil {
				return err
			}
			if !matched {
				klog.V(5).Infof("file(%s) under(%s) does not match pattern(%s)", file, config.MountDir, config.NamePattern)
				continue
			}
		}
		startTime := time.Now()
		filePath := filepath.Join(config.MountDir, file)
		volMode, err := common.GetVolumeMode(d.VolUtil, filePath)
		if err != nil {
			discoErrors = append(discoErrors, err)
			continue
		}
		// Check if PV already exists for it
		pvName := generatePVName(file, d.Node.Name, class)
		pv, exists := d.Cache.GetPV(pvName)
		if exists {
			if pv.Spec.VolumeMode != nil && *pv.Spec.VolumeMode == v1.PersistentVolumeBlock &&
				volMode == v1.PersistentVolumeFilesystem {
				err := fmt.Errorf("incorrect Volume Mode: PV %q requires block mode but path %q was in fs mode", pvName, filePath)
				discoErrors = append(discoErrors, err)
				d.Recorder.Eventf(pv, v1.EventTypeWarning, common.EventVolumeFailedDelete, err.Error())
			}
			continue
		}

		// Check that the local filePath is not already in use in any other local volume
		// note: this check relies on the cache only containing PVs from this node and no others
		outsidePath := filepath.Join(config.HostDir, file)
		existingPVNames := d.Cache.LookupPVsByPath(outsidePath)
		if len(existingPVNames) > 0 {
			errStr := fmt.Sprintf("Volume path already in use: PV %q wants path %q which was already found in %q.", pvName, outsidePath, strings.Join(existingPVNames, ","))
			klog.Errorf(errStr)
			continue
		}

		usejob := false
		if volMode == v1.PersistentVolumeBlock {
			usejob = d.RuntimeConfig.UseJobForCleaning
		}
		if d.CleanupTracker.InProgress(pvName, usejob) {
			klog.Infof("PV %s is still being cleaned, not going to recreate it", pvName)
			continue
		}

		// remove old cleanup status
		_, _, err = d.CleanupTracker.RemoveStatus(pvName, usejob)
		if err != nil {
			klog.Errorf("expected status exists and fail to remove cleanup status for pv %s", pvName)
			continue
		}

		mountOptions, err := d.getMountOptionsFromStorageClass(class)
		if err != nil {
			discoErrors = append(discoErrors, fmt.Errorf("failed to get mount options from storage class %s: %v", class, err))
			continue
		}

		var capacityByte int64
		desireVolumeMode := v1.PersistentVolumeMode(config.VolumeMode)
		switch volMode {
		case v1.PersistentVolumeBlock:
			capacityByte, err = d.VolUtil.GetBlockCapacityByte(filePath)
			if err != nil {
				discoErrors = append(discoErrors, fmt.Errorf("path %q block stats error: %v", filePath, err))
				continue
			}
			totalCapacityBlockBytes += capacityByte
			if desireVolumeMode == v1.PersistentVolumeBlock && len(mountOptions) != 0 {
				klog.Warningf("Path %q will be used to create block volume, "+
					"mount options %v will not take effect.", filePath, mountOptions)
			}
		case v1.PersistentVolumeFilesystem:
			if desireVolumeMode == v1.PersistentVolumeBlock {
				discoErrors = append(discoErrors, fmt.Errorf("path %q of filesystem mode cannot be used to create block volume", filePath))
				continue
			}
			// Validate that this path is an actual mountpoint
			if _, isMntPnt := mountPointMap[filePath]; isMntPnt == false {
				discoErrors = append(discoErrors, fmt.Errorf("path %q is not an actual mountpoint", filePath))
				continue
			}
			capacityByte, err = d.VolUtil.GetFsCapacityByte(filePath)
			if err != nil {
				discoErrors = append(discoErrors, fmt.Errorf("path %q fs stats error: %v", filePath, err))
				continue
			}
			totalCapacityFSBytes += capacityByte
		default:
			discoErrors = append(discoErrors, fmt.Errorf("path %q has unexpected volume type %q", filePath, volMode))
			continue
		}

		err = d.createPV(file, class, reclaimPolicy, mountOptions, config, capacityByte, desireVolumeMode, startTime)
		if err != nil {
			discoErrors = append(discoErrors, err)
		}
	}
	metrics.PersistentVolumeCapacityBytes.WithLabelValues(string(v1.PersistentVolumeBlock)).Set(float64(totalCapacityBlockBytes))
	metrics.PersistentVolumeCapacityBytes.WithLabelValues(string(v1.PersistentVolumeFilesystem)).Set(float64(totalCapacityFSBytes))
	if discoErrors == nil {
		return nil
	}
	return fmt.Errorf("%d error(s) while discovering volumes: %v", len(discoErrors), discoErrors)
}

func generatePVName(file, node, class string) string {
	h := fnv.New32a()
	h.Write([]byte(file))
	h.Write([]byte(node))
	h.Write([]byte(class))
	// This is the FNV-1a 32-bit hash
	return fmt.Sprintf("local-pv-%x", h.Sum32())
}

func (d *Discoverer) createPV(file, class string, reclaimPolicy v1.PersistentVolumeReclaimPolicy, mountOptions []string, config common.MountConfig, capacityByte int64, volMode v1.PersistentVolumeMode, startTime time.Time) error {
	pvName := generatePVName(file, d.Node.Name, class)
	outsidePath := filepath.Join(config.HostDir, file)

	klog.Infof("Found new volume at host path %q with capacity %d, creating Local PV %q, required volumeMode %q",
		outsidePath, capacityByte, pvName, volMode)

	localPVConfig := &common.LocalPVConfig{
		Name:            pvName,
		HostPath:        outsidePath,
		Capacity:        roundDownCapacityPretty(capacityByte),
		StorageClass:    class,
		ReclaimPolicy:   reclaimPolicy,
		ProvisionerName: d.Name,
		VolumeMode:      volMode,
		Labels:          d.Labels,
		MountOptions:    mountOptions,
		SetPVOwnerRef:   d.SetPVOwnerRef,
		OwnerReference:  d.ownerReference,
	}

	if d.UseAlphaAPI {
		localPVConfig.UseAlphaAPI = true
		localPVConfig.AffinityAnn = d.nodeAffinityAnn
	} else {
		localPVConfig.NodeAffinity = d.nodeAffinity
	}

	if config.FsType != "" {
		localPVConfig.FsType = &config.FsType
	}

	pvSpec := common.CreateLocalPVSpec(localPVConfig)

	_, err := d.APIUtil.CreatePV(pvSpec)
	if err != nil {
		return fmt.Errorf("error creating PV %q for volume at %q: %v", pvName, outsidePath, err)
	}
	klog.Infof("Created PV %q for volume at %q", pvName, outsidePath)
	mode := string(volMode)
	metrics.PersistentVolumeDiscoveryTotal.WithLabelValues(mode).Inc()
	metrics.PersistentVolumeDiscoveryDurationSeconds.WithLabelValues(mode).Observe(time.Since(startTime).Seconds())
	return nil
}

// Round down the capacity to an easy to read value.
func roundDownCapacityPretty(capacityBytes int64) int64 {

	easyToReadUnitsBytes := []int64{esUtil.GiB, esUtil.MiB}

	// Round down to the nearest easy to read unit
	// such that there are at least 10 units at that size.
	for _, easyToReadUnitBytes := range easyToReadUnitsBytes {
		// Round down the capacity to the nearest unit.
		size := capacityBytes / easyToReadUnitBytes
		if size >= 10 {
			return size * easyToReadUnitBytes
		}
	}
	return capacityBytes
}

// GetStorageNodeAffinityFromAnnotation gets the json serialized data from PersistentVolume.Annotations
// and converts it to the NodeAffinity type in core.
func GetStorageNodeAffinityFromAnnotation(annotations map[string]string) (*v1.NodeAffinity, error) {
	if len(annotations) > 0 && annotations[common.AlphaStorageNodeAffinityAnnotation] != "" {
		var affinity v1.NodeAffinity
		err := json.Unmarshal([]byte(annotations[common.AlphaStorageNodeAffinityAnnotation]), &affinity)
		if err != nil {
			return nil, err
		}
		return &affinity, nil
	}
	return nil, nil
}

// StorageNodeAffinityToAlphaAnnotation converts NodeAffinity type to Alpha annotation for use in PersistentVolumes
func StorageNodeAffinityToAlphaAnnotation(annotations map[string]string, affinity *v1.NodeAffinity) error {
	if affinity == nil {
		return nil
	}

	json, err := json.Marshal(*affinity)
	if err != nil {
		return err
	}
	annotations[common.AlphaStorageNodeAffinityAnnotation] = string(json)
	return nil
}
