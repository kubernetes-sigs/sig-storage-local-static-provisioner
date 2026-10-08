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
	testCSIDriver               = "lis.csi.aws.com"
	azureCSIDriver              = "localdisk.csi.acstor.io"
	customCSIDriver             = "custom.csi.example.com"
	customTopologyKey           = "topology.example.com/node"
	azureSelectedNodeAnnotation = "localdisk.csi.acstor.io/selected-node"
	azureInitialNodeAttribute   = "localdisk.csi.acstor.io/selected-initial-node"
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
		// Value of the --csi-drivers flag: CSI drivers whose PVs are eligible for cleanup (in addition to local PVs).
		csiDrivers      []string
		expectedActions []core.Action
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
			name:              "released CSI pv with allowlisted driver and delete reclaim",
			pv:                pvWithCSIDriver(localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimDelete, testStorageClassName), testCSIDriver),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{testCSIDriver},
			expectedActions: []core.Action{
				deletePVAction(pvWithCSIDriver(localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimDelete, testStorageClassName), testCSIDriver)),
			},
		},
		{
			name:              "available CSI pv with allowlisted driver",
			pv:                pvWithCSIDriver(localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimDelete, testStorageClassName), testCSIDriver),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{testCSIDriver},
			expectedActions: []core.Action{
				deletePVAction(pvWithCSIDriver(localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimDelete, testStorageClassName), testCSIDriver)),
			},
		},
		{
			name:              "bound CSI pv with allowlisted driver is never deleted",
			pv:                pvWithCSIDriver(localPV(node, v1.VolumeBound, v1.PersistentVolumeReclaimDelete, testStorageClassName), testCSIDriver),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{testCSIDriver},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "released CSI pv with retain reclaim is not deleted",
			pv:                pvWithCSIDriver(localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimRetain, testStorageClassName), testCSIDriver),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{testCSIDriver},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "CSI pv with driver that is not allowlisted",
			pv:                pvWithCSIDriver(localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimDelete, testStorageClassName), "ebs.csi.aws.com"),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{testCSIDriver},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "CSI pv with allowlisted driver but wrong storage class",
			pv:                pvWithCSIDriver(pvWithCustomStorageClass(localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimDelete, testStorageClassName)), testCSIDriver),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{testCSIDriver},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "CSI pv with listed storage class but no csi drivers configured",
			pv:                pvWithCSIDriver(localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimDelete, testStorageClassName), testCSIDriver),
			storageClassNames: []string{testStorageClassName},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "CSI pv with allowlisted driver has affinity to node that still exists",
			pv:                pvWithCSIDriver(localPV(node, v1.VolumeAvailable, v1.PersistentVolumeReclaimDelete, testStorageClassName), testCSIDriver),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{testCSIDriver},
			node:              node,
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "released azure pv (no node affinity) whose initial node is deleted",
			pv:                pvWithAzureNodeRefs(localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimDelete, testStorageClassName), node.Name, ""),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{azureCSIDriver},
			expectedActions: []core.Action{
				deletePVAction(localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimDelete, testStorageClassName)),
			},
		},
		{
			name:              "released azure pv whose initial node is deleted but selected-node annotation points to existing node",
			pv:                pvWithAzureNodeRefs(localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimDelete, testStorageClassName), nonExistentNodeName, node.Name),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{azureCSIDriver},
			node:              node,
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "released azure pv without any node reference",
			pv:                pvWithAzureNodeRefs(localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimDelete, testStorageClassName), "", ""),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{azureCSIDriver},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "bound azure pv is never deleted",
			pv:                pvWithAzureNodeRefs(localPV(node, v1.VolumeBound, v1.PersistentVolumeReclaimDelete, testStorageClassName), node.Name, ""),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{azureCSIDriver},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "released azure pv but azure driver not allowlisted",
			pv:                pvWithAzureNodeRefs(localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimDelete, testStorageClassName), node.Name, ""),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{testCSIDriver},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "custom affinity key: live Node has the label value under a different name",
			pv:                pvWithCustomAffinity(localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimDelete, testStorageClassName), customCSIDriver, customTopologyKey, "n9"),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{customCSIDriver + "=affinity:" + customTopologyKey},
			node:              nodeWithLabel("ip-10-0-0-1", customTopologyKey, "n9"),
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "custom affinity key: no Node has the label value",
			pv:                pvWithCustomAffinity(localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimDelete, testStorageClassName), customCSIDriver, customTopologyKey, "n9"),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{customCSIDriver + "=affinity:" + customTopologyKey},
			expectedActions: []core.Action{
				deletePVAction(localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimDelete, testStorageClassName)),
			},
		},
		{
			name:              "CSI pv whose node affinity has an extra term without the hostname key",
			pv:                pvWithExtraZoneTerm(pvWithCSIDriver(localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimDelete, testStorageClassName), testCSIDriver)),
			storageClassNames: []string{testStorageClassName},
			csiDrivers:        []string{testCSIDriver},
			expectedActions:   []core.Action{
				// Intentionally left empty
			},
		},
		{
			name:              "local pv whose node affinity has an extra term without the hostname key",
			pv:                pvWithExtraZoneTerm(localPV(node, v1.VolumeReleased, v1.PersistentVolumeReclaimDelete, testStorageClassName)),
			storageClassNames: []string{testStorageClassName},
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

			csiDrivers, err := common.ParseCSIDrivers(test.csiDrivers)
			if err != nil {
				t.Fatalf("invalid csiDrivers %v: %v", test.csiDrivers, err)
			}

			deleter := NewDeleter(client, pvInformer.Lister(), nodeInformer.Lister(), test.storageClassNames, csiDrivers)

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

func pvWithCSIDriver(pv *v1.PersistentVolume, driver string) *v1.PersistentVolume {
	pv.Spec.PersistentVolumeSource = v1.PersistentVolumeSource{CSI: &v1.CSIPersistentVolumeSource{Driver: driver}}
	return pv
}

// pvWithAzureNodeRefs turns the PV into one shaped like a localdisk.csi.acstor.io PV:
// no node affinity, the creating node in a volume attribute and, only after a failover,
// the owning node in the selected-node annotation. Empty node names are left out.
func pvWithAzureNodeRefs(pv *v1.PersistentVolume, initialNode, selectedNode string) *v1.PersistentVolume {
	pv.Spec.NodeAffinity = nil
	attributes := map[string]string{}
	if initialNode != "" {
		attributes[azureInitialNodeAttribute] = initialNode
	}
	pv.Spec.PersistentVolumeSource = v1.PersistentVolumeSource{CSI: &v1.CSIPersistentVolumeSource{Driver: azureCSIDriver, VolumeAttributes: attributes}}
	if selectedNode != "" {
		pv.Annotations = map[string]string{azureSelectedNodeAnnotation: selectedNode}
	}
	return pv
}

// pvWithCustomAffinity makes the PV a CSI PV of the given driver whose node affinity uses
// a custom topology key instead of kubernetes.io/hostname.
func pvWithCustomAffinity(pv *v1.PersistentVolume, driver, key, value string) *v1.PersistentVolume {
	pv.Spec.PersistentVolumeSource = v1.PersistentVolumeSource{CSI: &v1.CSIPersistentVolumeSource{Driver: driver}}
	pv.Spec.NodeAffinity.Required.NodeSelectorTerms[0].MatchExpressions[0].Key = key
	pv.Spec.NodeAffinity.Required.NodeSelectorTerms[0].MatchExpressions[0].Values = []string{value}
	return pv
}

// pvWithExtraZoneTerm adds a second (ORed) node selector term that has no hostname expression.
func pvWithExtraZoneTerm(pv *v1.PersistentVolume) *v1.PersistentVolume {
	pv.Spec.NodeAffinity.Required.NodeSelectorTerms = append(pv.Spec.NodeAffinity.Required.NodeSelectorTerms, v1.NodeSelectorTerm{
		MatchExpressions: []v1.NodeSelectorRequirement{{Key: "topology.kubernetes.io/zone", Operator: v1.NodeSelectorOpIn, Values: []string{"zone-a"}}},
	})
	return pv
}

func nodeWithLabel(name, key, value string) *v1.Node {
	return &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{key: value}}}
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
									Values:   []string{node.Name},
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
			Name: testNodeName,
		},
	}
}

func deletePVAction(pv *v1.PersistentVolume) core.DeleteActionImpl {
	return core.NewDeleteAction(schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumes"}, pv.Namespace, pv.Name)
}
