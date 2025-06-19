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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/util"
)

// Remover is responsible for removing the node taint that indidcates the provisioner is not ready yet.
type Remover struct {
	RuntimeConfig *common.RuntimeConfig
	taintRemoved  bool
}

// NewRemover creates an instances of RemoveNodeNotReadyTaint.
func NewRemover(runtimeConfig *common.RuntimeConfig) *Remover {
	return &Remover{
		RuntimeConfig: runtimeConfig,
		taintRemoved:  false,
	}
}

// RemoveNodeTaint searches for the provisionerNotReadyNodeTaintKey and removes it from the node.
// it only removes the taint once.
func (n *Remover) RemoveNodeTaint() {
	userConfig := n.RuntimeConfig.UserConfig
	if !userConfig.RemoveNodeNotReadyTaint || n.taintRemoved {
		return
	}

	client := n.RuntimeConfig.Client.CoreV1()
	node := util.GetNode(client, n.RuntimeConfig.Node.Name)

	var taintExists bool
	currTaints := []corev1.Taint{}
	for _, taint := range node.Spec.Taints {
		if taint.Key == userConfig.ProvisionerNotReadyNodeTaintKey {
			taintExists = true
		} else {
			currTaints = append(currTaints, taint)
		}
	}

	if !taintExists {
		klog.Infof("ProvisionerNotReadyNodeTaintKey %s was not found on node %s", userConfig.ProvisionerNotReadyNodeTaintKey, node.Name)
		return
	}

	node.Spec.Taints = currTaints
	_, err := client.Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("failed to remove node taint %s from node %s: %v", userConfig.ProvisionerNotReadyNodeTaintKey, node.Name, err)
		return
	}

	n.taintRemoved = true
	klog.Infof("removed node taint %s from node %s", userConfig.ProvisionerNotReadyNodeTaintKey, node.Name)
}

// ShouldRemoveTaint returns true if the taint is not removed already and the user config is set to remove the taint.
func (n *Remover) ShouldRemoveTaint() bool {
	return !n.taintRemoved && n.RuntimeConfig.UserConfig.RemoveNodeNotReadyTaint
}
