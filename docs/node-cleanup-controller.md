# Local Volume Node Cleanup Controller

The local volume node cleanup controller removes PersistentVolumes and PersistentVolumeClaims that reference deleted Nodes.

## Overview

Pods using a Local Persistent Volume are always scheduled to the same Node as the Local PV it uses (as opposed to if they were using a HostPath volume, for instance). When Nodes fail while having a Local PV attached to them, Pods using the Local PV become stuck since they can't be scheduled to a deleted Node. In other words, the Local PV and its corresponding PVC aren’t cleaned up when a Node becomes unavailable and so each of them (PV/PVC/Pods) becomes stuck. This results in a degraded or unavailable workload. This controller aims to clean up the stale Local PVs and their bound PVCs after Node deletion, allowing workloads to automatically recover.

### Important Considerations
- The controller does not clean data from disks, it only removes PV/PVC objects. This is because we make the assumption that when a Node is deleted, local ephemeral storage data is irrevocably lost. This is common in cloud environments, but not as common on-prem.
- Since the controller removes stale PVCs, a StatefulSet must be backing Pods in order for the workload to be automatically rescheduled. Otherwise, a new PVC will need to be manually created.

## Usage

Please see the example [deployment](../deployment/kubernetes/example/node-cleanup-controller/deployment.yaml) and [rbac](../deployment/kubernetes/example/node-cleanup-controller/rbac.yaml) for deploying the controller.

### CSI node-local volumes

By default the controller only handles PVs of type `local`. Some CSI drivers also provision node-local volumes, for example the AWS EC2 instance store CSI driver (`lis.csi.aws.com`) or the Azure Container Storage local disk CSI driver (`localdisk.csi.acstor.io`). Their PVs are of type `csi`, and when the Node is deleted they get stuck in the same way as local PVs.

To opt such volumes in, pass both flags:

```
--storageclass-names=<StorageClass using the driver>
--csi-drivers=<csi driver>[,<csi driver>...]
```

A CSI PV is only cleaned up if **both** its StorageClass is listed in `--storageclass-names` **and** its driver is listed in `--csi-drivers`. CSI PVs of any other driver (for example network-attached volumes such as EBS or EFS) are never touched, even if their StorageClass is listed, and when `--csi-drivers` is not set the behavior is unchanged.

Drivers record the Node a volume depends on in different places, so each `--csi-drivers` entry is `<driver>[=<ref>[|<ref>...]]`, where `<ref>` is one of:

| Reference | Reads the Node name from |
|---|---|
| `affinity[:<topologyKey>]` | The PV node affinity. `<topologyKey>` defaults to `kubernetes.io/hostname`. For `kubernetes.io/hostname` the values are Node names; for any other key they are values of that Node label, so a Node exists if it carries `<topologyKey>=<value>`. |
| `annotation:<key>` | A PV annotation. |
| `attribute:<key>` | A CSI volume attribute (`spec.csi.volumeAttributes`). |

References are tried in order and the first one that yields a Node name is used. If none does, the Node is unknown and the PV is left alone: an unknown Node is never treated as a deleted Node. Node selector terms of a PV are ORed, so for an `affinity` reference every term must contain a `<topologyKey> In (...)` expression; if any term does not, the Node is unknown.

Without references, a driver defaults to `affinity:kubernetes.io/hostname`, except for the well-known drivers below:

| Driver | Default references |
|---|---|
| `lis.csi.aws.com` | `affinity:kubernetes.io/hostname` |
| `localdisk.csi.acstor.io` | `annotation:localdisk.csi.acstor.io/selected-node\|attribute:localdisk.csi.acstor.io/selected-initial-node` |

Examples:

```
# AWS and Azure
--storageclass-names=ec2-instance-store-sc,local-csi
--csi-drivers=lis.csi.aws.com,localdisk.csi.acstor.io

# A driver that records the Node under its own topology key
--csi-drivers=example.csi.vendor.io=affinity:topology.example.io/node
```

The Azure driver's PVs have no node affinity. The Node that currently owns the volume is in the `localdisk.csi.acstor.io/selected-node` annotation, which is set after a failover, and otherwise the Node that created the volume is in the `localdisk.csi.acstor.io/selected-initial-node` volume attribute. This is the same precedence the driver's own webhook uses, so a volume that failed over to a live Node is not cleaned up because its initial Node is gone. With that driver's default `availability` failover mode a Pod can move to another Node and get a new empty volume, so consider a larger `--pvc-deletion-delay` there.

### CleanupController command line options

#### Important optional arguments that are highly recommended to be used
* `--storageclass-names`: Comma separated list of names of StorageClasses to opt-in PVs and PVCs for cleanup.
* `--csi-drivers`: Comma separated list of CSI drivers whose PVs are also eligible for cleanup, in addition to local PVs, as `<driver>[=<ref>[|<ref>...]]`. See [CSI node-local volumes](#csi-node-local-volumes). Defaults to empty, in which case only local PVs are handled.
* `--pvc-deletion-delay`: Duration, in seconds, to wait after Node deletion for PVC cleanup. Defaults to 60 seconds.
* `--stale-pv-discovery-interval`: Duration, in seconds, the Local PV Deleter should wait between tries to clean up stale PVs. Defaults to 10 seconds.

#### Other recognized arguments
* `--kubeconfig`: Absolute path to the kubeconfig file. Either this or kube-api-endpoint needs to be set if the provisioner is being run out of cluster.
* `--kube-api-endpoint`: Master URL to build a client config from. Either this or kubeconfig needs to be set if the provisioner is being run out of cluster.
* `--resync`: Duration, in minutes, of the resync interval of the controller. Defaults to 10 minutes.
* `--worker-threads`: Number of controller worker threads. Defaults to 10.
* `--listen-address`: The TCP network address where the prometheus metrics endpoint will listen. Defaults to `:8080`.
* `--metrics-path`: The HTTP path where prometheus metrics will be exposed. Defaults to "/metrics".

## Design

There are two separate routines, a **CleanupController** and a **Deleter**, running to delete stale resources:

- The [CleanupController](../pkg/node-cleanup/controller/controller.go) looks for Local Persistent Volumes that have a [NodeAffinity](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#node-affinity) to a deleted Node. When it finds such a PV, it starts a timer to wait and see if the deleted Node comes back up again. If at the end of the timer, the Node is not back up, the **PVC** bound to that PV is deleted. The PV is deleted in the next step.

    - Note: We wait to see if the Node comes back before cleaning up resources since there may be some edge cases in which a Node is deleted but comes back quickly without data loss. The wait duration is configurable.

- The [Deleter](../pkg/node-cleanup/deleter/deleter.go) looks for Local PVs with a NodeAffinity to deleted Nodes. When it finds such a PV it deletes the PV if (and only if) the PV's status is Available or if its status is Released and it has a Delete reclaim policy.

The controller manages the lifecycle of the Deleter. Further, the controller is **opt-in per StorageClass**. It takes a command line argument that specifies which StorageClasses Local PVs/PVCs must belong to in order to be cleaned up.

The cleanup controller follows the [controller](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/controllers.md) pattern and uses informers to watch for events. The controller watches for `Node` delete events and when that event occurs it uses a `PersistentVolume` lister to look for PVs (and their bound PVC) with a NodeAffinity to a deleted Node. 

The deleter runs on a specified interval and uses a `PersistentVolume` lister to find which PVs have references to deleted Nodes.