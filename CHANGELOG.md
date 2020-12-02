# HEAD

# [v2.4.0](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/releases/tag/v2.4.0)

Image updates:

- add `namePattern` field to filter volumes
  ([#187](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/187))

- blkdiscard.sh no longer zeros disks. This script was passing the -z option to
  blkdiscard which meant it was not performing discards. This has been fixed.
  If you desire zeroing, rather than block discarding, please switch to
  dd_zero.sh.
  ([#200](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/200))

- handle DeletedFinalStateUnknown object when receiving PV delete event
  ([222](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/222))

- We start to push multi-arch images to Kubernetes main image-serving system,
  our repository is hosted at k8s.gcr.io/sig-storage/local-volume-provisioner.
  Our legacy repository quay.io/external_storage/local-volume-provisioner is
  deprecated but still maintained. Note that only amd64 images will be pushed
  to this repository.
  ([206](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/206))

Helm updates:

- **Action required**: As the helm-chart structure changed the already running
  pod will be recreated during upgrade. Documentation can be found under
  [helm/README.md](./helm/README.md). Compare your existing values with the new
  chart parameter before upgrade.
  ([#179](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/179))

- Added daemonset.podAnnotations and daemonset.podLabels to Helm chart values.
  ([#213](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/213))

- Add opt-out for `/dev` volume in the chart
  ([#219](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/219))

- Accept `labelsForPV` elements in the chart
  ([220](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/220))

- Allow unprivileged provisioner in chart
  ([221](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/221))

# [v2.3.4](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/releases/tag/v2.3.4)

Image updates:

- A readiness check is added to expose discovery state
  Refer to [docs](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/blob/v2.3.4/docs/provisioner.md#readiness) for more information.
  ([#135](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/135))
- A new metric `local_volume_provisioner_persistentvolume_capacity_bytes` is
  added to report the capacity in bytes of the local volumes discovered
  ([#160](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/160))
- Fix an issue that may cause released PVs not to be recycled
  ([#174](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/174))

# [v2.3.3](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/releases/tag/v2.3.3)

Image updates:
- Allow user to configure additional PV labels
  ([#118](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/118))
- Add an option to create PVs owned by their respective Nodes
  ([#123](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/123))

Deployment updates:
- Fix invalid pod security policy in helm chart
  ([#93](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/93))
- Able to set storage class default in Kubernetes
  ([#125](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/125))

# [v2.3.2](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/releases/tag/v2.3.2)

Image updates:
- Fix an issue in block devices cleanup by Kubernetes Job
  ([#60](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/60))

Deployment updates:
- Support pod security policy
  ([#73](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/73))
- Support pod priority class
  ([#53](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/53))
- Minor bugs fixed

# [v2.3.1](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/releases/tag/v2.3.1)

Abandoned and not released.

# [v2.3.0](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/releases/tag/v2.3.0)

Image updates:
* Support mount options from StorageClass
  ([#1005](https://github.com/kubernetes-incubator/external-storage/pull/1005))
* Support fs volumes on block
  ([#980](https://github.com/kubernetes-incubator/external-storage/pull/980)).
  **Breaking change:** The change breaks backwards compatibility for block volumes: Users must explicitly set volumeMode to "Block" in config if a StorageClass is expected to be used for block volumes.
* Update base image to k8s.gcr.io/debian-base-amd64:0.4.0
  ([#1040](https://github.com/kubernetes-incubator/external-storage/pull/1040))

Deployment updates:
* Add option for nodeSelector in DaemonSet template
  ([#1022](https://github.com/kubernetes-incubator/external-storage/pull/1022))
* Add option to create namespace and use apps/v1 DaemonSet
  ([#1039](https://github.com/kubernetes-incubator/external-storage/pull/1039))
* Fixes provisioner jobs role name in helm template
  ([#1073](https://github.com/kubernetes-incubator/external-storage/pull/1073))

# [v2.2.0](https://github.com/kubernetes-incubator/external-storage/releases/tag/local-volume-provisioner-v2.2.0)
Image updates:
* Add Prometheus metrics
  ([#721](https://github.com/kubernetes-incubator/external-storage/pull/721))
* Support Retain reclaim policy
  ([#776](https://github.com/kubernetes-incubator/external-storage/pull/776))
* Add option for resync period and add a default of 5 minutes
  ([#800](https://github.com/kubernetes-incubator/external-storage/pull/800))
* Add option for cleaning filesystem PVs in a Job
  ([#863](https://github.com/kubernetes-incubator/external-storage/pull/863))
* Add option for using only Node.Name as the provisioner name, instead of Node.UID ([#947](https://github.com/kubernetes-incubator/external-storage/pull/947))

Deployment updates:
* Refactor helm generation
  ([#789](https://github.com/kubernetes-incubator/external-storage/pull/789))
* Add option for tolerations
  ([#792](https://github.com/kubernetes-incubator/external-storage/pull/792))
* Add /dev volume mount for raw block support
  ([#799)](https://github.com/kubernetes-incubator/external-storage/pull/799)
* Add option for resource requests and limits
  ([#831](https://github.com/kubernetes-incubator/external-storage/pull/831))

# [v2.1.0](https://github.com/kubernetes-incubator/external-storage/releases/tag/local-volume-provisioner-v2.1.0)
The following changes require Kubernetes 1.10 or higher.
* Add block volumeMode discovery and cleanup.
* **Important:** Beta PV.NodeAffinity field is used by default. If running against an older K8s version,
  the `useAlphaAPI` flag must be set in the configMap.

# [v2.0.0](https://github.com/kubernetes-incubator/external-storage/releases/tag/local-volume-provisioner-v2.0.0)
**Important:** This version is incompatible and has breaking changes with v1!
* Remove default config, a configmap is now required.
* Configmap data is changed from json to yaml syntax.
* All local volumes must be mount points.  For directory-based volumes, a
  bind-mount must be done in order for the provisioner to discover them. This
  requires the K8s [mount propagation feature](https://kubernetes.io/docs/concepts/storage/volumes/#mount-propagation)
  to be enabled.
* Detected capacity is rounded down to the nearest GB.
* New option to specify which node labels to add to the PV.

# [v1.0.1](https://github.com/kubernetes-incubator/external-storage/releases/tag/local-volume-provisioner-bootstrap-v1.0.1)
* Change fs capacity detection to use K8s volume util method.
* Add event on PV if cleanup or deletion fails.

# [v1.0.0](https://github.com/kubernetes-incubator/external-storage/releases/tag/local-volume-provisioner-bootstrap-v1.0.0)
* Run a provisioner on each node via DaemonSet.
* Discovers file-based volumes under configurable discovery directories and creates a local PV for each.
* When PV created by the provisioner is released, delete file contents and delete PV, to be discovered again.
* Use PV informer to populate volume cache.
