/*
Copyright 2026 The Kubernetes Authors.

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
	"reflect"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	awsDriver   = "lis.csi.aws.com"
	azureDriver = "localdisk.csi.acstor.io"
	azureSel    = "localdisk.csi.acstor.io/selected-node"
	azureInit   = "localdisk.csi.acstor.io/selected-initial-node"
)

func TestParseCSIDrivers(t *testing.T) {
	hostnameRefs := []NodeRef{{Kind: NodeRefAffinity, Key: v1.LabelHostname}}
	azureRefs := []NodeRef{
		{Kind: NodeRefAnnotation, Key: azureSel},
		{Kind: NodeRefAttribute, Key: azureInit},
	}

	tests := []struct {
		name        string
		entries     []string
		expected    CSIDrivers
		expectedErr string
	}{
		{name: "nil", entries: nil, expected: CSIDrivers{}},
		{name: "empty", entries: []string{}, expected: CSIDrivers{}},
		{
			name:     "unknown driver defaults to hostname node affinity",
			entries:  []string{awsDriver},
			expected: CSIDrivers{awsDriver: hostnameRefs},
		},
		{
			name:     "well-known azure driver uses its built-in references",
			entries:  []string{azureDriver},
			expected: CSIDrivers{azureDriver: azureRefs},
		},
		{
			name:    "multiple drivers",
			entries: []string{awsDriver, azureDriver},
			expected: CSIDrivers{
				awsDriver:   hostnameRefs,
				azureDriver: azureRefs,
			},
		},
		{
			name:     "explicit affinity without key defaults to hostname",
			entries:  []string{"x.csi.example.com=affinity"},
			expected: CSIDrivers{"x.csi.example.com": hostnameRefs},
		},
		{
			name:     "explicit affinity key",
			entries:  []string{"x.csi.example.com=affinity:topology.x/node"},
			expected: CSIDrivers{"x.csi.example.com": {{Kind: NodeRefAffinity, Key: "topology.x/node"}}},
		},
		{
			name:    "explicit references override the built-in ones, in order",
			entries: []string{azureDriver + "=attribute:my/attr|annotation:my/ann"},
			expected: CSIDrivers{azureDriver: {
				{Kind: NodeRefAttribute, Key: "my/attr"},
				{Kind: NodeRefAnnotation, Key: "my/ann"},
			}},
		},
		{
			name:     "whitespace is trimmed",
			entries:  []string{" " + awsDriver + " "},
			expected: CSIDrivers{awsDriver: hostnameRefs},
		},
		{name: "blank entries are ignored", entries: []string{"", "  "}, expected: CSIDrivers{}},
		{name: "duplicate driver", entries: []string{awsDriver, awsDriver}, expectedErr: "duplicate"},
		{name: "empty driver name", entries: []string{"=annotation:a/b"}, expectedErr: "driver name"},
		{name: "empty references", entries: []string{awsDriver + "="}, expectedErr: "reference"},
		{name: "unknown reference kind", entries: []string{awsDriver + "=label:foo"}, expectedErr: "unknown"},
		{name: "annotation without key", entries: []string{awsDriver + "=annotation"}, expectedErr: "key"},
		{name: "attribute with empty key", entries: []string{awsDriver + "=attribute:"}, expectedErr: "key"},
		{name: "affinity with empty key", entries: []string{awsDriver + "=affinity:"}, expectedErr: "key"},
		{name: "empty alternative", entries: []string{awsDriver + "=affinity||annotation:a/b"}, expectedErr: "reference"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseCSIDrivers(test.entries)
			if test.expectedErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.expectedErr) {
					t.Fatalf("expected error containing %q, got %v", test.expectedErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, test.expected) {
				t.Errorf("expected %#v, got %#v", test.expected, got)
			}
		})
	}
}

func TestParseCSIDriversDoesNotShareBuiltInSlices(t *testing.T) {
	a, err := ParseCSIDrivers([]string{azureDriver})
	if err != nil {
		t.Fatal(err)
	}
	a[azureDriver][0].Key = "mutated"
	b, _ := ParseCSIDrivers([]string{azureDriver})
	if b[azureDriver][0].Key != azureSel {
		t.Errorf("built-in references were mutated through a parsed result: %#v", b[azureDriver])
	}
}

func csiPV(driver, storageClassName string) *v1.PersistentVolume {
	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "pv"},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{CSI: &v1.CSIPersistentVolumeSource{Driver: driver}},
			StorageClassName:       storageClassName,
		},
	}
}

func affinityPV(pv *v1.PersistentVolume, key string, nodes ...string) *v1.PersistentVolume {
	pv.Spec.NodeAffinity = &v1.VolumeNodeAffinity{Required: &v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{{
		MatchExpressions: []v1.NodeSelectorRequirement{{Key: key, Operator: v1.NodeSelectorOpIn, Values: nodes}},
	}}}}
	return pv
}

func twoTermPV(pv *v1.PersistentVolume, key1, value1, key2, value2 string) *v1.PersistentVolume {
	pv.Spec.NodeAffinity = &v1.VolumeNodeAffinity{Required: &v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{
		{MatchExpressions: []v1.NodeSelectorRequirement{{Key: key1, Operator: v1.NodeSelectorOpIn, Values: []string{value1}}}},
		{MatchExpressions: []v1.NodeSelectorRequirement{{Key: key2, Operator: v1.NodeSelectorOpIn, Values: []string{value2}}}},
	}}}
	return pv
}

func notInPV(pv *v1.PersistentVolume, key, value string) *v1.PersistentVolume {
	pv.Spec.NodeAffinity = &v1.VolumeNodeAffinity{Required: &v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{
		{MatchExpressions: []v1.NodeSelectorRequirement{{Key: key, Operator: v1.NodeSelectorOpNotIn, Values: []string{value}}}},
	}}}
	return pv
}

func attrPV(pv *v1.PersistentVolume, key, value string) *v1.PersistentVolume {
	if pv.Spec.CSI.VolumeAttributes == nil {
		pv.Spec.CSI.VolumeAttributes = map[string]string{}
	}
	pv.Spec.CSI.VolumeAttributes[key] = value
	return pv
}

func annPV(pv *v1.PersistentVolume, key, value string) *v1.PersistentVolume {
	if pv.Annotations == nil {
		pv.Annotations = map[string]string{}
	}
	pv.Annotations[key] = value
	return pv
}

func mustParse(t *testing.T, entries ...string) CSIDrivers {
	t.Helper()
	d, err := ParseCSIDrivers(entries)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestIsPVEligibleForNodeCleanup(t *testing.T) {
	localPV := func(sc string) *v1.PersistentVolume {
		return &v1.PersistentVolume{Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{Local: &v1.LocalVolumeSource{}},
			StorageClassName:       sc,
		}}
	}

	tests := []struct {
		name              string
		pv                *v1.PersistentVolume
		storageClassNames []string
		csiDrivers        CSIDrivers
		expected          bool
	}{
		{"local PV with matching StorageClass, no csi drivers", localPV("sc"), []string{"sc"}, nil, true},
		{"local PV without matching StorageClass, csi drivers set", localPV("other"), []string{"sc"}, mustParse(t, awsDriver), false},
		{"CSI PV with matching StorageClass and driver", csiPV(awsDriver, "sc"), []string{"sc"}, mustParse(t, awsDriver), true},
		{"CSI PV, one of multiple drivers", csiPV(awsDriver, "sc"), []string{"sc", "sc2"}, mustParse(t, azureDriver, awsDriver), true},
		{"azure CSI PV with matching StorageClass and driver", csiPV(azureDriver, "sc"), []string{"sc"}, mustParse(t, azureDriver), true},
		{"CSI PV, no csi drivers configured (default)", csiPV(awsDriver, "sc"), []string{"sc"}, nil, false},
		{"CSI PV, empty csi drivers", csiPV(awsDriver, "sc"), []string{"sc"}, CSIDrivers{}, false},
		{"CSI PV, driver not allowlisted (e.g. EBS)", csiPV("ebs.csi.aws.com", "sc"), []string{"sc"}, mustParse(t, awsDriver, azureDriver), false},
		{"CSI PV, allowlisted driver but StorageClass not listed", csiPV(awsDriver, "gp3"), []string{"sc"}, mustParse(t, awsDriver), false},
		{"CSI PV, allowlisted driver and empty storageClassNames", csiPV(awsDriver, "sc"), []string{}, mustParse(t, awsDriver), false},
		{"CSI PV with empty driver name", csiPV("", "sc"), []string{"sc"}, CSIDrivers{"": {{Kind: NodeRefAffinity, Key: v1.LabelHostname}}}, false},
		{"PV with neither local nor CSI source", &v1.PersistentVolume{Spec: v1.PersistentVolumeSpec{StorageClassName: "sc"}}, []string{"sc"}, mustParse(t, awsDriver), false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsPVEligibleForNodeCleanup(test.pv, test.storageClassNames, test.csiDrivers); got != test.expected {
				t.Errorf("expected %t, got %t", test.expected, got)
			}
		})
	}
}

func TestResolvePVNodes(t *testing.T) {
	localPV := func(key string, nodes ...string) *v1.PersistentVolume {
		return affinityPV(&v1.PersistentVolume{Spec: v1.PersistentVolumeSpec{
			PersistentVolumeSource: v1.PersistentVolumeSource{Local: &v1.LocalVolumeSource{}},
		}}, key, nodes...)
	}

	tests := []struct {
		name       string
		pv         *v1.PersistentVolume
		csiDrivers CSIDrivers
		expected   []string
	}{
		{"local PV uses hostname affinity", localPV(v1.LabelHostname, "n1"), nil, []string{"n1"}},
		{"local PV with non-hostname affinity yields no node (never 'node gone')", localPV("other", "n1"), nil, nil},
		{"local PV without affinity yields no node", &v1.PersistentVolume{Spec: v1.PersistentVolumeSpec{PersistentVolumeSource: v1.PersistentVolumeSource{Local: &v1.LocalVolumeSource{}}}}, nil, nil},

		{"aws CSI PV uses hostname affinity", affinityPV(csiPV(awsDriver, "sc"), v1.LabelHostname, "n1"), mustParse(t, awsDriver), []string{"n1"}},
		{"aws CSI PV with multiple nodes", affinityPV(csiPV(awsDriver, "sc"), v1.LabelHostname, "n1", "n2"), mustParse(t, awsDriver), []string{"n1", "n2"}},
		{"aws CSI PV without hostname affinity yields no node", affinityPV(csiPV(awsDriver, "sc"), "topology.kubernetes.io/zone", "us-east-1a"), mustParse(t, awsDriver), nil},
		{"aws CSI PV without any affinity yields no node", csiPV(awsDriver, "sc"), mustParse(t, awsDriver), nil},

		{"aws CSI PV with an extra selector term lacking the hostname key yields no node (terms are ORed)", twoTermPV(csiPV(awsDriver, "sc"), v1.LabelHostname, "n1", "topology.kubernetes.io/zone", "us-east-1a"), mustParse(t, awsDriver), nil},
		{"aws CSI PV whose hostname term uses NotIn yields no node", notInPV(csiPV(awsDriver, "sc"), v1.LabelHostname, "n1"), mustParse(t, awsDriver), nil},

		// Real shape of a localdisk.csi.acstor.io PV: no nodeAffinity, node only in volumeAttributes.
		{"azure PV never failed over: falls back to selected-initial-node attribute", attrPV(csiPV(azureDriver, "local-csi"), azureInit, "aks-vmss000001"), mustParse(t, azureDriver), []string{"aks-vmss000001"}},
		{"azure PV after failover: selected-node annotation wins", annPV(attrPV(csiPV(azureDriver, "local-csi"), azureInit, "aks-vmss000001"), azureSel, "aks-vmss000002"), mustParse(t, azureDriver), []string{"aks-vmss000002"}},
		{"azure PV with only the annotation", annPV(csiPV(azureDriver, "local-csi"), azureSel, "aks-vmss000002"), mustParse(t, azureDriver), []string{"aks-vmss000002"}},
		{"azure PV with empty annotation falls back to attribute", annPV(attrPV(csiPV(azureDriver, "local-csi"), azureInit, "aks-vmss000001"), azureSel, ""), mustParse(t, azureDriver), []string{"aks-vmss000001"}},
		{"azure PV with no node reference yields no node", csiPV(azureDriver, "local-csi"), mustParse(t, azureDriver), nil},
		{"azure PV with only a hostname affinity is not used (not a reference for this driver)", affinityPV(csiPV(azureDriver, "local-csi"), v1.LabelHostname, "n1"), mustParse(t, azureDriver), nil},

		{"custom references: attribute then annotation", annPV(attrPV(csiPV("x", "sc"), "a/attr", "from-attr"), "a/ann", "from-ann"), mustParse(t, "x=attribute:a/attr|annotation:a/ann"), []string{"from-attr"}},
		{"custom affinity key", affinityPV(csiPV("x", "sc"), "topology.x/node", "n9"), mustParse(t, "x=affinity:topology.x/node"), []string{"n9"}},

		{"driver not configured yields no node even with hostname affinity", affinityPV(csiPV("ebs.csi.aws.com", "gp3"), v1.LabelHostname, "n1"), mustParse(t, awsDriver), nil},
		{"CSI PV with no drivers configured yields no node", affinityPV(csiPV(awsDriver, "sc"), v1.LabelHostname, "n1"), nil, nil},
		{"nil PV", nil, mustParse(t, awsDriver), nil},
		{"PV without any source yields no node", &v1.PersistentVolume{}, mustParse(t, awsDriver), nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ResolvePVNodes(test.pv, test.csiDrivers).Names
			if len(got) == 0 && len(test.expected) == 0 {
				return
			}
			if !reflect.DeepEqual(got, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, got)
			}
		})
	}
}

func TestResolvePVNodesLabelKey(t *testing.T) {
	localPV := affinityPV(&v1.PersistentVolume{Spec: v1.PersistentVolumeSpec{
		PersistentVolumeSource: v1.PersistentVolumeSource{Local: &v1.LocalVolumeSource{}},
	}}, v1.LabelHostname, "n1")

	tests := []struct {
		name       string
		pv         *v1.PersistentVolume
		csiDrivers CSIDrivers
		expected   PVNodes
	}{
		{"local PV: node names", localPV, nil, PVNodes{Names: []string{"n1"}}},
		{"hostname affinity ref", affinityPV(csiPV(awsDriver, "sc"), v1.LabelHostname, "n1"), mustParse(t, awsDriver), PVNodes{Names: []string{"n1"}, LabelKey: v1.LabelHostname}},
		{"custom affinity key: values of that node label", affinityPV(csiPV("x", "sc"), "topology.x/node", "n9"), mustParse(t, "x=affinity:topology.x/node"), PVNodes{Names: []string{"n9"}, LabelKey: "topology.x/node"}},
		{"annotation ref: node names", annPV(csiPV(azureDriver, "sc"), azureSel, "n2"), mustParse(t, azureDriver), PVNodes{Names: []string{"n2"}}},
		{"attribute ref: node names", attrPV(csiPV(azureDriver, "sc"), azureInit, "n3"), mustParse(t, azureDriver), PVNodes{Names: []string{"n3"}}},
		{"unknown", csiPV(awsDriver, "sc"), mustParse(t, awsDriver), PVNodes{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ResolvePVNodes(test.pv, test.csiDrivers)
			if len(got.Names) == 0 && len(test.expected.Names) == 0 && got.LabelKey == "" {
				return
			}
			if !reflect.DeepEqual(got, test.expected) {
				t.Errorf("expected %#v, got %#v", test.expected, got)
			}
		})
	}
}

func TestAnyPVNodeExists(t *testing.T) {
	// A Node whose name differs from the value of its topology label.
	labelled := &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "ip-10-0-0-1", Labels: map[string]string{"topology.x/node": "n9"}}}
	named := &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "n1"}}
	hostLabelled := &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "other", Labels: map[string]string{NodeLabelKey: "n2"}}}

	tests := []struct {
		name     string
		nodes    []*v1.Node
		query    PVNodes
		expected bool
	}{
		{"node names: found by name", []*v1.Node{named}, PVNodes{Names: []string{"n1"}}, true},
		{"node names: found by hostname label", []*v1.Node{hostLabelled}, PVNodes{Names: []string{"n2"}}, true},
		{"node names: not found", []*v1.Node{named}, PVNodes{Names: []string{"gone"}}, false},
		{"hostname label key behaves like node names", []*v1.Node{named}, PVNodes{Names: []string{"n1"}, LabelKey: v1.LabelHostname}, true},
		{"hostname label key: not found", []*v1.Node{named}, PVNodes{Names: []string{"gone"}, LabelKey: v1.LabelHostname}, false},
		{"custom label key: live node has the label value but a different name", []*v1.Node{labelled}, PVNodes{Names: []string{"n9"}, LabelKey: "topology.x/node"}, true},
		{"custom label key: no node has the label value", []*v1.Node{labelled}, PVNodes{Names: []string{"n8"}, LabelKey: "topology.x/node"}, false},
		{"custom label key: a node merely NAMED like the value does not count", []*v1.Node{{ObjectMeta: metav1.ObjectMeta{Name: "n9"}}}, PVNodes{Names: []string{"n9"}, LabelKey: "topology.x/node"}, false},
		{"custom label key: any of several values", []*v1.Node{labelled}, PVNodes{Names: []string{"n8", "n9"}, LabelKey: "topology.x/node"}, true},
		{"custom label key: invalid label value is treated as existing (conservative)", []*v1.Node{}, PVNodes{Names: []string{"not a valid value!"}, LabelKey: "topology.x/node"}, true},
		{"no names", []*v1.Node{named}, PVNodes{LabelKey: "topology.x/node"}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			informers := informers.NewSharedInformerFactory(fake.NewSimpleClientset(), time.Duration(0))
			nodeInformer := informers.Core().V1().Nodes()
			for _, n := range test.nodes {
				nodeInformer.Informer().GetStore().Add(n)
			}
			if got := AnyPVNodeExists(nodeInformer.Lister(), test.query); got != test.expected {
				t.Errorf("expected %t, got %t", test.expected, got)
			}
		})
	}
}
