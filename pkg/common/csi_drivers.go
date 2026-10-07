/*
Copyright 2026 The Kubernetes Authors.

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

package common

import (
	"fmt"
	"slices"
	"strings"

	v1 "k8s.io/api/core/v1"

	"sigs.k8s.io/sig-storage-local-static-provisioner/pkg/util"
)

// NodeRefKind is the place on a PV where a CSI driver records the node
// that its (node-local) volume depends on.
type NodeRefKind string

const (
	// NodeRefAffinity reads the node name(s) from the PV node affinity.
	NodeRefAffinity NodeRefKind = "affinity"
	// NodeRefAnnotation reads the node name from a PV annotation.
	NodeRefAnnotation NodeRefKind = "annotation"
	// NodeRefAttribute reads the node name from a CSI volume attribute.
	NodeRefAttribute NodeRefKind = "attribute"
)

// NodeRef describes where to find the node a PV depends on.
type NodeRef struct {
	Kind NodeRefKind
	// Key is the node affinity topology key, annotation key or volume
	// attribute key, depending on Kind.
	Key string
}

// CSIDrivers maps a CSI driver name to the ordered list of places where its
// PVs record the node they depend on. The first reference that yields a node
// wins. A CSI PV is only eligible for node cleanup if its driver is a key.
type CSIDrivers map[string][]NodeRef

// wellKnownCSIDrivers holds the node references of CSI drivers that do not use
// the default (kubernetes.io/hostname node affinity). They are used when the
// driver is listed without explicit references.
var wellKnownCSIDrivers = map[string][]NodeRef{
	// Azure Container Storage local disk CSI driver. Its PVs have no node
	// affinity. The node that currently owns the volume is recorded in the
	// selected-node annotation (set after a failover), otherwise the node that
	// created the volume is in the selected-initial-node volume attribute. This
	// is the same precedence the driver's own webhook uses.
	"localdisk.csi.acstor.io": {
		{Kind: NodeRefAnnotation, Key: "localdisk.csi.acstor.io/selected-node"},
		{Kind: NodeRefAttribute, Key: "localdisk.csi.acstor.io/selected-initial-node"},
	},
}

// ParseCSIDrivers parses the entries of the --csi-drivers flag. Each entry is
//
//	<driver>[=<ref>[|<ref>...]]
//
// where <ref> is one of
//
//	affinity[:<topologyKey>]   (topologyKey defaults to kubernetes.io/hostname)
//	annotation:<key>
//	attribute:<key>
//
// Without references, a well-known driver uses its built-in references and any
// other driver uses kubernetes.io/hostname node affinity.
func ParseCSIDrivers(entries []string) (CSIDrivers, error) {
	drivers := CSIDrivers{}
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		name, spec, hasSpec := strings.Cut(entry, "=")
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("invalid csi driver %q: empty driver name", entry)
		}
		if _, found := drivers[name]; found {
			return nil, fmt.Errorf("invalid csi drivers: duplicate driver %q", name)
		}

		if !hasSpec {
			if refs, found := wellKnownCSIDrivers[name]; found {
				drivers[name] = slices.Clone(refs)
			} else {
				drivers[name] = []NodeRef{{Kind: NodeRefAffinity, Key: v1.LabelHostname}}
			}
			continue
		}

		var refs []NodeRef
		for _, rawRef := range strings.Split(spec, "|") {
			ref, err := parseNodeRef(strings.TrimSpace(rawRef))
			if err != nil {
				return nil, fmt.Errorf("invalid csi driver %q: %w", entry, err)
			}
			refs = append(refs, ref)
		}
		drivers[name] = refs
	}
	return drivers, nil
}

func parseNodeRef(raw string) (NodeRef, error) {
	if raw == "" {
		return NodeRef{}, fmt.Errorf("empty node reference")
	}
	kind, key, hasKey := strings.Cut(raw, ":")
	key = strings.TrimSpace(key)
	if hasKey && key == "" {
		return NodeRef{}, fmt.Errorf("node reference %q has an empty key", raw)
	}

	switch NodeRefKind(kind) {
	case NodeRefAffinity:
		if !hasKey {
			key = v1.LabelHostname
		}
		return NodeRef{Kind: NodeRefAffinity, Key: key}, nil
	case NodeRefAnnotation, NodeRefAttribute:
		if !hasKey {
			return NodeRef{}, fmt.Errorf("node reference %q needs a key, e.g. %s:<key>", raw, kind)
		}
		return NodeRef{Kind: NodeRefKind(kind), Key: key}, nil
	default:
		return NodeRef{}, fmt.Errorf("unknown node reference kind %q (want affinity, annotation or attribute)", kind)
	}
}

// IsPVEligibleForNodeCleanup checks that a PV belongs to any of the passed in
// StorageClasses and is either a local PV, or a CSI PV provisioned by one of
// the passed in CSI drivers.
//
// CSI PVs are opt-in: with no csiDrivers only local PVs are eligible, so volumes
// managed by any other CSI driver (for example network-attached ones) are never
// considered for cleanup.
func IsPVEligibleForNodeCleanup(pv *v1.PersistentVolume, storageClassNames []string, csiDrivers CSIDrivers) bool {
	if pv == nil {
		return false
	}
	if IsLocalPVWithStorageClass(pv, storageClassNames) {
		return true
	}
	if pv.Spec.CSI == nil || pv.Spec.CSI.Driver == "" {
		return false
	}
	if _, found := csiDrivers[pv.Spec.CSI.Driver]; !found {
		return false
	}
	return slices.Contains(storageClassNames, pv.Spec.StorageClassName)
}

// GetNodeNamesForCleanup returns the name(s) of the node(s) a PV depends on:
// the node affinity hostname(s) for a local PV, or the node(s) found through
// the configured references of the PV's CSI driver. It returns nil if the node
// can not be determined, which callers must treat as "unknown", never as
// "the node is gone".
func GetNodeNamesForCleanup(pv *v1.PersistentVolume, csiDrivers CSIDrivers) []string {
	if pv == nil {
		return nil
	}

	if pv.Spec.CSI == nil {
		return nilIfEmpty(util.GetLocalPersistentVolumeNodeNames(pv))
	}

	for _, ref := range csiDrivers[pv.Spec.CSI.Driver] {
		var names []string
		switch ref.Kind {
		case NodeRefAffinity:
			names = util.GetPersistentVolumeNodeNames(pv, ref.Key)
		case NodeRefAnnotation:
			if value := pv.Annotations[ref.Key]; value != "" {
				names = []string{value}
			}
		case NodeRefAttribute:
			if value := pv.Spec.CSI.VolumeAttributes[ref.Key]; value != "" {
				names = []string{value}
			}
		}
		if len(names) > 0 {
			return names
		}
	}
	return nil
}

func nilIfEmpty(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	return names
}
