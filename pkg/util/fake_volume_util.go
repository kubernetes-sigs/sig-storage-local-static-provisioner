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
	"path/filepath"

	"k8s.io/klog/v2"
)

var _ VolumeUtil = &FakeVolumeUtil{}

// FakeVolumeUtil is a stub interface for unit testing
type FakeVolumeUtil struct {
	// List of files underneath the given path
	directoryFiles map[string][]*FakeDirEntry
	// True if DeleteContents should fail
	deleteShouldFail bool
}

const (
	// FakeEntryFile is mock dir entry of type file.
	FakeEntryFile = "file"
	// FakeEntryBlock is mock dir entry of type block.
	FakeEntryBlock = "block"
	// FakeEntryUnknown is mock dir entry of type unknown.
	FakeEntryUnknown = "unknown"
)

// FakeDirEntry contains a representation of a file under a directory
type FakeDirEntry struct {
	Name       string
	VolumeType string
	// Expected hash value of the PV name
	Hash     uint32
	Capacity int64
}

// NewFakeVolumeUtil returns a VolumeUtil object for use in unit testing
func NewFakeVolumeUtil(deleteShouldFail bool, dirFiles map[string][]*FakeDirEntry) *FakeVolumeUtil {
	return &FakeVolumeUtil{
		directoryFiles:   dirFiles,
		deleteShouldFail: deleteShouldFail,
	}
}

// IsDir checks if the given path is a directory
func (u *FakeVolumeUtil) IsDir(fullPath string) (bool, error) {
	dir, file := filepath.Split(fullPath)
	dir = filepath.Clean(dir)
	files, found := u.directoryFiles[dir]
	if !found {
		return false, fmt.Errorf("Directory %q not found", dir)
	}

	for _, f := range files {
		if file == f.Name {
			if f.VolumeType != FakeEntryFile {
				// Accurately simulate how a check on a non file returns error with actual OS call.
				return false, fmt.Errorf("%q not a file or directory", fullPath)
			}
			return true, nil
		}
	}
	return false, fmt.Errorf("Directory entry %q not found", fullPath)
}

// IsBlock checks if the given path is a block device
func (u *FakeVolumeUtil) IsBlock(fullPath string) (bool, error) {
	dir, file := filepath.Split(fullPath)
	dir = filepath.Clean(dir)
	files, found := u.directoryFiles[dir]
	if !found {
		return false, fmt.Errorf("Directory %q not found", dir)
	}

	for _, f := range files {
		if file == f.Name {
			return f.VolumeType == FakeEntryBlock, nil
		}
	}
	return false, fmt.Errorf("Directory entry %q not found", fullPath)
}

// ReadDir returns the list of all files under the given directory
func (u *FakeVolumeUtil) ReadDir(fullPath string) ([]string, error) {
	fileNames := []string{}
	files, found := u.directoryFiles[fullPath]
	if !found {
		return nil, fmt.Errorf("Directory %q not found", fullPath)
	}
	for _, file := range files {
		fileNames = append(fileNames, file.Name)
	}
	return fileNames, nil
}

// DeleteContents removes all the contents under the given directory
func (u *FakeVolumeUtil) DeleteContents(hostPath, mountPath string) error {
	if u.deleteShouldFail {
		return fmt.Errorf("Fake delete contents failed")
	}
	return nil
}

// GetFsCapacityByte returns capacity in byte about a mounted filesystem.
func (u *FakeVolumeUtil) GetFsCapacityByte(hostPath, mountPath string) (int64, error) {
	return u.getDirEntryCapacity(mountPath, FakeEntryFile)
}

// GetBlockCapacityByte returns the space in the specified block device.
func (u *FakeVolumeUtil) GetBlockCapacityByte(fullPath string) (int64, error) {
	return u.getDirEntryCapacity(fullPath, FakeEntryBlock)
}

func (u *FakeVolumeUtil) getDirEntryCapacity(fullPath string, entryType string) (int64, error) {
	dir, file := filepath.Split(fullPath)
	dir = filepath.Clean(dir)
	files, found := u.directoryFiles[dir]
	if !found {
		return 0, fmt.Errorf("Directory %q not found", dir)
	}

	for _, f := range files {
		if file == f.Name {
			if f.VolumeType != entryType {
				return 0, fmt.Errorf("Directory entry %q is not a %q", f, entryType)
			}
			return f.Capacity, nil
		}
	}
	return 0, fmt.Errorf("Directory entry %q not found", fullPath)
}

// AddNewDirEntries adds the given files to the current directory listing
// This is only for testing
func (u *FakeVolumeUtil) AddNewDirEntries(mountDir string, dirFiles map[string][]*FakeDirEntry) {
	for dir, files := range dirFiles {
		mountedPath := filepath.Join(mountDir, dir)
		curFiles := u.directoryFiles[mountedPath]
		if curFiles == nil {
			curFiles = []*FakeDirEntry{}
		}
		klog.Infof("Adding to directory %q: files %v\n", dir, files)
		u.directoryFiles[mountedPath] = append(curFiles, files...)
	}
}

// IsLikelyMountPoint checks if the given path is likely a mountpoint
func (u *FakeVolumeUtil) IsLikelyMountPoint(hostPath, mountPath string, mountPointMap map[string]interface{}) (bool, error) {
	if _, isMntPnt := mountPointMap[mountPath]; isMntPnt == false {
		// mountPointMap is built in discovery.go by using k8s.io/utils/mount
		return false, fmt.Errorf("hostPath=%q wasn't found in the /proc/mounts file", hostPath)
	}
	return true, nil
}
