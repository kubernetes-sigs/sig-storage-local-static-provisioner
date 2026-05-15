/*
Copyright 2018 The Kubernetes Authors.

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

package e2e

import (
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Resolve the helm/provisioner chart directory
func getChartPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "../../helm/provisioner")
}

// Return a release name unique to the test namespace.
func helmReleaseName(ns string) string {
	return fmt.Sprintf("lsp-%s", ns)
}

// Return the DaemonSet name Helm will create for this release.
// Matches the fullnameOverride in buildHelmValues.
func helmDaemonSetName(ns string) string {
	return helmReleaseName(ns)
}

func newHelmActionConfig(namespace string) *action.Configuration {
	configFlags := genericclioptions.NewConfigFlags(true)
	configFlags.Namespace = &namespace
	ac := new(action.Configuration)
	err := ac.Init(configFlags, namespace, "memory", func(string, ...interface{}) {})
	Expect(err).NotTo(HaveOccurred())
	return ac
}

// Installs the local Helm chart into the test namespace.
// Creates ServiceAccount, RBAC, ConfigMap and DaemonSet
// Waits until DaemonSet pods are running.
func installProvisionerViaHelm(config *localTestConfig, testConfig *testConfig) {
	chart, err := loader.Load(getChartPath())
	Expect(err).NotTo(HaveOccurred())

	ac := newHelmActionConfig(config.ns)
	client := action.NewInstall(ac)
	client.Namespace = config.ns
	client.ReleaseName = helmReleaseName(config.ns)
	client.CreateNamespace = false
	client.Wait = true
	client.Timeout = 5 * time.Minute

	_, err = client.Run(chart, buildHelmValues(config, testConfig))
	Expect(err).NotTo(HaveOccurred())
}

// Removes all resources created by the release.
func uninstallProvisionerViaHelm(config *localTestConfig) {
	ac := newHelmActionConfig(config.ns)
	client := action.NewUninstall(ac)
	client.Wait = true
	client.Timeout = 2 * time.Minute
	_, err := client.Run(helmReleaseName(config.ns))
	Expect(err).NotTo(HaveOccurred())
}

// Constructs the values map for the provisioner Helm chart
func buildHelmValues(config *localTestConfig, testConfig *testConfig) map[string]interface{} {
	volumeMode := "Filesystem"
	if testConfig != nil && testConfig.VolumeType == BlockLocalVolumeType {
		volumeMode = "Block"
	}

	useJobForCleaning := false
	if testConfig != nil {
		useJobForCleaning = testConfig.UseJobForCleaning
	}

	return map[string]interface{}{
		// fullnameOverride gives all resources a namespace-unique name.
		"fullnameOverride":  helmReleaseName(config.ns),
		"image":             provisionerImageName,
		"imagePullPolicy":   string(provisionerImagePullPolicy),
		"useJobForCleaning": useJobForCleaning,
		"mountDevVolume":    false,
		"classes": []interface{}{
			map[string]interface{}{
				"name":         config.scName,
				"hostDir":      config.discoveryDir,
				"mountDir":     provisionerDefaultMountRoot,
				"volumeMode":   volumeMode,
				"storageClass": false,
			},
		},
	}
}
