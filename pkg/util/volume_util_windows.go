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
	"os"
	"path/filepath"

	"k8s.io/klog/v2"
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
	fi, err := os.Lstat(mountPath)
	if err != nil {
		return 0, err
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return 0, fmt.Errorf("file mountPath=%q is not a symlink", mountPath)
	}

	var volumeID string
	var errNoVolumeID, errNoMoveUpVolumeID error

	// initially assume that the symlink points to a Volume{<volumeid>}
	volumePath := hostPath
	volumeID, errNoVolumeID = u.csiProxy.GetVolumeId(volumePath)

	if errNoVolumeID != nil {
		klog.V(5).Infof("path=%s is not a Volume, attempting to follow the symlink to get the volume capacity", volumePath)

		// in the case where it isn't a Volume{<volumeid>} assume that
		// after dereferencing the symlink and moving one level up
		// in the hierarchy we get a volumeID
		//
		// symlinkTarget is the result of dereferencing the symlink
		// it could be in the form Volume{<volumeid>} or an actual file
		// outside the mounted dir
		symlinkTarget, err := os.Readlink(mountPath)
		if err != nil {
			return 0, err
		}

		// the symlink is pointing to a directory, assume that it has this structure
		//
		// volume/ (symlink -> \\\Volume{}\)
		//   dir0/
		// disks/ (discovery directory)
		//   dir0/ (symlink -> volume/dir0)
		//
		// symlinkTarget is volume/dir0 at this point, trim the last element of the path
		volumePath = filepath.Dir(symlinkTarget)
		volumeID, errNoMoveUpVolumeID = u.csiProxy.GetVolumeId(volumePath)
		if errNoMoveUpVolumeID != nil {
			return 0, fmt.Errorf("Failed to find a volume either by checking if file=%s is a volume or by following the symlink, moving one level up and checking if file=%s is a volume", hostPath, volumePath, errNoMoveUpVolumeID)
		}
	}

	totalBytes, _, err := u.csiProxy.GetVolumeStats(volumeID)
	if err != nil {
		return 0, err
	}
	return totalBytes, nil
}

func (u *volumeUtil) recreateDirectory(hostPath string) error {
	err := u.csiProxy.Rmdir(hostPath, true)
	if err != nil {
		return err
	}
	err = u.csiProxy.Mkdir(hostPath)
	if err != nil {
		return err
	}
	return nil
}

// DeleteContents deletes all the contents under the given directory.
func (u *volumeUtil) DeleteContents(hostPath, mountPath string) error {
	// mountPath is in the context of the volume inside local volume provisioner
	// the path to use in Windows is the one that CSI Proxy will use and it should
	// be in the context of the host (because CSI Proxy doesn't know about the context
	// of the local volume provisioner volumes)

	// assume that it's pointing to a Volume
	volumeID, err := u.csiProxy.GetVolumeId(hostPath)
	if err == nil {
		// if so, reformat the volume
		err = u.csiProxy.FormatVolume(volumeID)
		if err != nil {
			return err
		}
		return nil
	}

	// otherwise it's pointing to a directory in the host
	klog.V(5).Info("Cannot format hostPath=%q because it's not a Volume, attempting to follow the symlink", hostPath)

	// symlinkTarget is the result of dereferencing the symlink
	// this path only makes sense in the context of the host
	symlinkTarget, err := os.Readlink(mountPath)
	if err != nil {
		return err
	}

	klog.V(5).Info("Attempting to delete hostPath=%q through CSI Proxy", symlinkTarget)
	// recreate the directory that the symlink points to
	err = u.recreateDirectory(symlinkTarget)
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
