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

package populator

import (
	"strings"

	"k8s.io/klog/v2"
	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/common"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

// The Populator uses an Informer to populate the VolumeCache.
type Populator struct {
	*common.RuntimeConfig
}

// NewPopulator returns a Populator object to update the PV cache
func NewPopulator(config *common.RuntimeConfig) *Populator {
	p := &Populator{RuntimeConfig: config}
	sharedInformer := config.InformerFactory.Core().V1().PersistentVolumes()
	sharedInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pv, ok := obj.(*v1.PersistentVolume)
			if !ok {
				klog.Errorf("Added object is not a v1.PersistentVolume type")
				return
			}
			p.handlePVUpdate(pv)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newPV, ok := newObj.(*v1.PersistentVolume)
			if !ok {
				klog.Errorf("Updated object is not a v1.PersistentVolume type")
				return
			}
			p.handlePVUpdate(newPV)
		},
		DeleteFunc: func(obj interface{}) {
			pv, ok := obj.(*v1.PersistentVolume)
			if !ok {
				klog.Warningf("Deleted object is not a v1.PersistentVolume type")

				// When a delete is dropped, the relist will notice a pv in the local cache but not
				// in the list, leading to the insertion of a tombstone object which contains
				// the deleted pv.
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					klog.Errorf("Unknown object type in delete event %+v", obj)
					return
				}
				pv, ok = tombstone.Obj.(*v1.PersistentVolume)
				if !ok {
					klog.Errorf("Tombstone contained object is not a v1.PersistentVolume %+v", obj)
					return
				}
			}
			p.handlePVDelete(pv)
		},
	})
	return p
}

func (p *Populator) handlePVUpdate(pv *v1.PersistentVolume) {
	_, exists := p.Cache.GetPV(pv.Name)
	if exists {
		p.Cache.UpdatePV(pv)
	} else {
		if pv.Annotations != nil {
			provisioner, found := pv.Annotations[common.AnnProvisionedBy]
			if !found {
				return
			}
			if provisioner == p.Name {
				// This PV was created by this provisioner
				p.Cache.AddPV(pv)
				return
			}
			if p.UseNodeNameOnly {
				nodeLabel, ok := pv.ObjectMeta.Labels[common.NodeNameLabel]
				if !ok {
					return
				}
				if nodeLabel != p.Node.Name {
					return
				}
				if strings.HasPrefix(provisioner, p.Name+"-") {
					// This PV was created by this provisioner with useNodeNameOnly disabled
					klog.Infof("cacheing pv %q (useNodeNameOnly mode)", pv.Name)
					p.Cache.AddPV(pv)
				}
			}
		}
	}
}

func (p *Populator) handlePVDelete(pv *v1.PersistentVolume) {
	_, exists := p.Cache.GetPV(pv.Name)
	if exists {
		// Don't do cleanup, just delete from cache
		p.Cache.DeletePV(pv.Name)
	}
}
