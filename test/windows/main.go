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

package main

import (
	"flag"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/klog/v2"
	e2estorageutils "k8s.io/kubernetes/test/e2e/storage/utils"
)

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	flag.Parse()

	klog.Infof("Remotely executing a program!")

	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "e2e-test-mauriciopoppe-windows-node-group-l64t",
		},
	}
	hostExec := NewHostExec()
	result, err := hostExec.Execute("echo $env:USERPROFILE", node)
	if err != nil {
		panic(err)
	}
	e2estorageutils.LogResult(result)
}
