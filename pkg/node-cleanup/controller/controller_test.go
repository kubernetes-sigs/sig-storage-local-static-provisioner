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

package controller

import (
	"context"
	"reflect"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/klog/v2"

	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"
)

const (
	defaultPVName               = "defaultPV"
	nonExistentNodeName         = "non-existent-node"
	testStorageClassName        = "test-storageclass"
	defaultNodeName             = "defaultNode"
	defaultPVCName              = "defaultPVC"
	defaultNamespace            = "default"
	alternativeStorageClassName = "alternative-storageclass"
	defaultPVCUID               = "123"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
	localSource        = v1.PersistentVolumeSource{Local: &v1.LocalVolumeSource{}}
	remoteSource       = v1.PersistentVolumeSource{
		CSI: &v1.CSIPersistentVolumeSource{},
	}
)

func TestCleanupController(t *testing.T) {
	node := node()
	pvc := pvc()

	tests := []struct {
		name string
		// Objects to insert into fake kubeclient before the test starts.
		initialObjects []runtime.Object
		// PV object. This will automatically be added to initialObjects.
		pv *v1.PersistentVolume
		// PVC object. This will automatically be added to initialObjects.
		pvc *v1.PersistentVolumeClaim
		// Node object. This will automatically be added to initialObjects.
		node *v1.Node
		// Names of StorageClasses that the PV/PVC need to belong to to be cleaned up.
		storageClassNames []string
		expectedActions   []core.Action
		// Whether the give node exists when the controller starts
		nodeIsInitialObject bool
		// Whether to bring up the given node in the middle of the test
		bringBackNode bool
	}{
		{
			name:              "pv with affinity to deleted node + node still deleted -> delete pvc",
			pv:                pvWithPVCAndNode(pvc, node),
			pvc:               pvc,
			storageClassNames: []string{testStorageClassName},
			expectedActions: []core.Action{
				deletePVCAction(pvc),
			},
		},
		{
			name:              "pv with affinity to deleted node + pv references pvc but pvc doesn't reference pv -> don't delete pvc",
			pv:                pvWithPVCAndNode(pvc, node),
			pvc:               pvcWithVolumeName("different-volume"),
			storageClassNames: []string{testStorageClassName},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "pv with affinity to deleted node and node still deleted + pvc uid changed -> don't delete pvc",
			pv:                pvWithPVCAndNode(pvc, node),
			pvc:               pvcWithUID("randomUID"),
			storageClassNames: []string{testStorageClassName},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "multiple storageclass names + pv with affinity to deleted node -> delete pvc",
			pv:                pvWithPVCAndNode(pvc, node),
			pvc:               pvc,
			storageClassNames: []string{testStorageClassName, alternativeStorageClassName},
			expectedActions: []core.Action{
				deletePVCAction(pvc),
			},
		},
		{
			name:              "no storageclass names + pv with affinity to deleted node -> don't delete pvc",
			pv:                pvWithPVCAndNode(pvc, node),
			pvc:               pvc,
			storageClassNames: []string{},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "pv with affinity to deleted node + node brought back up -> don't delete pvc",
			pv:                pvWithPVCAndNode(pvc, node),
			pvc:               pvc,
			node:              node,
			storageClassNames: []string{testStorageClassName},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
			bringBackNode: true,
		},
		{
			name:              "pv with affinity to node that exists -> don't delete pvc",
			pv:                pvWithPVCAndNode(pvc, node),
			pvc:               pvc,
			node:              node,
			storageClassNames: []string{testStorageClassName},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
			nodeIsInitialObject: true,
		},
		{
			name:              "remote PV with affinity to deleted node -> don't delete pvc",
			pv:                pvWithRemoteSource(pvWithPVCAndNode(pvc, node)),
			pvc:               pvc,
			node:              node,
			storageClassNames: []string{testStorageClassName},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "PV with wrong storageclass + affinity to deleted node -> don't delete pvc",
			pv:                pvWithPVCAndNode(pvc, node),
			pvc:               pvc,
			node:              node,
			storageClassNames: []string{alternativeStorageClassName},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "PV with affinity to deleted node + PVC already deleted -> don't try to delete PVC",
			pv:                pvWithPVCAndNode(pvc, node),
			storageClassNames: []string{testStorageClassName},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "PV without PVC -> do nothing",
			pv:                pvWithNode(node),
			storageClassNames: []string{testStorageClassName},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// Create initial data for client
			if test.nodeIsInitialObject && test.node != nil {
				test.initialObjects = append(test.initialObjects, test.node)
			}
			if test.pv != nil {
				test.initialObjects = append(test.initialObjects, test.pv)
			}
			if test.pvc != nil {
				test.initialObjects = append(test.initialObjects, test.pvc)
			}

			// Create client with initial data
			client := fake.NewSimpleClientset(test.initialObjects...)

			informers := informers.NewSharedInformerFactory(client, noResyncPeriodFunc())
			pvInformer := informers.Core().V1().PersistentVolumes()
			pvcInformer := informers.Core().V1().PersistentVolumeClaims()
			nodeInformer := informers.Core().V1().Nodes()

			// Set delay for entryQueue. Delay only needed when the test
			// adds back a deleted Node before the processing deadline.
			var queueDelay time.Duration
			if test.bringBackNode {
				queueDelay = 1 * time.Second
			} else {
				queueDelay = time.Duration(0)
			}

			ctrl := NewCleanupController(client, pvInformer, pvcInformer, nodeInformer, test.storageClassNames, queueDelay, time.Duration(0))

			// Populate the informers with initial objects so the controller can
			// Get() and List() it.
			for _, obj := range test.initialObjects {
				switch obj.(type) {
				case *v1.PersistentVolume:
					pvInformer.Informer().GetStore().Add(obj)
				case *v1.Node:
					nodeInformer.Informer().GetStore().Add(obj)
				case *v1.PersistentVolumeClaim:
					pvcInformer.Informer().GetStore().Add(obj)
				default:
					t.Fatalf("Unknown initalObject type: %+v", obj)
				}
			}

			// Start test by simulating an event
			ctrl.nodeDeleted(struct{}{})

			if test.bringBackNode && test.node != nil {
				nodeInformer.Informer().GetStore().Add(test.node)
			}

			// Process the controller queue until we get expected results
			timeout := time.Now().Add(10 * time.Second)
			queueWaitPeriod := time.Now().Add(queueDelay + 1*time.Second)
			lastReportedActionCount := 0
			for {
				if time.Now().After(timeout) {
					t.Errorf("Test %q: timed out", test.name)
					break
				}
				if ctrl.pvQueue.Len() > 0 {
					klog.V(5).Infof("Test %q: %d events queue, processing one", test.name, ctrl.pvQueue.Len())
					ctrl.processNextWorkItem(context.TODO())
				}
				if ctrl.pvQueue.Len() > 0 {
					// There is still some work in the queue, process it now
					continue
				}
				currentActionCount := len(client.Actions())
				if currentActionCount < len(test.expectedActions) {
					// Do not log every wait, only when the action count changes.
					if lastReportedActionCount < currentActionCount {
						klog.V(5).Infof("Test %q: got %d actions out of %d, waiting for the rest", test.name, currentActionCount, len(test.expectedActions))
						lastReportedActionCount = currentActionCount
					}
					// The test expected more to happen, wait for the actions.
					time.Sleep(10 * time.Millisecond)
					continue
				}
				if !time.Now().After(queueWaitPeriod) {
					// Since the queues are delayed, we give them a chance to
					// add their items.
					continue
				}
				break
			}

			actions := client.Actions()
			for i, action := range actions {
				print(action.GetVerb())
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
					t.Logf("    %+v", a)
				}
			}
		})
	}
}

func pv() *v1.PersistentVolume {
	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultPVName,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: localSource,
			StorageClassName:       testStorageClassName,
		},
	}
}

func pvWithRemoteSource(pv *v1.PersistentVolume) *v1.PersistentVolume {
	pv.Spec.PersistentVolumeSource = remoteSource
	return pv
}

func pvWithPVCAndNode(pvc *v1.PersistentVolumeClaim, node *v1.Node) *v1.PersistentVolume {
	pv := pv()
	pv.Spec.ClaimRef = &v1.ObjectReference{
		Kind:      "PersistentVolumeClaim",
		Name:      pvc.Name,
		Namespace: pvc.Namespace,
		UID:       pvc.UID,
	}
	pv.Spec.NodeAffinity = &v1.VolumeNodeAffinity{
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
	}
	pvc.Spec.VolumeName = pv.Name
	return pv
}

func pvWithNode(node *v1.Node) *v1.PersistentVolume {
	pv := pv()
	pv.Spec.NodeAffinity = &v1.VolumeNodeAffinity{
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
	}
	return pv
}

func pvc() *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultPVCName,
			Namespace: defaultNamespace,
			UID:       defaultPVCUID,
		},
	}
}

func pvcWithVolumeName(volumeName string) *v1.PersistentVolumeClaim {
	pvc := pvc()
	pvc.Spec.VolumeName = volumeName
	return pvc
}

func pvcWithUID(uid string) *v1.PersistentVolumeClaim {
	pvc := pvc()
	pvc.UID = types.UID(uid)
	return pvc
}

func node() *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{common.NodeLabelKey: defaultNodeName},
		},
	}
}

func deletePVCAction(pvc *v1.PersistentVolumeClaim) core.DeleteActionImpl {
	return core.NewDeleteActionWithOptions(schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}, pvc.Namespace, pvc.Name, metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &pvc.UID}})
}
