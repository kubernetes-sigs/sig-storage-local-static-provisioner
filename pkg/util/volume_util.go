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

package util

import (
	"os"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// VolumeUtil is an interface for local filesystem operations
type VolumeUtil interface {
	// IsDir checks if the given path is a directory
	IsDir(fullPath string) (bool, error)

	// IsBlock checks if the given path is a directory
	IsBlock(fullPath string) (bool, error)

	// IsLikelyMountPoint checks if the given path is likely a mountpoint
	IsLikelyMountPoint(hostPath, mountPath string, mountPointMap map[string]interface{}) (bool, error)

	// ReadDir returns a list of files under the specified directory
	ReadDir(fullPath string) ([]string, error)

	// Delete all the contents under the given path, but not the path itself
	DeleteContents(hostPath, mountPath string) error

	// Get capacity for fs on full path
	GetFsCapacityByte(hostPath, mountPath string) (int64, error)

	// Get capacity of the block device
	GetBlockCapacityByte(fullPath string) (int64, error)
}

// IsDir checks if the given path is a directory
func (u *volumeUtil) IsDir(fullPath string) (bool, error) {
	dir, err := os.Open(fullPath)
	if err != nil {
		return false, err
	}
	defer dir.Close()

	stat, err := dir.Stat()
	if err != nil {
		return false, err
	}

	return stat.IsDir(), nil
}

// ReadDir returns a list all the files under the given directory
func (u *volumeUtil) ReadDir(fullPath string) ([]string, error) {
	dir, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	return files, nil
}

// GetLocalPersistentVolumeNodeNames returns the node affinity node name(s) for
// local PersistentVolumes. nil is returned if the PV does not have any
// specific node affinity node selector terms and match expressions.
// PersistentVolume with node affinity has select and match expressions
// in the form of:
//
//	nodeAffinity:
//	  required:
//	    nodeSelectorTerms:
//	    - matchExpressions:
//	      - key: kubernetes.io/hostname
//	        operator: In
//	        values:
//	        - <node1>
//	        - <node2>
func GetLocalPersistentVolumeNodeNames(pv *v1.PersistentVolume) []string {
	if pv == nil || pv.Spec.NodeAffinity == nil || pv.Spec.NodeAffinity.Required == nil {
		return nil
	}

	var result sets.Set[string]
	for _, term := range pv.Spec.NodeAffinity.Required.NodeSelectorTerms {
		var nodes sets.Set[string]
		for _, matchExpr := range term.MatchExpressions {
			if matchExpr.Key == v1.LabelHostname && matchExpr.Operator == v1.NodeSelectorOpIn {
				if nodes == nil {
					nodes = sets.New(matchExpr.Values...)
				} else {
					nodes = nodes.Intersection(sets.New(matchExpr.Values...))
				}
			}
		}
		result = result.Union(nodes)
	}

	return sets.List(result)
}
