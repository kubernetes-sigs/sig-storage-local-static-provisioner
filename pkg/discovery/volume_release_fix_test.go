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

package discovery

import (
	"context"
	"fmt"
	"testing"

	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/cache"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/util"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestDiscoverVolumes_ReleasedPVWithDelete tests that a Released PV with Delete reclaim policy
// is properly handled by creating a new Available PV for the same volume path.
// This test validates the fix for the issue where pods remain pending after local volume release.
func TestDiscoverVolumes_ReleasedPVWithDelete(t *testing.T) {
	// Setup directory layout for a single volume
	vols := map[string][]*util.FakeDirEntry{
		"dir1": {
			{Name: "mount1", Hash: 0xaaaafef5, VolumeType: util.FakeEntryFile, Capacity: 100 * 1024},
		},
	}

	test := &testConfig{
		dirLayout:       vols,
		expectedVolumes: vols, // We expect the volume to be recreated
	}

	// Create a Released PV with Delete reclaim policy and add it to cache
	volumeCache := cache.NewVolumeCache()
	releasedPV := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: generatePVName("mount1", testNodeName, "sc1"),
		},
		Spec: v1.PersistentVolumeSpec{
			Capacity: v1.ResourceList{
				v1.ResourceStorage: *resource.NewQuantity(102400, resource.BinarySI),
			},
			AccessModes:                   []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimDelete,
			StorageClassName:              "sc1",
			PersistentVolumeSource: v1.PersistentVolumeSource{
				Local: &v1.LocalVolumeSource{
					Path: fmt.Sprintf("%s/dir1/mount1", testHostDir),
				},
			},
		},
		Status: v1.PersistentVolumeStatus{
			Phase: v1.VolumeReleased, // This is the key - PV is Released
		},
	}
	volumeCache.AddPV(releasedPV)
	test.cache = volumeCache

	// Setup discoverer with the pre-existing Released PV
	d := testSetup(t, test, false, false)

	// Add the Released PV to the fake client so it can be deleted
	_, err := test.client.CoreV1().PersistentVolumes().Create(context.TODO(), releasedPV, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create Released PV in fake client: %v", err)
	}

	// Before discovery: verify the Released PV exists in cache
	_, exists := test.cache.GetPV(releasedPV.Name)
	if !exists {
		t.Fatalf("Released PV should exist in cache before discovery")
	}

	// Run discovery
	d.DiscoverLocalVolumes()

	// After discovery: verify a PV exists (the old one should have been deleted and a new one created)
	newPV, exists := test.cache.GetPV(releasedPV.Name)
	if !exists {
		t.Fatalf("PV should exist in cache after discovery")
	}

	// The new PV should not be in Released state
	if newPV.Status.Phase == v1.VolumeReleased {
		t.Errorf("Expected PV to not be in Released state after discovery, but it is still Released")
	}

	// Verify the PV has the correct reclaim policy and other properties
	if newPV.Spec.PersistentVolumeReclaimPolicy != v1.PersistentVolumeReclaimDelete {
		t.Errorf("Expected PV reclaim policy to be Delete, got %v", newPV.Spec.PersistentVolumeReclaimPolicy)
	}

	if newPV.Spec.StorageClassName != "sc1" {
		t.Errorf("Expected PV storage class to be sc1, got %s", newPV.Spec.StorageClassName)
	}

	expectedPath := fmt.Sprintf("%s/dir1/mount1", testHostDir)
	if newPV.Spec.Local == nil || newPV.Spec.Local.Path != expectedPath {
		t.Errorf("Expected PV local path to be %s, got %v", expectedPath, newPV.Spec.Local)
	}
}

// TestDiscoverVolumes_ReleasedPVWithRetain tests that a Released PV with Retain reclaim policy
// is NOT replaced by discovery (existing behavior should be preserved).
func TestDiscoverVolumes_ReleasedPVWithRetain(t *testing.T) {
	// Setup directory layout for a single volume
	vols := map[string][]*util.FakeDirEntry{
		"dir1": {
			{Name: "mount1", Hash: 0xaaaafef5, VolumeType: util.FakeEntryFile, Capacity: 100 * 1024},
		},
	}

	test := &testConfig{
		dirLayout:       vols,
		expectedVolumes: map[string][]*util.FakeDirEntry{}, // No new volumes should be created
	}

	// Create a Released PV with Retain reclaim policy and add it to cache
	volumeCache := cache.NewVolumeCache()
	releasedPV := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: generatePVName("mount1", testNodeName, "sc1"),
		},
		Spec: v1.PersistentVolumeSpec{
			Capacity: v1.ResourceList{
				v1.ResourceStorage: *resource.NewQuantity(102400, resource.BinarySI),
			},
			AccessModes:                   []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimRetain, // Retain policy
			StorageClassName:              "sc1",
			PersistentVolumeSource: v1.PersistentVolumeSource{
				Local: &v1.LocalVolumeSource{
					Path: fmt.Sprintf("%s/dir1/mount1", testHostDir),
				},
			},
		},
		Status: v1.PersistentVolumeStatus{
			Phase: v1.VolumeReleased,
		},
	}
	volumeCache.AddPV(releasedPV)
	test.cache = volumeCache

	// Setup discoverer with the pre-existing Released PV
	d := testSetup(t, test, false, false)

	// Before discovery: verify the Released PV exists in cache
	originalPV, exists := test.cache.GetPV(releasedPV.Name)
	if !exists {
		t.Fatalf("Released PV should exist in cache before discovery")
	}

	// Run discovery
	d.DiscoverLocalVolumes()

	// After discovery: verify the Released PV with Retain policy is unchanged
	currentPV, exists := test.cache.GetPV(releasedPV.Name)
	if !exists {
		t.Fatalf("PV should still exist in cache after discovery")
	}

	// Verify the PV is still Released (unchanged behavior for Retain policy)
	if currentPV.Status.Phase != v1.VolumeReleased {
		t.Errorf("Expected PV to remain in Released state for Retain policy, got %v", currentPV.Status.Phase)
	}

	// Verify it's still the same PV object
	if currentPV.Spec.PersistentVolumeReclaimPolicy != v1.PersistentVolumeReclaimRetain {
		t.Errorf("Expected PV reclaim policy to remain Retain, got %v", currentPV.Spec.PersistentVolumeReclaimPolicy)
	}

	// Verify the PV wasn't modified
	if originalPV.Name != currentPV.Name {
		t.Errorf("PV name should not change, original: %s, current: %s", originalPV.Name, currentPV.Name)
	}

	// Verify no new PVs were created
	verifyCreatedPVs(t, test)
}