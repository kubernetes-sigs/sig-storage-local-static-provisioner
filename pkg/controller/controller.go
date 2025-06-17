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
	nodetaint "sigs.k8s.io/sig-storage-local-static-provisioner/pkg/node-taint"
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

// signal represents an indication to from client to terminate a service and waits for a callback
// indicating that the service has successfully stopped.
type signal struct {
	closing chan chan struct{}
}

func newSignal() *signal {
	return &signal{
		closing: make(chan chan struct{}),
	}
}

func (s *signal) stop() {
	stopped := make(chan struct{})
	s.closing <- stopped
	<-stopped
}

func (s *signal) close() {
	close(s.closing)
}

// RunLocalController facilitates and manages the sync loop.
// It launches the main sync loop and if there is an updated configuration from the ConfigWatcher,
// it will inform the main sync loop to terminate and then will launch a new sync loop with the
// updated configuration.
func RunLocalController(configUpdate <-chan common.ProvisionerConfiguration, client *kubernetes.Clientset, ptable deleter.ProcTable, discoveryPeriod time.Duration, node *v1.Node, namespace, jobImage string, config common.ProvisionerConfiguration) {
	s := newSignal()
	defer s.close()

	startController := func(config common.ProvisionerConfiguration) {
		StartLocalController(s, client, ptable, discoveryPeriod, common.UserConfigFromProvisionerConfig(node, namespace, jobImage, config))
	}
	go startController(config)

	for {
		select {
		case newConfig := <-configUpdate:
			s.stop()
			go startController(newConfig)
		}
	}
}

// StartLocalController starts the sync loop for the local PV discovery and deleter
func StartLocalController(signal *signal, client *kubernetes.Clientset, ptable deleter.ProcTable, discoveryPeriod time.Duration, config *common.UserConfig) {
	klog.Info("Initializing volume cache\n")

	informerStopChan := make(chan struct{})
	jobControllerStopChan := make(chan struct{})

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

	var err error
	volumeUtil, err := util.NewVolumeUtil()
	if err != nil {
		klog.Fatalf("Error initializing VolumeUtil: %v", err)
	}

	runtimeConfig := &common.RuntimeConfig{
		UserConfig:      config,
		Cache:           cache.NewVolumeCache(),
		VolUtil:         volumeUtil,
		APIUtil:         util.NewAPIUtil(client),
		Client:          client,
		Name:            provisionerName,
		Recorder:        recorder,
		Mounter:         mount.New("" /* default mount path */),
		InformerFactory: informers.NewSharedInformerFactory(client, resyncPeriod),
	}

	populator.NewPopulator(runtimeConfig)

	var jobController deleter.JobController
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
	runtimeConfig.InformerFactory.Start(informerStopChan)
	// Wait for all started informers' cache were synced.
	for v, synced := range runtimeConfig.InformerFactory.WaitForCacheSync(wait.NeverStop) {
		if !synced {
			klog.Fatalf("Error syncing informer for %v", v)
		}
	}
	// Run controller logic.
	if jobController != nil {
		go jobController.Run(jobControllerStopChan)
	}
	klog.Info("Controller started\n")

	nodeTaintRemover := nodetaint.NewRemover(runtimeConfig)

	for {
		select {
		case stopped := <-signal.closing:
			close(informerStopChan)
			if jobController != nil {
				close(jobControllerStopChan)
			}
			stopped <- struct{}{}
			klog.Info("Controller stopped\n")
			return
		default:
			deleter.DeletePVs()
			discoverer.DiscoverLocalVolumes()
			if !nodeTaintRemover.ShouldRemoveTaint() && discoverer.Readyz.Check(nil) == nil {
				nodeTaintRemover.RemoveTaintWithBackoff()
			}
			time.Sleep(discoveryPeriod)
		}
	}
}
