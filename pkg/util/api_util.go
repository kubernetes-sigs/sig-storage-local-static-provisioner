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

package util

import (
	"context"
	"time"

	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/metrics"

	batch_v1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// APIUtil is an interface for the K8s API
type APIUtil interface {
	// Create PersistentVolume object
	CreatePV(pv *v1.PersistentVolume) (*v1.PersistentVolume, error)

	// Delete PersistentVolume object
	DeletePV(pvName string) error

	// CreateJob Creates a Job execution.
	CreateJob(job *batch_v1.Job) error

	// DeleteJob deletes specified Job by its name and namespace.
	DeleteJob(jobName string, namespace string) error
}

var _ APIUtil = &apiUtil{}

type apiUtil struct {
	client kubernetes.Interface
}

// NewAPIUtil creates a new APIUtil object that represents the K8s API
func NewAPIUtil(client kubernetes.Interface) APIUtil {
	return &apiUtil{client: client}
}

// CreatePV will create a PersistentVolume
func (u *apiUtil) CreatePV(pv *v1.PersistentVolume) (*v1.PersistentVolume, error) {
	startTime := time.Now()
	metrics.APIServerRequestsTotal.WithLabelValues(metrics.APIServerRequestCreate).Inc()
	pv, err := u.client.CoreV1().PersistentVolumes().Create(context.TODO(), pv, metav1.CreateOptions{})
	metrics.APIServerRequestsDurationSeconds.WithLabelValues(metrics.APIServerRequestCreate).Observe(time.Since(startTime).Seconds())
	if err != nil {
		metrics.APIServerRequestsFailedTotal.WithLabelValues(metrics.APIServerRequestCreate).Inc()
	}
	return pv, err
}

// DeletePV will delete a PersistentVolume
func (u *apiUtil) DeletePV(pvName string) error {
	startTime := time.Now()
	metrics.APIServerRequestsTotal.WithLabelValues(metrics.APIServerRequestDelete).Inc()
	err := u.client.CoreV1().PersistentVolumes().Delete(context.TODO(), pvName, metav1.DeleteOptions{})
	metrics.APIServerRequestsDurationSeconds.WithLabelValues(metrics.APIServerRequestDelete).Observe(time.Since(startTime).Seconds())
	if err != nil {
		metrics.APIServerRequestsFailedTotal.WithLabelValues(metrics.APIServerRequestDelete).Inc()
	}
	return err
}

func (u *apiUtil) CreateJob(job *batch_v1.Job) error {
	_, err := u.client.BatchV1().Jobs(job.Namespace).Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (u *apiUtil) DeleteJob(jobName string, namespace string) error {
	deleteProp := metav1.DeletePropagationForeground
	if err := u.client.BatchV1().Jobs(namespace).Delete(context.TODO(), jobName, metav1.DeleteOptions{PropagationPolicy: &deleteProp}); err != nil {
		return err
	}

	return nil
}
