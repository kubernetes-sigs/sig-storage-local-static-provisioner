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

package common

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/util"
)

func TestSetupClientByKubeConfigEnv(t *testing.T) {
	oldEnv := os.Getenv(KubeConfigEnv)
	os.Setenv(KubeConfigEnv, "/etc/foo/config")
	defer func() { os.Setenv(KubeConfigEnv, oldEnv) }()

	// Mock BuildConfigFromFlags
	oldBuildConfig := BuildConfigFromFlags
	defer func() { BuildConfigFromFlags = oldBuildConfig }()

	methodInvoked := false
	BuildConfigFromFlags = func(masterUrl, kubeconfigPath string) (*rest.Config, error) {
		methodInvoked = true
		if kubeconfigPath != "/etc/foo/config" {
			t.Errorf("Got unexpected oldEnv for config file %s", kubeconfigPath)
		}
		return &rest.Config{}, nil
	}

	SetupClient()
	if !methodInvoked {
		t.Errorf("BuildConfigFromFlags not invoked")
	}
}

func TestSetupClientByInCluster(t *testing.T) {
	// Make sure environment variable is unset
	if oldEnv := os.Getenv(KubeConfigEnv); oldEnv != "" {
		os.Unsetenv(KubeConfigEnv)
		defer func() { os.Setenv(KubeConfigEnv, oldEnv) }()
	}

	// Mock InClusterConfig
	oldInClusterConfig := InClusterConfig
	defer func() { InClusterConfig = oldInClusterConfig }()

	methodInvoked := false
	InClusterConfig = func() (*rest.Config, error) {
		methodInvoked = true
		return &rest.Config{}, nil
	}

	SetupClient()
	if !methodInvoked {
		t.Errorf("InClusterConfig not invoked")
	}
}

func TestLoadProvisionerConfigs(t *testing.T) {
	tmpConfigPath, err := ioutil.TempDir("", "local-provisioner-config")
	if err != nil {
		t.Fatalf("create temp dir error: %v", err)
	}
	defer func() {
		os.RemoveAll(tmpConfigPath)
	}()
	testcases := []struct {
		data        map[string]string
		expected    ProvisionerConfiguration
		expectedErr error
	}{
		{
			nil,
			ProvisionerConfiguration{},
			nil,
		},
		{
			map[string]string{
				"useAlphaAPI": "true",
			},
			ProvisionerConfiguration{
				UseAlphaAPI: true,
			},
			nil,
		},
		{
			map[string]string{
				"storageClassMap": `local-storage:
   hostDir: /mnt/disks
   mountDir: /mnt/disks
   fsType: ext4
`,
				"useAlphaAPI":     "true",
				"minResyncPeriod": "1h30m",
			},
			ProvisionerConfiguration{
				StorageClassConfig: map[string]MountConfig{
					"local-storage": {
						HostDir:             "/mnt/disks",
						MountDir:            "/mnt/disks",
						BlockCleanerCommand: []string{"/scripts/quick_reset.sh"},
						VolumeMode:          "Filesystem",
						FsType:              "ext4",
						NamePattern:         "*",
					},
				},
				UseAlphaAPI: true,
				MinResyncPeriod: metav1.Duration{
					Duration: time.Hour + time.Minute*30,
				},
			},
			nil,
		},
		{
			map[string]string{
				"storageClassMap": `local-storage:
   hostDir: /mnt/disks
   mountDir: /mnt/disks
   blockCleanerCommand:
   - "/scripts/shred.sh"
   - "2"
   volumeMode: WrongFilesystem
   fsType: ext4
   namePattern: nvm*,sdb*
`,
				"useAlphaAPI":     "true",
				"minResyncPeriod": "1h30m",
			},
			ProvisionerConfiguration{
				StorageClassConfig: map[string]MountConfig{
					"local-storage": {
						HostDir:             "/mnt/disks",
						MountDir:            "/mnt/disks",
						BlockCleanerCommand: []string{"/scripts/shred.sh", "2"},
						VolumeMode:          "WrongFilesystem",
						FsType:              "ext4",
						NamePattern:         "nvm*,sdb*",
					},
				},
				UseAlphaAPI: true,
				MinResyncPeriod: metav1.Duration{
					Duration: time.Hour + time.Minute*30,
				},
			},
			fmt.Errorf("unsupported volume mode WrongFilesystem"),
		},
		{
			map[string]string{
				"storageClassMap": `local-storage:
   hostDir: /mnt/disks
   blockCleanerCommand:
     - "/scripts/shred.sh"
     - "2"
   volumeMode: Filesystem
   fsType: ext4
   namePattern: nvm*,sdb*
`,
				"useAlphaAPI":     "true",
				"minResyncPeriod": "1h30m",
			},
			ProvisionerConfiguration{
				StorageClassConfig: map[string]MountConfig{
					"local-storage": {
						HostDir:             "/mnt/disks",
						BlockCleanerCommand: []string{"/scripts/shred.sh", "2"},
						VolumeMode:          "Filesystem",
						FsType:              "ext4",
						NamePattern:         "nvm*,sdb*",
					},
				},
				UseAlphaAPI: true,
				MinResyncPeriod: metav1.Duration{
					Duration: time.Hour + time.Minute*30,
				},
			},
			fmt.Errorf("Storage Class local-storage is misconfigured, missing HostDir or MountDir parameter"),
		},
		{
			map[string]string{
				"storageClassMap": `local-storage:
   hostDir: /mnt/disks
   mountDir: /mnt/disks
   blockCleanerCommand:
     - "/scripts/shred.sh"
     - "2"
   volumeMode: Filesystem
   fsType: ext4
   namePattern: nvm*,sdb*
`,
				"useAlphaAPI":     "true",
				"minResyncPeriod": "1h30m",
			},
			ProvisionerConfiguration{
				StorageClassConfig: map[string]MountConfig{
					"local-storage": {
						HostDir:             "/mnt/disks",
						MountDir:            "/mnt/disks",
						BlockCleanerCommand: []string{"/scripts/shred.sh", "2"},
						VolumeMode:          "Filesystem",
						FsType:              "ext4",
						NamePattern:         "nvm*,sdb*",
					},
				},
				UseAlphaAPI: true,
				MinResyncPeriod: metav1.Duration{
					Duration: time.Hour + time.Minute*30,
				},
			},
			nil,
		},
	}
	for _, v := range testcases {
		for name, value := range v.data {
			err1 := ioutil.WriteFile(filepath.Join(tmpConfigPath, name), []byte(value), 0644)
			if err1 != nil {
				t.Fatalf("Failed to write data into directory %s", tmpConfigPath)
			}
		}
		provisionerConfig := ProvisionerConfiguration{}
		err = LoadProvisionerConfigs(tmpConfigPath, &provisionerConfig)
		if !reflect.DeepEqual(err, v.expectedErr) {
			t.Errorf("LoadProvisionerConfigs error: expected %v, got %v", v.expectedErr, err)
		}
		if !reflect.DeepEqual(provisionerConfig, v.expected) {
			t.Errorf("Failed to parse config from data %q, expected %+v, got %+v", v.data, v.expected, provisionerConfig)
		}
	}
}

func TestVolumeConfigToConfigMapData(t *testing.T) {
	testcases := []struct {
		provisionerConfig *ProvisionerConfiguration
		expected          map[string]string
		expectedErr       error
	}{
		{
			provisionerConfig: &ProvisionerConfiguration{},
			expected: map[string]string{
				"storageClassMap": "null\n",
				"useAlphaAPI":     "false\n",
			},
			expectedErr: nil,
		},
		{
			provisionerConfig: &ProvisionerConfiguration{
				NodeLabelsForPV: []string{"node1", "node2"},
			},
			expected: map[string]string{
				"storageClassMap": "null\n",
				"nodeLabelsForPV": "- node1\n- node2\n",
				"useAlphaAPI":     "false\n",
			},
			expectedErr: nil,
		},
	}
	for _, test := range testcases {
		mapData, err := VolumeConfigToConfigMapData(test.provisionerConfig)
		if !reflect.DeepEqual(mapData, test.expected) || !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("ProvisionerConfig: %v, expected mapData: %v, got mapData: %v, expected err: %v, got err: %v", test.provisionerConfig, test.expected, mapData, test.expectedErr, err)
		}
	}
}

func TestCreateLocalPVSpec(t *testing.T) {
	volumeModeBlock := v1.PersistentVolumeBlock
	ownerReference := &metav1.OwnerReference{}
	testcases := []struct {
		config   *LocalPVConfig
		expected *v1.PersistentVolume
	}{
		{
			config: &LocalPVConfig{
				VolumeMode: volumeModeBlock,
			},
			expected: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnProvisionedBy: "",
					},
				},
				Spec: v1.PersistentVolumeSpec{
					Capacity: v1.ResourceList{
						v1.ResourceName(v1.ResourceStorage): *resource.NewQuantity(int64(0), resource.BinarySI),
					},
					PersistentVolumeSource: v1.PersistentVolumeSource{
						Local: &v1.LocalVolumeSource{},
					},
					AccessModes: []v1.PersistentVolumeAccessMode{
						v1.ReadWriteOnce,
					},
					VolumeMode: &volumeModeBlock,
				},
			},
		},
		{
			config: &LocalPVConfig{
				VolumeMode:     volumeModeBlock,
				UseAlphaAPI:    true,
				SetPVOwnerRef:  true,
				OwnerReference: ownerReference,
			},
			expected: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						AnnProvisionedBy:                   "",
						AlphaStorageNodeAffinityAnnotation: "",
					},
					OwnerReferences: []metav1.OwnerReference{*ownerReference},
				},
				Spec: v1.PersistentVolumeSpec{
					Capacity: v1.ResourceList{
						v1.ResourceName(v1.ResourceStorage): *resource.NewQuantity(int64(0), resource.BinarySI),
					},
					PersistentVolumeSource: v1.PersistentVolumeSource{
						Local: &v1.LocalVolumeSource{},
					},
					AccessModes: []v1.PersistentVolumeAccessMode{
						v1.ReadWriteOnce,
					},
					VolumeMode: &volumeModeBlock,
				},
			},
		},
	}
	for _, test := range testcases {
		pv := CreateLocalPVSpec(test.config)
		if !reflect.DeepEqual(pv, test.expected) {
			t.Errorf("LocalPVConfig: %v, expected PV spec: %v, got PV spec: %v", test.config, test.expected, pv)
		}
	}
}

func TestGetContainerPath(t *testing.T) {
	testcases := []struct {
		pv          *v1.PersistentVolume
		config      MountConfig
		expected    string
		expectedErr error
	}{
		{
			pv: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						Local: &v1.LocalVolumeSource{
							Path: "/mnt/disks",
						},
					},
				},
			},
			config: MountConfig{
				HostDir: "/mnt/disks",
			},
			expected:    ".",
			expectedErr: nil,
		},
		{
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "wrongpath",
				},
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						Local: &v1.LocalVolumeSource{
							Path: "wrongpath",
						},
					},
				},
			},
			config: MountConfig{
				HostDir: "/mnt/disks",
			},
			expected:    "",
			expectedErr: fmt.Errorf("Could not get relative path for pv %q: %v", "wrongpath", errors.New("Rel: can't make wrongpath relative to /mnt/disks")),
		},
	}
	for _, test := range testcases {
		path, err := GetContainerPath(test.pv, test.config)
		if !reflect.DeepEqual(path, test.expected) || !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("pv: %v, config: %v, expected path: %v, got path: %v, expected error: %v, got error: %v", test.pv, test.config, test.expected, path, test.expectedErr, err)
		}
	}
}

func TestGenerateMountName(t *testing.T) {
	testcases := []struct {
		mountConfig *MountConfig
	}{
		{
			mountConfig: &MountConfig{
				HostDir:  "/mnt/disks",
				MountDir: "/mnt/disks",
			},
		},
	}
	for _, test := range testcases {
		mountName := GenerateMountName(test.mountConfig)
		if !strings.HasPrefix(mountName, "mount-") {
			t.Errorf("mountConfig: %v, got mountName: %v", test.mountConfig, mountName)
		}
	}
}

func TestGetVolumeMode(t *testing.T) {
	fakeDirFile := "/file"
	fakeDirBlock := "/block"
	fakeDirFiles := map[string][]*util.FakeDirEntry{
		fakeDirFile: {&util.FakeDirEntry{
			Name:       "disks",
			VolumeType: "file",
			Capacity:   64,
		}},
		fakeDirBlock: {&util.FakeDirEntry{
			Name:       "disks",
			VolumeType: "block",
			Capacity:   64,
		}},
	}
	fakeVolUtil := util.NewFakeVolumeUtil(true, fakeDirFiles)
	testcases := []struct {
		volumeUtil  util.VolumeUtil
		fullPath    string
		expected    v1.PersistentVolumeMode
		expectedErr error
	}{
		{
			volumeUtil:  fakeVolUtil,
			fullPath:    "/file/disks",
			expected:    v1.PersistentVolumeFilesystem,
			expectedErr: nil,
		},
		{
			volumeUtil:  fakeVolUtil,
			fullPath:    "/block/disks",
			expected:    v1.PersistentVolumeBlock,
			expectedErr: nil,
		},
		{
			volumeUtil:  fakeVolUtil,
			fullPath:    "/file/disk",
			expected:    "",
			expectedErr: fmt.Errorf("Directory check for %q failed: %s", "/file/disk", fmt.Errorf("Directory entry %q not found", "/file/disk")),
		},
	}
	for _, test := range testcases {
		mode, err := GetVolumeMode(test.volumeUtil, test.fullPath)
		if !reflect.DeepEqual(mode, test.expected) || !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("volumeUtil: %v, fullPath: %v, expected mode: %v, got mode: %v, expected error: %v, got error: %v", test.volumeUtil, test.fullPath, test.expected, mode, test.expectedErr, err)
		}
	}
}

func TestNodeExists(t *testing.T) {
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				NodeLabelKey: "test-node",
			},
		},
	}

	tests := []struct {
		nodeAdded *v1.Node
		// Required.
		nodeQueried    *v1.Node
		expectedResult bool
	}{
		{
			nodeAdded:      node,
			nodeQueried:    node,
			expectedResult: true,
		},
		{
			nodeQueried:    node,
			expectedResult: false,
		},
	}

	for _, test := range tests {
		client := fake.NewSimpleClientset()
		informers := informers.NewSharedInformerFactory(client, time.Duration(0))
		nodeInformer := informers.Core().V1().Nodes()

		if test.nodeAdded != nil {
			nodeInformer.Informer().GetStore().Add(test.nodeAdded)
		}

		exists, err := NodeExists(nodeInformer.Lister(), test.nodeQueried.Labels[NodeLabelKey])
		if err != nil {
			t.Errorf("Got unexpected error: %s", err.Error())
		}
		if exists != test.expectedResult {
			t.Errorf("expected result: %t, actual: %t", test.expectedResult, exists)
		}
	}
}

func TestNodeAttachedToLocalPV(t *testing.T) {
	nodeName := "testNodeName"

	tests := []struct {
		name             string
		pv               *v1.PersistentVolume
		expectedNodeName string
		expectedStatus   bool
	}{
		{
			name:             "NodeAffinity will all necessary fields",
			pv:               withNodeAffinity(pv(), []string{nodeName}, NodeLabelKey),
			expectedNodeName: nodeName,
			expectedStatus:   true,
		},
		{
			name:             "empty nodeNames array",
			pv:               withNodeAffinity(pv(), []string{}, NodeLabelKey),
			expectedNodeName: "",
			expectedStatus:   false,
		},
		{
			name:             "multiple nodeNames",
			pv:               withNodeAffinity(pv(), []string{nodeName, "newNode"}, NodeLabelKey),
			expectedNodeName: "",
			expectedStatus:   false,
		},
		{
			name:             "wrong node label key",
			pv:               withNodeAffinity(pv(), []string{nodeName}, "wrongLabel"),
			expectedNodeName: "",
			expectedStatus:   false,
		},
	}

	for _, test := range tests {
		nodeName, ok := NodeAttachedToLocalPV(test.pv)
		if ok != test.expectedStatus {
			t.Errorf("test: %s, status: %t, expectedStaus: %t", test.name, ok, test.expectedStatus)
		}
		if nodeName != test.expectedNodeName {
			t.Errorf("test: %s, nodeName: %s, expectedNodeName: %s", test.name, nodeName, test.expectedNodeName)
		}
	}
}

func TestIsLocalPVWithStorageClass(t *testing.T) {
	tests := []struct {
		name              string
		pv                *v1.PersistentVolume
		storageClassNames []string
		expected          bool
	}{
		{
			name: "local PV with matching StorageClass",
			pv: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{Local: &v1.LocalVolumeSource{}},
					StorageClassName:       "testStorageClassName",
				},
			},
			storageClassNames: []string{"testStorageClassName"},
			expected:          true,
		},
		{
			name: "local PV with matching StorageClass + multiple storageClassNames",
			pv: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{Local: &v1.LocalVolumeSource{}},
					StorageClassName:       "testStorageClassName",
				},
			},
			storageClassNames: []string{"testStorageClassName", "alternative"},
			expected:          true,
		},
		{
			name: "local PV without matching StorageClass",
			pv: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{Local: &v1.LocalVolumeSource{}},
					StorageClassName:       "wrongName",
				},
			},
			storageClassNames: []string{"testStorageClassName"},
			expected:          false,
		},
		{
			name: "local PV  + empty storageClassNames",
			pv: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{Local: &v1.LocalVolumeSource{}},
					StorageClassName:       "testStorageClassName",
				},
			},
			storageClassNames: []string{},
			expected:          false,
		},
		{
			name: "non-local PV with matching StorageClass",
			pv: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{},
					StorageClassName:       "testStorageClassName",
				},
			},
			storageClassNames: []string{"testStorageClassName"},
			expected:          false,
		},
	}

	for _, test := range tests {
		result := IsLocalPVWithStorageClass(test.pv, test.storageClassNames)
		if result != test.expected {
			t.Errorf("name: %s, pv: %v, storageClassName: %s, expected result: %t, actual: %t", test.name, test.pv, test.storageClassNames, test.expected, result)
		}
	}
}

func pv() *v1.PersistentVolume {
	return &v1.PersistentVolume{
		Spec: v1.PersistentVolumeSpec{},
	}
}

func withNodeAffinity(pv *v1.PersistentVolume, nodeNames []string, nodeLabelKey string) *v1.PersistentVolume {
	pv.Spec.NodeAffinity = &v1.VolumeNodeAffinity{
		Required: &v1.NodeSelector{
			NodeSelectorTerms: []v1.NodeSelectorTerm{
				{
					MatchExpressions: []v1.NodeSelectorRequirement{
						{
							Key:      nodeLabelKey,
							Operator: v1.NodeSelectorOpIn,
							Values:   nodeNames,
						},
					},
				},
			},
		},
	}
	return pv
}

func removeAllNodeNames(pv *v1.PersistentVolume) *v1.PersistentVolume {
	pv.Spec.NodeAffinity.Required.NodeSelectorTerms[0].MatchExpressions[0].Values = []string{}
	return pv
}
