# Cleanup Controller

The cleanup controller removes PersistentVolumes and PersistentVolumeClaims that reference deleted Nodes.

## Overview

Pods using a Local Persistent Volume are always scheduled to the same Node as the Local PV it uses (as opposed to if they were using a HostPath volume, for instance). When Nodes fail while having a Local PV attached to them, Pods using the Local PV become stuck since they can't be scheduled to a deleted Node. In other words, the Local PV and its corresponding PVC aren’t cleaned up when a Node becomes unavailable and so each of them (PV/PVC/Pods) becomes stuck. This results in a degraded or unavailable workload. This controller aims to clean up the stale Local PVs and their bound PVCs after Node deletion, allowing workloads to automatically recover.

## Design

There are two separate processes, a **CleanupController** and a **Deleter**, running to delete stale resources:

- The [CleanupController](../../pkg/cleanup/controller/controller.go) looks for Local Persistent Volumes that have a [NodeAffinity](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#node-affinity) to a deleted Node. When it finds such a PV, it starts a timer to wait and see if the deleted Node comes back up again. If at the end of the timer, the Node is not back up, the **PVC** bound to that PV is deleted. The PV is deleted in the next step.

    - Note: We wait to see if the Node comes back before cleaning up resources since there may be some edge cases in which a Node is deleted but comes back quickly without data loss. The wait duration is configurable.

- The [Deleter](../../pkg/cleanup/deleter/deleter.go) looks for Local PVs with a NodeAffinity to deleted Nodes. When it finds such a PV it deletes the PV if (and only if) the PV's status is Available or if its status is Released and it has a Delete reclaim policy.

The controller manages the lifecycle of the Deleter. Further, the controller is **opt-in per StorageClass**. It takes a command line argument that specifies which StorageClasses Local PVs/PVCs must belong to in order to be cleaned up.

The cleanup controller follows the [controller](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/controllers.md) pattern and uses informers to watch for events. The controller watches for `Node` delete events and when that event occurs it uses a `PersistentVolume` lister to look for PVs (and their bound PVC) with a NodeAffinity to a deleted Node. 

The deleter runs on a specified interval and uses a `PersistentVolume` lister to find which PVs have references to deleted Nodes.

## Usage

Please see the example [deployment](./deployment.yaml) and [rbac](./rbac.yaml) for deploying the controller.

### CleanupController command line options

#### Important optional arguments that are highly recommended to be used
* `--storageclass-names`: Comma separated list of names of StorageClasses to opt-in PVs and PVCs for cleanup.
* `--pvc-deletion-delay`: Duration, in seconds, to wait after Node deletion for PVC cleanup. Defaults to 60 seconds.
* `--stale-pv-discovery-interval`: Duration, in seconds, the Local PV Deleter should wait between tries to clean up stale PVs. Defaults to 10 seconds.

#### Other recognized arguments
* `--kubeconfig`: Absolute path to the kubeconfig file. Either this or kube-api-endpoint needs to be set if the provisioner is being run out of cluster.
* `--kube-api-endpoint`: Master URL to build a client config from. Either this or kubeconfig needs to be set if the provisioner is being run out of cluster.
* `--resync`: Duration, in minutes, of the resync interval of the controller. Defaults to 10 minutes.
* `--worker-threads`: Number of controller worker threads. Defaults to 10.