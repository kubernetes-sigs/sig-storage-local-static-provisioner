//go:build linux
// +build linux

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
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/unix"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/kubernetes/pkg/volume/util/fs"
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
	_, capacity, _, _, _, _, err := fs.Info(mountPath)
	return capacity, err
}

// GetBlockCapacityByte returns  capacity in bytes of a block device.
// fullPath is the pathname of block device.
func (u *volumeUtil) GetBlockCapacityByte(fullPath string) (int64, error) {
	file, err := os.OpenFile(fullPath, os.O_RDONLY, 0)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var size int64
	// Get size of block device into 64 bit int.
	// Ref: http://www.microhowto.info/howto/get_the_size_of_a_linux_block_special_device_in_c.html
	if _, _, err := unix.Syscall(unix.SYS_IOCTL, file.Fd(), unix.BLKGETSIZE64, uintptr(unsafe.Pointer(&size))); err != 0 {
		return 0, err
	}

	return size, err
}

// IsLikelyMountPoint is not implemented in linux because the discovery implementation
// already checks if a path is a mount point by analyzing the /proc/mounts file
func (u *volumeUtil) IsLikelyMountPoint(hostPath, mountPath string, mountPointMap map[string]interface{}) (bool, error) {
	if _, isMntPnt := mountPointMap[mountPath]; isMntPnt == false {
		// mountPointMap is built in discovery.go by using k8s.io/utils/mount
		return false, fmt.Errorf("mountPath=%q wasn't found in the /proc/mounts file", mountPath)
	}
	return true, nil
}

// IsBlock checks if the given path is a block device
func (u *volumeUtil) IsBlock(fullPath string) (bool, error) {
	var st unix.Stat_t
	err := unix.Stat(fullPath, &st)
	if err != nil {
		return false, err
	}

	return (st.Mode & unix.S_IFMT) == unix.S_IFBLK, nil
}

// DeleteContents deletes all the contents under the given directory
func (u *volumeUtil) DeleteContents(hostPath, mountPath string) error {
	dir, err := os.Open(mountPath)
	if err != nil {
		return err
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}
	errList := []error{}
	for _, file := range files {
		err = os.RemoveAll(filepath.Join(mountPath, file))
		if err != nil {
			errList = append(errList, err)
		}
	}

	return utilerrors.NewAggregate(errList)
}
