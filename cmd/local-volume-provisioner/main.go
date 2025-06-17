/*
Copyright 2017 The Kubernetes Authors.

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
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/controller"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/deleter"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/metrics"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/metrics/collectors"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/util"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/watcher"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	optListenAddress string
	optMetricsPath   string
	discoveryPeriod  time.Duration
	configSyncPeriod time.Duration
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	klog.InitFlags(nil)
	flag.StringVar(&optListenAddress, "listen-address", ":8080", "address on which to expose metrics and readiness status")
	flag.StringVar(&optMetricsPath, "metrics-path", "/metrics", "path under which to expose metrics")
	flag.DurationVar(&discoveryPeriod, "discovery-period", 10*time.Second, "the period for local volume discovery")
	flag.DurationVar(&configSyncPeriod, "config-sync-period", 5*time.Second, "the period to check if there has been any config changes")
	flag.Parse()
	flag.Set("logtostderr", "true")

	provisionerConfig := common.ProvisionerConfiguration{
		StorageClassConfig: make(map[string]common.MountConfig),
		MinResyncPeriod:    metav1.Duration{Duration: 5 * time.Minute},
	}
	if err := common.LoadProvisionerConfigs(common.ProvisionerConfigPath, &provisionerConfig); err != nil {
		klog.Fatalf("Error parsing Provisioner's configuration: %#v. Exiting...\n", err)
	}
	klog.Infof("Loaded configuration: %+v", provisionerConfig)
	klog.Infof("Ready to run...")

	nodeName := os.Getenv("MY_NODE_NAME")
	if nodeName == "" {
		klog.Fatalf("MY_NODE_NAME environment variable not set\n")
	}

	namespace := os.Getenv("MY_NAMESPACE")
	if namespace == "" {
		klog.Warningf("MY_NAMESPACE environment variable not set, will be set to default.\n")
		namespace = "default"
	}

	jobImage := os.Getenv("JOB_CONTAINER_IMAGE")
	if jobImage == "" {
		klog.Warningf("JOB_CONTAINER_IMAGE environment variable not set.\n")
	}

	client := common.SetupClient()
	node := util.GetNode(client.CoreV1(), nodeName)

	configUpdate := make(chan common.ProvisionerConfiguration)
	defer close(configUpdate)

	configWatcher := watcher.NewConfigWatcher(common.ProvisionerConfigPath, configSyncPeriod, provisionerConfig)
	klog.Info("Starting config watcher\n")
	go configWatcher.Run(configUpdate)

	klog.Info("Starting controller\n")
	procTable := deleter.NewProcTable()
	go controller.RunLocalController(configUpdate, client, procTable, discoveryPeriod, node, namespace, jobImage, provisionerConfig)

	klog.Infof("Starting metrics server at %s\n", optListenAddress)
	prometheus.MustRegister([]prometheus.Collector{
		metrics.PersistentVolumeCapacityBytes,
		metrics.PersistentVolumeDiscoveryTotal,
		metrics.PersistentVolumeDiscoveryDurationSeconds,
		metrics.PersistentVolumeDeleteTotal,
		metrics.PersistentVolumeDeleteDurationSeconds,
		metrics.PersistentVolumeDeleteFailedTotal,
		metrics.APIServerRequestsTotal,
		metrics.APIServerRequestsFailedTotal,
		metrics.APIServerRequestsDurationSeconds,
		collectors.NewProcTableCollector(procTable),
	}...)
	http.Handle(optMetricsPath, promhttp.Handler())
	log.Fatal(http.ListenAndServe(optListenAddress, nil))
}
