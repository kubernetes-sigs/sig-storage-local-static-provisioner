/*
Copyright 2021 The Kubernetes Authors.

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
	windows "sigs.k8s.io/sig-storage-local-static-provisioner/test/e2e/windows"
)

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	flag.Parse()

	klog.Infof("Remotely executing a program!")
	klog.Infof(`This script tests that the windows hostExec implementation can execute commands through SSH`)

	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "<gce instance name>",
		},
	}
	hostExec := windows.NewHostExec()
	result, err := hostExec.Execute("echo $env:USERPROFILE", node)
	if err != nil {
		panic(err)
	}
	e2estorageutils.LogResult(result)

	vhd := windows.NewVHD("C:\\var\\lib\\kubelet\\plugins\\testplugin-0.csi.io\\disk-dev-9.vhdx", 1024*1024*1024)

	vhdStage, err := vhd.StageScript()
	if err != nil {
		panic(err)
	}
	result, err = hostExec.Execute(vhdStage, node)
	if err != nil {
		panic(err)
	}
	e2estorageutils.LogResult(result)

	vhdPublish, err := vhd.PublishScript("C:\\var\\lib\\kubelet\\plugins\\testplugin-0.csi.io\\mount-9")
	if err != nil {
		panic(err)
	}
	result, err = hostExec.Execute(vhdPublish, node)
	if err != nil {
		panic(err)
	}
	e2estorageutils.LogResult(result)

	result, err = hostExec.Execute(`"& { Get-Volume -UniqueId ("\\\\?\\" + (Get-Item -Path C:\\var\\lib\\kubelet\\plugins\\testplugin-0.csi.io\\mount-9).Target) }"`, node)
	if err != nil {
		panic(err)
	}
	e2estorageutils.LogResult(result)
}
