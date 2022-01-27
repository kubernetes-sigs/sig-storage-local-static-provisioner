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
	"context"
	"fmt"

	filesystemapi "github.com/kubernetes-csi/csi-proxy/client/api/filesystem/v2alpha1"
	volumeapi "github.com/kubernetes-csi/csi-proxy/client/api/volume/v2alpha1"
	filesystemclient "github.com/kubernetes-csi/csi-proxy/client/groups/filesystem/v2alpha1"
	volumeclient "github.com/kubernetes-csi/csi-proxy/client/groups/volume/v2alpha1"
)

// CSIProxyV2 is the CSI Proxy implementation that uses the v2alpha1 API
type CSIProxyV2 struct {
	VolumeClient     *volumeclient.Client
	FilesystemClient *filesystemclient.Client
}

// check that CSIProxyV2 implements CSIProxy
var _ CSIProxy = &CSIProxyV2{}

func NewCSIProxyV2() (*CSIProxyV2, error) {
	volumeClient, err := volumeclient.NewClient()
	if err != nil {
		return nil, err
	}
	filesystemClient, err := filesystemclient.NewClient()
	if err != nil {
		return nil, err
	}
	return &CSIProxyV2{
		VolumeClient:     volumeClient,
		FilesystemClient: filesystemClient,
	}, nil
}

// GetAPIVersions returns the versions of the client APIs.
func (proxy *CSIProxyV2) GetAPIVersions() string {
	return fmt.Sprintf(
		"API Versions Volume: %s",
		volumeclient.Version,
	)
}

// GetVolumeId returns the volumeId of the volume mounted at `mountPath`
func (proxy *CSIProxyV2) GetVolumeId(mountPath string) (volumeId string, err error) {
	getVolumeIdFromTargetPathResponse, err := proxy.VolumeClient.GetVolumeIDFromTargetPath(
		context.Background(),
		&volumeapi.GetVolumeIDFromTargetPathRequest{
			TargetPath: mountPath,
		},
	)
	if err != nil {
		return "", err
	}
	return getVolumeIdFromTargetPathResponse.VolumeId, nil
}

// GetVolumeStats gets the volume stats of a volume identified by `volumeId`
func (proxy *CSIProxyV2) GetVolumeStats(volumeId string) (totalBytes int64, usedBytes int64, err error) {
	getVolumeStatsResponse, err := proxy.VolumeClient.GetVolumeStats(
		context.Background(),
		&volumeapi.GetVolumeStatsRequest{
			VolumeId: volumeId,
		},
	)
	if err != nil {
		return 0, 0, err
	}
	return getVolumeStatsResponse.TotalBytes, getVolumeStatsResponse.UsedBytes, nil
}

// RmdirContents removes the contents of a directory in the host filesystem.
func (proxy *CSIProxyV2) RmdirContents(path string) (err error) {
	_, err = proxy.FilesystemClient.RmdirContents(
		context.Background(),
		&filesystemapi.RmdirContentsRequest{
			Path: path,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// IsSymlink checks if the given path is a symlink
func (proxy *CSIProxyV2) IsSymlink(mountPath string) (isSymlink bool, err error) {
	isSymlinkResponse, err := proxy.FilesystemClient.IsSymlink(
		context.Background(),
		&filesystemapi.IsSymlinkRequest{
			Path: mountPath,
		},
	)
	if err != nil {
		return false, err
	}
	return isSymlinkResponse.IsSymlink, nil
}

// GetClosestVolumeIDFromTargetPath gets the closest volume id for a given target path
// by following symlinks and moving up in the filesystem, if after moving up in the filesystem
// we get to a DriveLetter then the volume corresponding to this drive letter is returned instead.
func (proxy *CSIProxyV2) GetClosestVolumeIDFromTargetPath(targetPath string) (volumeId string, err error) {
	getClosestVolumeIDFromTargetPathResponse, err := proxy.VolumeClient.GetClosestVolumeIDFromTargetPath(
		context.Background(),
		&volumeapi.GetClosestVolumeIDFromTargetPathRequest{
			TargetPath: targetPath,
		},
	)
	if err != nil {
		return "", err
	}
	return getClosestVolumeIDFromTargetPathResponse.VolumeId, nil
}
