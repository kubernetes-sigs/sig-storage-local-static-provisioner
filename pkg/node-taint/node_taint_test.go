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

package nodetaint

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"
)

type testCase struct {
	name                 string
	node                 *corev1.Node
	userConfig           *common.UserConfig
	expectedTaints       []corev1.Taint
	expectedTaintRemoved bool
}

func TestRemoveNodeTaint(t *testing.T) {
	testCases := []testCase{
		{
			name: "should remove taint when it exists",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{Key: "test-taint-key", Value: "test-value", Effect: corev1.TaintEffectNoSchedule},
						{Key: "other-taint", Value: "other-value", Effect: corev1.TaintEffectNoSchedule},
					},
				},
			},
			userConfig: &common.UserConfig{
				RemoveNodeNotReadyTaint:         true,
				ProvisionerNotReadyNodeTaintKey: "test-taint-key",
			},
			expectedTaints: []corev1.Taint{
				{Key: "other-taint", Value: "other-value", Effect: corev1.TaintEffectNoSchedule},
			},
			expectedTaintRemoved: true,
		},
		{
			name: "should not remove taint when RemoveNodeNotReadyTaint is false",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{Key: "test-taint-key", Value: "test-value", Effect: corev1.TaintEffectNoSchedule},
					},
				},
			},
			userConfig: &common.UserConfig{
				RemoveNodeNotReadyTaint:         false,
				ProvisionerNotReadyNodeTaintKey: "test-taint-key",
			},
			expectedTaints: []corev1.Taint{
				{Key: "test-taint-key", Value: "test-value", Effect: corev1.TaintEffectNoSchedule},
			},
			expectedTaintRemoved: false,
		},
		{
			name: "should not remove taint when it doesn't exist",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{Key: "other-taint", Value: "other-value", Effect: corev1.TaintEffectNoSchedule},
					},
				},
			},
			userConfig: &common.UserConfig{
				RemoveNodeNotReadyTaint:         true,
				ProvisionerNotReadyNodeTaintKey: "test-taint-key",
			},
			expectedTaints: []corev1.Taint{
				{Key: "other-taint", Value: "other-value", Effect: corev1.TaintEffectNoSchedule},
			},
			expectedTaintRemoved: false,
		},
		{
			name: "should not remove taint when already removed",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{Key: "test-taint-key", Value: "test-value", Effect: corev1.TaintEffectNoSchedule},
					},
				},
			},
			userConfig: &common.UserConfig{
				RemoveNodeNotReadyTaint:         true,
				ProvisionerNotReadyNodeTaintKey: "test-taint-key",
			},
			expectedTaints: []corev1.Taint{
				{Key: "test-taint-key", Value: "test-value", Effect: corev1.TaintEffectNoSchedule},
			},
			expectedTaintRemoved: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset(tc.node)

			runtimeConfig := &common.RuntimeConfig{
				UserConfig: tc.userConfig,
				Client:     fakeClient,
			}
			runtimeConfig.UserConfig.Node = tc.node

			// For the "already removed" test case, mark taint as already removed
			remover := NewRemover(runtimeConfig)
			if tc.name == "should not remove taint when already removed" {
				remover.taintRemoved = true
			}

			err := remover.RemoveNodeTaint()
			if err != nil {
				t.Errorf("failed to remove node taint: %v", err)
			}

			updatedNode, err := fakeClient.CoreV1().Nodes().Get(context.Background(), tc.node.Name, metav1.GetOptions{})
			if err != nil {
				t.Errorf("failed to get updated node: %v", err)
			}
			if len(tc.expectedTaints) != len(updatedNode.Spec.Taints) {
				t.Errorf("expected %d taints, got %d", len(tc.expectedTaints), len(updatedNode.Spec.Taints))
			}

			// Verify taintRemoved flag
			if remover.taintRemoved != tc.expectedTaintRemoved {
				t.Errorf("expected taintRemoved to be %v, got %v", tc.expectedTaintRemoved, remover.taintRemoved)
			}
		})
	}
}
