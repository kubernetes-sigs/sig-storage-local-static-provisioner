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

	filesystemapi "github.com/kubernetes-csi/csi-proxy/client/api/filesystem/v1"
	volumeapi "github.com/kubernetes-csi/csi-proxy/client/api/volume/v1"
	filesystemclient "github.com/kubernetes-csi/csi-proxy/client/groups/filesystem/v1"
	volumeclient "github.com/kubernetes-csi/csi-proxy/client/groups/volume/v1"
)

// CSIProxyV1 is the CSI Proxy implementation that uses the v1 API
type CSIProxyV1 struct {
	VolumeClient     *volumeclient.Client
	FilesystemClient *filesystemclient.Client
}

// check that CSIProxyV1 implements CSIProxy
var _ CSIProxy = &CSIProxyV1{}

func NewCSIProxyV1() (*CSIProxyV1, error) {
	volumeClient, err := volumeclient.NewClient()
	if err != nil {
		return nil, err
	}
	filesystemClient, err := filesystemclient.NewClient()
	if err != nil {
		return nil, err
	}
	return &CSIProxyV1{
		VolumeClient:     volumeClient,
		FilesystemClient: filesystemClient,
	}, nil
}

// GetAPIVersions returns the versions of the client APIs.
func (proxy *CSIProxyV1) GetAPIVersions() string {
	return fmt.Sprintf(
		"API Versions Volume: %s",
		volumeclient.Version,
	)
}

// GetVolumeId returns the volumeId of the volume mounted at `mountPath`
func (proxy *CSIProxyV1) GetVolumeId(mountPath string) (volumeId string, err error) {
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
func (proxy *CSIProxyV1) GetVolumeStats(volumeId string) (totalBytes int64, usedBytes int64, err error) {
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

// FormatVolume formats a volume identified by `volumeId`
func (proxy *CSIProxyV1) FormatVolume(volumeId string) (err error) {
	_, err = proxy.VolumeClient.FormatVolume(
		context.Background(),
		&volumeapi.FormatVolumeRequest{
			VolumeId: volumeId,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// IsSymlink checks if the given path is a symlink
func (proxy *CSIProxyV1) IsSymlink(mountPath string) (isSymlink bool, err error) {
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
