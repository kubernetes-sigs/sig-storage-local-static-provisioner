# Local Persistence Volume Static Provisioner

[![Coverage Status](https://coveralls.io/repos/github/kubernetes-sigs/sig-storage-local-static-provisioner/badge.svg?branch=master)](https://coveralls.io/github/kubernetes-sigs/sig-storage-local-static-provisioner?branch=master)

The local volume static provisioner manages PersistentVolume lifecycle for
pre-allocated disks by detecting and creating PVs for each local disk on the
host, and cleaning up the disks when released. It does not support dynamic
provisioning.

## Table of Contents

- [Overview](#overview)
- [User Guide](#user-guide)
  * [Getting started](#getting-started)
  * [Managing your local volumes](#managing-your-local-volumes)
  * [Deploying](#deploying)
  * [Upgrading](#upgrading)
  * [FAQs](#faqs)
  * [Best Practices](#best-practices)
- [Version Compatibility](#version-compatibility)
- [K8s Feature Status](#k8s-feature-status)
  * GA from 1.14
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

A caveat to scheduling a Pod on the same node as its local PV is that when the node hosting the PV is deleted, while the data is likely lost, the PV object still exists and therefore the system is indefinitely trying to schedule the Pod to a deleted node. See our [local volume node cleanup](docs/node-cleanup-controller.md) documentation which contains information on how to make your workloads automatically recover from node deletion.

## User Guide

### Getting started

To get started with local static provisioning, you can follow our [getting
started guide](docs/getting-started.md) to bring up a Kubernetes cluster with
some local disks, deploy local-volume-provisioner to provision local volumes
and use PVC in your pod to request a local PV.

### Managing your local volumes

See our [operations](docs/operations.md) documentation which contains of
preparing, setting up and cleaning up local volumes on the nodes.

### Deploying

See our [helm](helm/README.md) documentation for how to deploy and configure
local-volume-provisioner in Kubernetes cluster with helm.

If you want to manage provisioner with plain YAML files, you can refer to our
[example yamls](deployment/kubernetes/example). [helm generated
yamls](helm/generated_examples/) are good sources of examples too.
[Here](docs/provisioner.md#configuration) is a full explanation of provisioner
configuration.

### Upgrading

See our [upgrading](docs/upgrading.md) documentation for how to upgrade
provisioner version or update configuration in Kubernetes cluster.

### FAQs

See [FAQs](docs/faqs.md).

### Best Practices

See [Best Practices](docs/best-practices.md).

## Version Compatibility

Recommended provisioner versions with Kubernetes versions

| Provisioner version | K8s version   |
| ------------------- | ------------- |
| [2.7.0][7]          | 1.21+         |
| [2.6.0][6]          | 1.12+         |
| [2.5.0][5]          | 1.12+         |

[5]: https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/tree/v2.5.0
[6]: https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/tree/v2.6.0
[7]: https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/tree/v2.7.0


## K8s Feature Status

Also see [known issues](KNOWN_ISSUES.md) and [CHANGELOG](CHANGELOG.md).

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
