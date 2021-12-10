//go:build windows
// +build windows

/*
Copyright 2021 The Kubernetes Authors.

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
	"fmt"
)

var _ VolumeUtil = &volumeUtil{}

type volumeUtil struct {
	// csiProxy is the CSIProxy implementation (either v1 or v1beta)
	csiProxy CSIProxy
}

// NewVolumeUtil returns a VolumeUtil object for performing local filesystem operations
func NewVolumeUtil() (VolumeUtil, error) {
	csiProxy, err := NewCSIProxy()
	if err != nil {
		return nil, err
	}
	return &volumeUtil{
		csiProxy,
	}, nil
}

// GetFsCapacityByte returns capacity in bytes about a mounted filesystem.
// In Windows the path is in the context of the host, not in the context of the container
// Capacity returned is total capacity.
func (u *volumeUtil) GetFsCapacityByte(hostPath, mountPath string) (int64, error) {
	volumeID, err := u.csiProxy.GetVolumeId(hostPath)
	if err != nil {
		return 0, err
	}
	totalBytes, _, err := u.csiProxy.GetVolumeStats(volumeID)
	if err != nil {
		return 0, err
	}
	return totalBytes, nil
}

// DeleteContents deletes all the contents under the given directory
func (u *volumeUtil) DeleteContents(hostPath, mountPath string) error {
	// mountPath is in the context of the volume inside local volume provisioner
	// the path to use in Windows is the one that CSI Proxy will use and it should
	// be in the context of the host (because CSI Proxy doesn't know about the context
	// of the local volume provisioner volumes)
	volumeID, err := u.csiProxy.GetVolumeId(hostPath)
	if err != nil {
		return err
	}
	err = u.csiProxy.FormatVolume(volumeID)
	if err != nil {
		return err
	}
	return nil
}

// GetBlockCapacityByte is defined here for darwin and other platforms
// so that make test succeeds on them.
func (u *volumeUtil) GetBlockCapacityByte(fullPath string) (int64, error) {
	return 0, fmt.Errorf("GetBlockCapacityByte is unsupported in this build")
}

// IsBlock for unsupported platform returns error.
func (u *volumeUtil) IsBlock(fullPath string) (bool, error) {
	return false, fmt.Errorf("IsBlock is unsupported in this build")
}

func (u *volumeUtil) IsLikelyMountPoint(hostPath, mountPath string, mountPointMap map[string]interface{}) (bool, error) {
	isLikelyMountPoint, err := u.csiProxy.IsSymlink(hostPath)
	if err != nil {
		return false, err
	}
	if !isLikelyMountPoint {
		return false, fmt.Errorf("hostPath %q is not a symlink", hostPath)
	}
	return isLikelyMountPoint, nil
}
