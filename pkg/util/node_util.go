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
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog/v2"
)

const maxGetNodesRetries = 3

// GetNode returns the node with the given name.
func GetNode(client corev1.CoreV1Interface, name string) *v1.Node {
	var retries int

	for {
		node, err := client.Nodes().Get(context.TODO(), name, metav1.GetOptions{})
		if err == nil {
			return node
		}

		retries++
		klog.Infof("Could not get node information (remaining retries: %d): %v", maxGetNodesRetries-retries, err)

		if retries >= maxGetNodesRetries {
			klog.Fatalf("Could not get node information: %v", err)
		}
		time.Sleep(time.Second)
	}
}
