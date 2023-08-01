/*
Copyright 2023 The Kubernetes Authors.

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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// LocalVolumeNodeCleanupSubsystem is prometheus subsystem name.
	LocalVolumeNodeCleanupSubsystem = "local_volume_node_cleanup"
)

var (
	// PersistentVolumeDeleteTotal is used to collect accumulated count of persistent volume delete attempts.
	// This metric will report as higher than the true amount of persistent volumes deleted if
	// the node-cleanup deleter has a short sync period.
	PersistentVolumeDeleteTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: LocalVolumeNodeCleanupSubsystem,
			Name:      "persistentvolume_delete_total",
			Help:      "Total number of successful persistent volume delete *attempts*. Broken down by persistent volume phase.",
		},
		[]string{"phase"},
	)
	// PersistentVolumeDeleteFailedTotal is used to collect accumulated count of persistent volume delete failed attempts.
	PersistentVolumeDeleteFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: LocalVolumeNodeCleanupSubsystem,
			Name:      "persistentvolume_delete_failed_total",
			Help:      "Total number of persistent volume delete failed attempts. Broken down by persistent volume phase.",
		},
		[]string{"phase"},
	)
	// PersistentVolumeClaimDeleteTotal is used to collect accumulated count of persistent volume claims deleted.
	PersistentVolumeClaimDeleteTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Subsystem: LocalVolumeNodeCleanupSubsystem,
			Name:      "persistentvolumeclaim_delete_total",
			Help:      "Total number of persistent volume claims deleted.",
		},
	)
	// PersistentVolumeClaimDeleteFailedTotal is used to collect accumulated count of persistent volume claim delete failed attempts.
	PersistentVolumeClaimDeleteFailedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Subsystem: LocalVolumeNodeCleanupSubsystem,
			Name:      "persistentvolumeclaim_delete_failed_total",
			Help:      "Total number of persistent volume claim delete failed attempts.",
		},
	)
)
