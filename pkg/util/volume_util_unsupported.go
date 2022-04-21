//go:build !linux && !windows
// +build !linux,!windows

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
	"fmt"
)

var _ VolumeUtil = &volumeUtil{}

type volumeUtil struct{}

// NewVolumeUtil returns a VolumeUtil object for performing local filesystem operations
func NewVolumeUtil() (VolumeUtil, error) {
	return &volumeUtil{}, nil
}

// GetFsCapacityByte returns capacity in bytes about a mounted filesystem.
// fullPath is the pathname of any file within the mounted filesystem. Capacity
// returned here is total capacity.
func (u *volumeUtil) GetFsCapacityByte(hostPath, mountPath string) (int64, error) {
	return 0, fmt.Errorf("GetFsCapacityByte is unsupported in this build")
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
	return false, fmt.Errorf("IsLikelyMountPoint is unsupported in this build")
}

// DeleteContents deletes all the contents under the given directory
func (u *volumeUtil) DeleteContents(hostPath, mountPath string) error {
	return fmt.Errorf("DeleteContents is unsupported in this build")
}
