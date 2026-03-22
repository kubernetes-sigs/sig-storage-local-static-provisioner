# Local Persistence Volume Static Provisioner

[![Coverage Status](https://coveralls.io/repos/github/kubernetes-sigs/sig-storage-local-static-provisioner/badge.svg?branch=master)](https://coveralls.io/github/kubernetes-sigs/sig-storage-local-static-provisioner?branch=master)

The local volume static provisioner manages PersistentVolume lifecycle for
pre-allocated disks by detecting and creating PVs for each local disk on the
host, and cleaning up the disks when released. It does not support dynamic
provisioning.

- **Project status:** GA (since Kubernetes 1.14)

## Table of Contents

- [Overview](#overview)
- [User Guide](#user-guide)
  - [Getting Started](#getting-started)
  - [Managing Your Local Volumes](#managing-your-local-volumes)
  - [Deploying](#deploying)
  - [Upgrading](#upgrading)
  - [FAQs](#faqs)
  - [Best Practices](#best-practices)
- [Version Compatibility](#version-compatibility)
- [Feature Status](#feature-status)
- [E2E Tests](#e2e-tests)
- [Community, Discussion, Contribution, and Support](#community-discussion-contribution-and-support)

## Overview

Local persistent volumes allow users to access local storage through the
standard PVC interface in a simple and portable way. The PV contains node
affinity information that the system uses to schedule pods to the correct
nodes.

An [external static provisioner](docs/provisioner.md) is provided here to help
simplify local storage management once the local volumes are configured. Note
that the local storage provisioner is different from most provisioners and does
not support dynamic provisioning. Instead, it requires that administrators
preconfigure the local volumes on each node and if volumes are supposed to be:

1. **Filesystem volumeMode** (default) PVs — mount them under discovery directories.
2. **Block volumeMode** PVs — create a symbolic link under the discovery directory to
   the block device on the node.

The provisioner will manage the volumes under the discovery directories by creating
and cleaning up PersistentVolumes for each volume.

> [!NOTE]
> When the node hosting a local PV is deleted, the data is likely lost, but the PV object still exists — the system will indefinitely try to schedule the Pod to the deleted node. See the [local volume node cleanup](docs/node-cleanup-controller.md) documentation for how to make your workloads automatically recover from node deletion.

## User Guide

### Getting Started

To get started with local static provisioning, follow the [getting started guide](docs/getting-started.md) to bring up a Kubernetes cluster with some local disks, deploy local-volume-provisioner to provision local volumes, and use PVC in your pod to request a local PV.

### Managing Your Local Volumes

See the [operations](docs/operations.md) documentation for preparing, setting up, and cleaning up local volumes on the nodes.

### Deploying

See the [Helm](helm/README.md) documentation for how to deploy and configure local-volume-provisioner in a Kubernetes cluster with Helm.

If you want to manage the provisioner with plain YAML files, you can refer to the [example YAMLs](deployment/kubernetes/example). [Helm generated YAMLs](helm/generated_examples/) are good sources of examples too. See [provisioner configuration](docs/provisioner.md#configuration) for a full explanation of all options.

### Upgrading

See the [upgrading](docs/upgrading.md) documentation for how to upgrade the provisioner version or update configuration in a Kubernetes cluster.

### FAQs

See [FAQs](docs/faqs.md).

### Best Practices

See [Best Practices](docs/best-practices.md).

## Version Compatibility

Recommended provisioner versions with Kubernetes versions:

| Provisioner Version | K8s Version |
|---------------------|-------------|
| [2.7.0][7]          | 1.21+       |
| [2.6.0][6]          | 1.12+       |
| [2.5.0][5]          | 1.12+       |

[5]: https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/tree/v2.5.0
[6]: https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/tree/v2.6.0
[7]: https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/tree/v2.7.0

## Feature Status

Also see [known issues](KNOWN_ISSUES.md) and [CHANGELOG](CHANGELOG.md).

### Future Features

- Local block devices as a volume source, with partitioning and filesystem formatting
- Dynamic provisioning for shared local persistent storage
- Local PV health monitoring, taints and tolerations
- Inline PV (use dedicated local disk as ephemeral storage)

## E2E Tests

### Running

Run `./hack/e2e.sh -h` to view help.

### CI Results

- TestGrid [sig-storage-local-static-provisioner](https://testgrid.k8s.io/sig-storage-local-static-provisioner) dashboard.
- Image build pipeline: [post-sig-storage-local-static-provisioner-push-images](https://testgrid.k8s.io/sig-storage-image-build#post-sig-storage-local-static-provisioner-push-images)

## Community, Discussion, Contribution, and Support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-dev)

### Code of Conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
