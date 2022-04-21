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

package watcher

import (
	"reflect"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"
)

// ConfigWatcher monitors the config file periodically with the provided interval
// and compares the provisioner config that is currently applied and the loaded provisioner
// config. If a difference between the configs is detected, it will signal to the sync loop
// to restart.
type ConfigWatcher struct {
	configPath        string
	resyncPeriod      time.Duration
	lastAppliedConfig common.ProvisionerConfiguration
}

// NewConfigWatcher creates a new ConfigWatcher object with the provided
// path of the config file, period to run the load and compare, and the initial
// configuration that is currently being applied for the provisioner.
func NewConfigWatcher(configPath string, resyncPeriod time.Duration, config common.ProvisionerConfiguration) *ConfigWatcher {
	return &ConfigWatcher{
		configPath:        configPath,
		resyncPeriod:      resyncPeriod,
		lastAppliedConfig: config,
	}
}

// Run will start running the ConfigWatcher in a loop. During each reload cycle,
// it loads the configuration from the config file on disk and compares with the last applied
// configuration. If there is a difference, it will send the loaded configuration to the
// channel indicating restart sync loop is needed and then update its last applied configuration.
func (cw *ConfigWatcher) Run(configUpdate chan<- common.ProvisionerConfiguration) {
	provisionerConfig := common.ProvisionerConfiguration{
		StorageClassConfig: make(map[string]common.MountConfig),
		MinResyncPeriod:    metav1.Duration{Duration: 5 * time.Minute},
	}

	for {
		select {
		case <-time.After(cw.resyncPeriod):
			if err := common.LoadProvisionerConfigs(cw.configPath, &provisionerConfig); err != nil {
				klog.Fatalf("Error parsing Provisioner's configuration: %#v. Exiting...\n", err)
			}

			if !reflect.DeepEqual(cw.lastAppliedConfig, provisionerConfig) {
				klog.Infof("Loaded and detected updated configuration: %+v", provisionerConfig)
				klog.Infof("Signalling sync loop to restart to pick up updated configuration...")

				configUpdate <- provisionerConfig
				cw.lastAppliedConfig = provisionerConfig
			}
		}
	}
}
