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

package controller

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"k8s.io/klog/v2"

	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/cache"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/deleter"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/discovery"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/populator"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/util"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/mount"
)

// StartLocalController starts the sync loop for the local PV discovery and deleter
func StartLocalController(client *kubernetes.Clientset, ptable deleter.ProcTable, discoveryPeriod time.Duration, config *common.UserConfig) {
	klog.Info("Initializing volume cache\n")

	var provisionerName string
	if config.UseNodeNameOnly {
		provisionerName = fmt.Sprintf("local-volume-provisioner-%v", config.Node.Name)
	} else {
		provisionerName = fmt.Sprintf("local-volume-provisioner-%v-%v", config.Node.Name, config.Node.UID)
	}

	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(client.CoreV1().RESTClient()).Events("")})
	recorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: provisionerName})

	// We choose a random resync period between MinResyncPeriod and 2 *
	// MinResyncPeriod, so that local provisioners deployed on multiple nodes
	// at same time don't list the apiserver simultaneously.
	resyncPeriod := time.Duration(config.MinResyncPeriod.Seconds()*(1+rand.Float64())) * time.Second

	runtimeConfig := &common.RuntimeConfig{
		UserConfig:      config,
		Cache:           cache.NewVolumeCache(),
		VolUtil:         util.NewVolumeUtil(),
		APIUtil:         util.NewAPIUtil(client),
		Client:          client,
		Name:            provisionerName,
		Recorder:        recorder,
		Mounter:         mount.New("" /* default mount path */),
		InformerFactory: informers.NewSharedInformerFactory(client, resyncPeriod),
	}

	populator.NewPopulator(runtimeConfig)

	var jobController deleter.JobController
	var err error
	if runtimeConfig.UseJobForCleaning {
		labels := map[string]string{common.NodeNameLabel: config.Node.Name}
		jobController, err = deleter.NewJobController(labels, runtimeConfig)
		if err != nil {
			klog.Fatalf("Error initializing jobController: %v", err)
		}
		klog.Infof("Enabling Jobs based cleaning.")
	}
	cleanupTracker := &deleter.CleanupStatusTracker{ProcTable: ptable, JobController: jobController}

	discoverer, err := discovery.NewDiscoverer(runtimeConfig, cleanupTracker)
	if err != nil {
		klog.Fatalf("Error initializing discoverer: %v", err)
	}
	healthz.InstallPathHandler(http.DefaultServeMux, "/ready", discoverer.Readyz)

	deleter := deleter.NewDeleter(runtimeConfig, cleanupTracker)

	// Start informers after all event listeners are registered.
	runtimeConfig.InformerFactory.Start(wait.NeverStop)
	// Wait for all started informers' cache were synced.
	for v, synced := range runtimeConfig.InformerFactory.WaitForCacheSync(wait.NeverStop) {
		if !synced {
			klog.Fatalf("Error syncing informer for %v", v)
		}
	}
	// Run controller logic.
	if jobController != nil {
		go jobController.Run(wait.NeverStop)
	}
	klog.Info("Controller started\n")
	for {
		deleter.DeletePVs()
		discoverer.DiscoverLocalVolumes()
		time.Sleep(discoveryPeriod)
	}
}
