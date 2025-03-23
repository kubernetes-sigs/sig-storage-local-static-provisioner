/*
Copyright 2025 The Kubernetes Authors.

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
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetPersistentVolumeNodeNames(t *testing.T) {
	tests := []struct {
		name              string
		pv                *v1.PersistentVolume
		expectedNodeNames []string
	}{
		{
			name: "nil PV",
			pv:   nil,
		},
		{
			name: "PV missing node affinity",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			},
		},
		{
			name: "PV node affinity missing required",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: v1.PersistentVolumeSpec{
					NodeAffinity: &v1.VolumeNodeAffinity{},
				},
			},
		},
		{
			name: "PV node affinity required zero selector terms",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: v1.PersistentVolumeSpec{
					NodeAffinity: &v1.VolumeNodeAffinity{
						Required: &v1.NodeSelector{
							NodeSelectorTerms: []v1.NodeSelectorTerm{},
						},
					},
				},
			},
			expectedNodeNames: []string{},
		},
		{
			name: "PV node affinity required zero selector terms",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: v1.PersistentVolumeSpec{
					NodeAffinity: &v1.VolumeNodeAffinity{
						Required: &v1.NodeSelector{
							NodeSelectorTerms: []v1.NodeSelectorTerm{},
						},
					},
				},
			},
			expectedNodeNames: []string{},
		},
		{
			name: "PV node affinity required zero match expressions",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: v1.PersistentVolumeSpec{
					NodeAffinity: &v1.VolumeNodeAffinity{
						Required: &v1.NodeSelector{
							NodeSelectorTerms: []v1.NodeSelectorTerm{
								{
									MatchExpressions: []v1.NodeSelectorRequirement{},
								},
							},
						},
					},
				},
			},
			expectedNodeNames: []string{},
		},
		{
			name: "PV node affinity required multiple match expressions",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: v1.PersistentVolumeSpec{
					NodeAffinity: &v1.VolumeNodeAffinity{
						Required: &v1.NodeSelector{
							NodeSelectorTerms: []v1.NodeSelectorTerm{
								{
									MatchExpressions: []v1.NodeSelectorRequirement{
										{
											Key:      "foo",
											Operator: v1.NodeSelectorOpIn,
										},
										{
											Key:      "bar",
											Operator: v1.NodeSelectorOpIn,
										},
									},
								},
							},
						},
					},
				},
			},
			expectedNodeNames: []string{},
		},
		{
			name: "PV node affinity required single match expression with no values",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: v1.PersistentVolumeSpec{
					NodeAffinity: &v1.VolumeNodeAffinity{
						Required: &v1.NodeSelector{
							NodeSelectorTerms: []v1.NodeSelectorTerm{
								{
									MatchExpressions: []v1.NodeSelectorRequirement{
										{
											Key:      v1.LabelHostname,
											Operator: v1.NodeSelectorOpIn,
											Values:   []string{},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedNodeNames: []string{},
		},
		{
			name: "PV node affinity required single match expression with single node",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: v1.PersistentVolumeSpec{
					NodeAffinity: &v1.VolumeNodeAffinity{
						Required: &v1.NodeSelector{
							NodeSelectorTerms: []v1.NodeSelectorTerm{
								{
									MatchExpressions: []v1.NodeSelectorRequirement{
										{
											Key:      v1.LabelHostname,
											Operator: v1.NodeSelectorOpIn,
											Values: []string{
												"node1",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedNodeNames: []string{
				"node1",
			},
		},
		{
			name: "PV node affinity required single match expression with multiple nodes",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: v1.PersistentVolumeSpec{
					NodeAffinity: &v1.VolumeNodeAffinity{
						Required: &v1.NodeSelector{
							NodeSelectorTerms: []v1.NodeSelectorTerm{
								{
									MatchExpressions: []v1.NodeSelectorRequirement{
										{
											Key:      v1.LabelHostname,
											Operator: v1.NodeSelectorOpIn,
											Values: []string{
												"node1",
												"node2",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedNodeNames: []string{
				"node1",
				"node2",
			},
		},
		{
			name: "PV node affinity required multiple match expressions with multiple nodes",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: v1.PersistentVolumeSpec{
					NodeAffinity: &v1.VolumeNodeAffinity{
						Required: &v1.NodeSelector{
							NodeSelectorTerms: []v1.NodeSelectorTerm{
								{
									MatchExpressions: []v1.NodeSelectorRequirement{
										{
											Key:      "bar",
											Operator: v1.NodeSelectorOpIn,
											Values: []string{
												"node1",
												"node2",
											},
										},
										{
											Key:      v1.LabelHostname,
											Operator: v1.NodeSelectorOpIn,
											Values: []string{
												"node3",
												"node4",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedNodeNames: []string{
				"node3",
				"node4",
			},
		},
		{
			name: "PV node affinity required multiple node selectors multiple match expressions with multiple nodes",
			pv: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: v1.PersistentVolumeSpec{
					NodeAffinity: &v1.VolumeNodeAffinity{
						Required: &v1.NodeSelector{
							NodeSelectorTerms: []v1.NodeSelectorTerm{
								{
									MatchExpressions: []v1.NodeSelectorRequirement{
										{
											Key:      v1.LabelHostname,
											Operator: v1.NodeSelectorOpIn,
											Values: []string{
												"node1",
												"node2",
											},
										},
										{
											Key:      v1.LabelHostname,
											Operator: v1.NodeSelectorOpIn,
											Values: []string{
												"node2",
												"node3",
											},
										},
									},
								},
								{
									MatchExpressions: []v1.NodeSelectorRequirement{
										{
											Key:      v1.LabelHostname,
											Operator: v1.NodeSelectorOpIn,
											Values: []string{
												"node1",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedNodeNames: []string{
				"node1",
				"node2",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nodeNames := GetLocalPersistentVolumeNodeNames(test.pv)
			if diff := cmp.Diff(test.expectedNodeNames, nodeNames); diff != "" {
				t.Errorf("Unexpected nodeNames (-want, +got):\n%s", diff)
			}
		})
	}
}
