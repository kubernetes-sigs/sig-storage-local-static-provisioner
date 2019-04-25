# sig-storage-local-static-provisioner

The local volume static provisioner manages the PersistentVolume lifecycle for
pre-allocated disks by detecting and creating PVs for each local disk on the
host, and cleaning up the disks when released. It does not support dynamic
provisioning.

## Table of Contents

- [Overview](#overview)
- [User Guide](#user-guide)
  * [Getting started](#getting-started)
  * [Managing your local volumes](#managing-your-local-volumes)
  * [FAQs](#faqs)
  * [Best Practices](#best-practices)
- [Version Compatibility](#version-compatibility)
- [K8s Feature Status](#k8s-feature-status)
  * [1.14: GA](#114-ga)
  * [1.12: Beta](#112-beta)
  * [1.10: Beta](#110-beta)
  * [1.9: Alpha](#19-alpha)
  * [1.7: Alpha](#17-alpha)
  * [Future features](#future-features)
- [E2E Tests](#e2e-tests)
  * [Running](#running)
  * [View CI Results](#view-ci-results)
- [Community, discussion, contribution, and support](#community-discussion-contribution-and-support)
  * [Code of conduct](#code-of-conduct)

## Overview

Local persistent volumes allows users to access local storage through the
standard PVC interface in a simple and portable way.  The PV contains node
affinity information that the system uses to schedule pods to the correct
nodes.

An [external static provisioner](docs/provisioner.md) is provided here to help
simplify local storage management once the local volumes are configured. Note
that the local storage provisioner is different from most provisioners and does
not support dynamic provisioning.  Instead, it requires that administrators
preconfigure the local volumes on each node and if volumes are supposed to be

 1. Filesystem volumeMode (default) PVs - mount them under discovery directories.
 2. Block volumeMode PVs - create a symbolic link under discovery directory to
    the block device on the node.

The provisioner will manage the volumes under the discovery directories by creating
and cleaning up PersistentVolumes for each volume.

## User Guide

### Getting started

To get started with local static provisioning, you can follow our [getting
started guide](docs/getting-started.md) to bring up a Kubernetes cluster with
some local disks, deploy local-volume-provisioner to provision local volumes
and use PVC in your pod to request a local PV.

### Managing your local volumes

See our [operations](docs/operations.md) documentation which contains of
preparing, setting up and cleaning up local volumes on the nodes.

### Deploy provisioner with helm

See our [helm](helm/README.md) documentation for how to deploy and configure
local-volume-provisioner in Kubernetes cluster.

### FAQs

See [FAQs](docs/faqs.md).

### Best Practices

See [Best Practices](docs/best-practices.md).

## Version Compatibility

Recommended provisioner versions with Kubernetes versions

| Provisioner version | K8s version   | Reason                    |
| ------------------- | ------------- | ------------------------- |
| [2.3.2][4]          | 1.12          | fs on block support       |
| [2.2.0][3]          | 1.10          | Beta API default, block   |
| [2.0.0][2]          | 1.8, 1.9      | Mount propagation         |
| [1.0.1][1]          | 1.7           |                           |

[1]: https://github.com/kubernetes-incubator/external-storage/tree/local-volume-provisioner-v1.0.1/local-volume
[2]: https://github.com/kubernetes-incubator/external-storage/tree/local-volume-provisioner-v2.0.0/local-volume
[3]: https://github.com/kubernetes-incubator/external-storage/tree/local-volume-provisioner-v2.2.0/local-volume
[4]: https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/tree/v2.3.2

## K8s Feature Status

Also see [known issues](KNOWN_ISSUES.md) and [CHANGELOG](CHANGELOG.md).

### 1.14: GA

* No new features added

### 1.12: Beta

* Added support for automatically formatting a filesystem on the given block device in `localVolumeSource.path`

### 1.10: Beta

* New PV.NodeAffinity field added.
* **Important:** Alpha PV NodeAffinity annotation is deprecated. Users must manually update
  their PVs to use the new NodeAffinity field or run a [one-time update job](utils/update-pv-to-beta).
* Alpha: Raw block support added.

### 1.9: Alpha

* New StorageClass `volumeBindingMode` parameter that will delay PVC binding
  until a pod is scheduled.

### 1.7: Alpha

* New `local` PersistentVolume source that allows specifying a directory or mount
  point with node affinity.
* Pod using the PVC that is bound to this PV will always get scheduled to that node.

### Future features

* Local block devices as a volume source, with partitioning and fs formatting
* Dynamic provisioning for shared local persistent storage
* Local PV health monitoring, taints and tolerations
* Inline PV (use dedicated local disk as ephemeral storage)

## E2E Tests

### Running

Run `./hack/e2e.sh -h` to view help.

### View CI Results

Check testgrid [sig-storage-local-static-provisioner](https://testgrid.k8s.io/sig-storage-local-static-provisioner) dashboard.

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-dev)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

[owners]: https://git.k8s.io/community/contributors/guide/owners.md
[Creative Commons 4.0]: https://git.k8s.io/website/LICENSE
