/*
Copyright 2023 The Kubernetes Authors.

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

package deleter

import (
	"context"
	"reflect"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"

	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"
)

const (
	nonExistentNodeName         = "testNonExistentNodeName"
	testNodeName                = "testNodeName"
	testStorageClassName        = "testStorageClassName"
	testPVName                  = "testPVName"
	alternativeStorageClassName = "alternativeStorageClassName"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
	localSource        = v1.PersistentVolumeSource{Local: &v1.LocalVolumeSource{}}
	remoteSource       = v1.PersistentVolumeSource{CSI: &v1.CSIPersistentVolumeSource{}}
)

func TestDeleter(t *testing.T) {
	node := node()

	tests := []struct {
		name string
		// Objects to insert into fake kubeclient before the test starts.
		initialObjects []runtime.Object
		// PV object. This will automatically be added to initialObjects.
		pv *v1.PersistentVolume
		// Node object. This will automatically be added to initialObjects.
		node *v1.Node
		// Names of StorageClasses that the PV/PVC need to belong to to be cleaned up.
		storageClassNames []string
		expectedActions   []core.Action
	}{
		{
			name:              "released local pv with delete reclaim",
			pv:                localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimDelete, testStorageClassName),
			storageClassNames: []string{testStorageClassName},
			expectedActions: []core.Action{
				deletePVAction(localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimDelete, testStorageClassName)),
			},
		},
		{
			name:              "available local pv with delete reclaim",
			pv:                localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimDelete, testStorageClassName),
			storageClassNames: []string{testStorageClassName},
			expectedActions: []core.Action{
				deletePVAction(localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimDelete, testStorageClassName)),
			},
		},
		{
			name:              "available local pv with recycle reclaim",
			pv:                localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimRecycle, testStorageClassName),
			storageClassNames: []string{testStorageClassName},
			expectedActions: []core.Action{
				deletePVAction(localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimRecycle, testStorageClassName)),
			},
		},
		{
			name:              "available local pv with retain reclaim",
			pv:                localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimRetain, testStorageClassName),
			storageClassNames: []string{testStorageClassName},
			expectedActions: []core.Action{
				deletePVAction(localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimRetain, testStorageClassName)),
			},
		},
		{
			name:              "local pv has wrong storage class name",
			pv:                pvWithCustomStorageClass(localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimRetain, testStorageClassName)),
			storageClassNames: []string{testStorageClassName},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "pv is not a local pv",
			pv:                pvWithRemoteSource(localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimRetain, testStorageClassName)), // change source to be remote
			storageClassNames: []string{testStorageClassName},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "local pv has affinity to node that still exists",
			pv:                localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimRetain, testStorageClassName),
			storageClassNames: []string{testStorageClassName},
			node:              node,
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:            "empty",
			expectedActions: []core.Action{
				// Intentionally left empty
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create initial data for client
			if test.node != nil {
				test.initialObjects = append(test.initialObjects, test.node)
			}
			if test.pv != nil {
				test.initialObjects = append(test.initialObjects, test.pv)
			}

			// Create client with initial data
			client := fake.NewSimpleClientset(test.initialObjects...)

			informers := informers.NewSharedInformerFactory(client, noResyncPeriodFunc())
			pvInformer := informers.Core().V1().PersistentVolumes()
			nodeInformer := informers.Core().V1().Nodes()

			deleter := NewDeleter(client, pvInformer.Lister(), nodeInformer.Lister(), test.storageClassNames)

			// Populate the informers with initial objects so the controller can
			// Get() and List() it.
			for _, obj := range test.initialObjects {
				switch obj.(type) {
				case *v1.PersistentVolume:
					pvInformer.Informer().GetStore().Add(obj)
				case *v1.Node:
					nodeInformer.Informer().GetStore().Add(obj)
				default:
					t.Fatalf("Unknown initalObject type: %+v", obj)
				}
			}

			// Start test by simulating an event.
			deleter.DeletePVs(context.TODO())

			actions := client.Actions()
			for i, action := range actions {
				if len(test.expectedActions) < i+1 {
					t.Errorf("Test %q: %d unexpected actions: %+v", test.name, len(actions)-len(test.expectedActions), actions[i:])
					break
				}

				expectedAction := test.expectedActions[i]
				if !reflect.DeepEqual(expectedAction, action) {
					t.Errorf("Test %q: action %d\nExpected:\n%s\ngot:\n%s", test.name, i, expectedAction, action)
				}
			}

			if len(test.expectedActions) > len(actions) {
				t.Errorf("Test %q: %d additional expected actions", test.name, len(test.expectedActions)-len(actions))
				for _, a := range test.expectedActions[len(actions):] {
					t.Logf("additional action: %+v", a)
				}
			}
		})
	}
}

func pvWithRemoteSource(pv *v1.PersistentVolume) *v1.PersistentVolume {
	pv.Spec.PersistentVolumeSource = remoteSource
	return pv
}

func pvWithCustomStorageClass(pv *v1.PersistentVolume) *v1.PersistentVolume {
	pv.Spec.StorageClassName = alternativeStorageClassName
	return pv
}

func localPV(node *v1.Node, phase v1.PersistentVolumePhase, reclaimPolicy v1.PersistentVolumeReclaimPolicy, storageClassName string) *v1.PersistentVolume {
	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: testPVName,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: localSource,
			NodeAffinity: &v1.VolumeNodeAffinity{
				Required: &v1.NodeSelector{
					NodeSelectorTerms: []v1.NodeSelectorTerm{
						{
							MatchExpressions: []v1.NodeSelectorRequirement{
								{
									Key:      common.NodeLabelKey,
									Operator: v1.NodeSelectorOpIn,
									Values:   []string{node.Labels[common.NodeLabelKey]},
								},
							},
						},
					},
				},
			},
			PersistentVolumeReclaimPolicy: reclaimPolicy,
			StorageClassName:              testStorageClassName,
		},
		Status: v1.PersistentVolumeStatus{
			Phase: phase,
		},
	}
}

func node() *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{common.NodeLabelKey: testNodeName},
		},
	}
}

func deletePVAction(pv *v1.PersistentVolume) core.DeleteActionImpl {
	return core.NewDeleteAction(schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumes"}, pv.Namespace, pv.Name)
}
