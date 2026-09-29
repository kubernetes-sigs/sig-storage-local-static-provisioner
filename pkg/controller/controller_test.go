/*
Copyright 2022 The Kubernetes Authors.

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
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"
	nodetaint "sigs.k8s.io/sig-storage-local-static-provisioner/pkg/node-taint"
)

func TestSignalStop(t *testing.T) {
	s := newSignal()
	defer s.close()

	sync := make(chan bool)

	service := func(signal *signal) {
		for {
			select {
			case stopped := <-signal.closing:
				stopped <- struct{}{}
				sync <- true
				return
			}
		}
	}

	go service(s)
	s.stop()

	if !<-sync {
		t.Error("Expected service to be successfully stopped")
	}
}

func TestShouldRemoveNodeTaint(t *testing.T) {
	newRemover := func(removeTaint bool) *nodetaint.Remover {
		node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "test-node"}}
		userConfig := &common.UserConfig{
			Node:                            node,
			RemoveNodeNotReadyTaint:         removeTaint,
			ProvisionerNotReadyNodeTaintKey: "test-taint-key",
		}
		runtimeConfig := &common.RuntimeConfig{
			UserConfig: userConfig,
			Client:     fake.NewSimpleClientset(node),
		}
		return nodetaint.NewRemover(runtimeConfig)
	}

	tests := []struct {
		name      string
		remover   *nodetaint.Remover
		readyzErr error
		want      bool
	}{
		{
			name:    "removes taint when enabled and ready",
			remover: newRemover(true),
			want:    true,
		},
		{
			name:      "does not remove taint when discovery is not ready",
			remover:   newRemover(true),
			readyzErr: errors.New("not ready"),
			want:      false,
		},
		{
			name:    "does not remove taint when feature disabled",
			remover: newRemover(false),
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldRemoveNodeTaint(test.remover, test.readyzErr); got != test.want {
				t.Fatalf("shouldRemoveNodeTaint() = %v, want %v", got, test.want)
			}
		})
	}
}
