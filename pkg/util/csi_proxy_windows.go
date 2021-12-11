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
	"k8s.io/klog/v2"
)

// CSIProxy is the mounter interface exposed as a utility to
// internal methods
type CSIProxy interface {
	// GetAPIVersions returns the versions of the client APIs this mounter is using.
	GetAPIVersions() string

	// GetVolumeId returns the `volumeId` of the volume mounted at `mountPath`
	GetVolumeId(mountPath string) (volumeId string, err error)

	// GetVolumeStats gets the volume stats of a volume identified by `volumeId`
	GetVolumeStats(volumeId string) (totalBytes int64, usedBytes int64, err error)

	// FormatVolume formats a volume identified by `volumeId`
	FormatVolume(volumeId string) (err error)

	// IsSymlink checks if the given path is a symlink
	IsSymlink(mountPath string) (isSymlink bool, err error)
}

// NewCSIProxy returns an instance of the CSIProxy client compatible with either v1 or v1beta
func NewCSIProxy() (CSIProxy, error) {
	csiProxyV1, err := NewCSIProxyV1()
	if err == nil {
		klog.V(2).Infof("using CSIProxyV1, %s", csiProxyV1.GetAPIVersions())
		return csiProxyV1, nil
	}
	klog.V(4).Infof("failed to connect to csi-proxy v1 with error=%v, will try with v1Beta", err)

	csiProxyV1Beta, err := NewCSIProxyV1Beta()
	if err == nil {
		klog.V(4).Infof("using CSIProxyV1Beta, %s", csiProxyV1Beta.GetAPIVersions())
		return csiProxyV1Beta, nil
	}
	klog.Errorf("failed to connect to csi-proxy v1beta with error=%v", err)
	return nil, err
}
