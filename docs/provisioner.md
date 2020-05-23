# local-volume-provisioner

local-volume-provisioner is an out-of-tree static provisioner for the
Kubernetes [local volume](https://kubernetes.io/docs/concepts/storage/volumes/#local), which is GA feature since 1.14.

It runs on each node in the cluster and monitors specified directories to look for
new local file-based volumes.  The volumes can be a mount point or a directory in
a shared filesystem.  It then statically creates a Local PersistentVolume for each
local volume.  It also monitors when the PersistentVolumes have been released, and
will clean up the volume, and recreate the PV.

## Table of Contents

- [Changelog](#changelog)
- [Design](#design)
- [Configuration](#configuration)
- [Monitoring](#monitoring)
  * [Metrics](#metrics)
  * [Readiness](#readiness)

## Changelog

See [CHANGELOG.md](../CHANGELOG.md).

## Design

There is one provisioner instance on each node in the cluster.  Each instance is
responsible for monitoring and managing the local volumes on its node.

The basic components of the provisioner are as follows:

- Discovery: The discovery routine periodically reads the configured discovery
  directories and looks for new mount points that don't have a PV, and creates
  a PV for it.

- Deleter: The deleter routine is invoked by the Informer when a PV phase changes.
  If the phase is Released, then it cleans up the volume and deletes the PV API
  object.

- Cache: A central cache stores all the Local PersistentVolumes that the provisioner
  has created.  It is populated by a PV informer that filters out the PVs that
  belong to this node and have been created by this provisioner.  It is used by
  the Discovery and Deleter routines to get the existing PVs.

- Controller: The controller runs a sync loop that coordinates the other components.
  The discovery and deleter run serially to simplify synchronization with the cache
  and create/delete operations.

## Configuration

We configure local volume provisioner using ConfigMap. The explanation and
default value of each key are as follows:

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: <provisioner-config-name>
  namespace: <provisioner-ns>
data:
  # `nodeLabelsForPV` key contains a list of node labels to be copied to the PVs
  # created by the provisioner.
  #
  # This shows an example to copy hostname label to PVs.
  #
  #   nodeLabelsForPV: |
  #   - kubernetes.io/hostname
  #
  # Change of this does not affect current PVs.
  # By default, this key is empty, no labels will be copied.

  # `useAlphaAPI` key indicates whether alpha API should be used or not. Enable
  # this only in Kubernetes pre-1.10.
  useAlphaAPI: "false"

  # `useJobForCleaning` key indicates whether to start a job to clean volume. By default,
  # provisioner will clean volume in its own process.
  useJobForCleaning: "false"

  # `minResyncPeriod` key specifies minimum resync period. By default, it's
  # value is `5m0s`.
  # It is usually not necessary to adjust it
  minResyncPeriod: "5m0s"

  # `useNodeNameOnly` indicates whether node name should be used in the
  # provisioner name. Default is false.
  # WARN: Provisioner sets its name in PV's annotations (key
  # `pv.kubernetes.io/provisioned-by`. If you change this setting, you must
  # update annotations of all previous PVs (if have).
  useNodeNameOnly: "false"

  # `labelsForPV` contains a map of key value pairs as additional labels to be
  # added to PVs.
  #
  # This shows an example to add `foo:bar` label to PVs.
  #
  #   labelsForPV: |
  #     foo: bar
  #
  # Change of this does not affect current PVs.
  # By default, this key is empty, no additional labels will be added.

  # `setPVOwnerRef` indicates whether PVs discovered should be dependents of
  # owner node. Default is false.
  # Change of this does not affect current PVs.
  setPVOwnerRef: false

  # `storageClassMap` is a map. The key is the name of local storage class.
  # More than one storage classes can be configured.
  #
  # Here is an example to discover volumes under `/mnt/fast-disks` as
  # Filesystem mode PV with ext4 as fs type.
  #
  #   storageClassMap: |
  #     # the name of local storage class
  #     fast-disks:
  #       # path to the directory of local volumes
  #       hostDir: /mnt/fast-disks
  #       # the mount path of host directory in provisioner pod
  #       mountDir:  /mnt/fast-disks
  #       # If the local volume is a device, command configured here will be
  #       # used to clean it. This can be omitted and the default command
  #       # `/scripts/quick_reset.sh` will be used.
  #       blockCleanerCommand:
  #       - "/scripts/shred.sh"
  #       - "2"
  #       # The volume mode of PV. It defines whehter a device volume is #
  #       # intended to use as a formatted filesystem volume or to remain in block
  #       # state. Value of Filesystem is implied when omitted.
  #       volumeMode: Filesystem
  #       # The filesystem to format before mounting on the node. This applies
  #       # only when the volume source is a device and mode is Filesystem.
  #       # The default value is to auto-select a fileystem in Kubernetes if unspecified.
  #       fsType: ext4
  #       # name pattern check
  #       # only discover file name matching pattern("*" by default).
  #       namePattern: "*"
  #
  # By default, no configuration is configured for any storage class. In
  # production, you must configure for at least one storage class.
```

Note that, when you deploy provisioner with `helm`. You must configure
provisioner via helm values, please refer to our [helm docs](/helm).

## Monitoring

A dedicated HTTP server (default listening on 0.0.0.0:8080) exposes metrics and
readiness state.

### Metrics

The metrics are exported through the Prometheus golang client on the path `/metrics`.

| Metric name                                                   | Metric type | Labels                                                                                                                                                                             |
| ----------                                                    | ----------- | -----------                                                                                                                                                                        |
| local_volume_provisioner_persistentvolume_capacity_bytes      | Gauge       | `mode`=&lt;persistentvolume-mode&gt;                                                                                                                                               |
| local_volume_provisioner_persistentvolume_discovery_total     | Counter     | `mode`=&lt;persistentvolume-mode&gt;                                                                                                                                               |
| local_volume_provisioner_persistentvolume_discovery_duration_seconds   | Histogram   | `mode`=&lt;persistentvolume-mode&gt;                                                                                                                                               |
| local_volume_provisioner_persistentvolume_delete_total        | Counter     | `mode`=&lt;persistentvolume-mode&gt; <br> `type`=&lt;process&#124;job&gt;                                                                                                          |
| local_volume_provisioner_persistentvolume_delete_failed_total | Counter     | `mode`=&lt;persistentvolume-mode&gt; <br> `type`=&lt;process&#124;job&gt;                                                                                                          |
| local_volume_provisioner_persistentvolume_delete_duration_seconds      | Histogram   | `mode`=&lt;persistentvolume-mode&gt; <br> `type`=&lt;process&#124;job&gt; <br> `capacity`=&lt;volume-capacity-breakdown-by-500G&gt; <br> `cleanup_command`=&lt;cleanup-command&gt; |
| local_volume_provisioner_apiserver_requests_total             | Counter     | `method`=&lt;request-method&gt;                                                                                                                                                    |
| local_volume_provisioner_apiserver_requests_failed_total      | Counter     | `method`=&lt;request-method&gt;                                                                                                                                                    |
| local_volume_provisioner_apiserver_requests_duration_seconds           | Histogram   | `method`=&lt;request-method&gt;                                                                                                                                                    |
| local_volume_provisioner_proctable_running                    | Gauge       |                                                                                                                                                                                    |
| local_volume_provisioner_proctable_failed                     | Gauge       |                                                                                                                                                                                    |
| local_volume_provisioner_proctable_succeeded                  | Gauge       |                                                                                                                                                                                    |

### Readiness

The readiness state is exposed on the path `/ready`.

The state become ready when discovered local volumes are successfully created.

Note that if there is no disk to create, the state will be marked as ready.
