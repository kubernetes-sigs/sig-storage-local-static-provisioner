# local-volume-configurator

local-volume-configurator is used to prepare and setup local volumes according
to the user-provided configuration on the node it runs on.

## Table of Contents

- [Motivation](#motivation)
- [Goals](#goals)
  * [Non-Goals](#non-goals)
- [Design](#design)
  * [Architectures](#architectures)
    + [As init container](#as-init-container)
    + [As long-running container](#as-long-running-container)
  * [Configuration syntax](#configuration-syntax)
    + [Storage](#storage)
    + [Discovery](#discovery)
  * [How to configure](#how-to-configure)
    + [metadata server](#metadata-server)
    + [node annotations](#node-annotations)
- [Use Cases](#use-cases)
  * [Format and mount multiple local SSD devices into a single logical volume in GKE](#format-and-mount-multiple-local-ssd-devices-into-a-single-logical-volume-in-gke)
- [Alternatives](#alternatives)
  * [Cloud startup-script](#cloud-startup-script)
  * [Implement as an operator](#implement-as-an-operator)
- [Plan](#plan)
  * [Phase 1 Implemented as init container and support raw/raid0](#phase-1-implemented-as-init-container-and-support-rawraid0)

## Motivation

There are various kinds of local volumes we can have on a node, e.g. device,
filesystem volume or shared filesystem volume. Each volume may backed by
different storage media, e.g. raw device, raid array or LVM logic volume.

In large clusters, it is repetitive and error-prone to prepare local disks and
setup volumes before the provisioner provision them for Kubernetes.

These tasks are automatable, we can provide a configurator to eliminate these
repetitive tasks.

## Goals

- Automate local volume setup
- Designed for scripting and API programming

### Non-Goals

- Work with other local volume setup (e.g. local volumes created manually)
- Tear down local volumes
  - For safety, we don't tear down local volumes. In cloud platform, you can
    delete or recreate the node to start over.
- Wipe the old filesystem
  - Sometimes, the disks contains old filesystem which should be wiped before
    preparation. It's the resposibility of the user to clean up old
    filesystems.
- Human-friendly interface

## Design

Basically, there are two stages to configure the local volumes:

- prepare the disks
  - use raw device
  - combine the disks with raid
  - combine the disks with lvm
  - separate the disk into multiple partitions
  - ...
- setup the local volumes
  - use device directly
  - format the disk and use the whole filesystem as one PV
  - format the disk and share the filesystem with multiple PVs

### Architectures

#### As init container

#### As long-running container

### Configuration syntax

Each rule consists of a storage field and a discovery field, each of them is a
object.

```
{
    "storage": {
        // storage fields
    },
    "discovery": {
        // discovery fields
    }
}
```

#### Storage

Storage object consists of storage-specific fields which describe how to
prepare storage, e.g. raiding or partitioning disks.

Examples:

1) Use raw devices /dev/sdb, /dev/sdc directly

```
{
    "provider": "raw",
    "disks": {
        "/dev/sdb",
        "/dev/sdc"
    }
}
```

2) Combines raw devices /dev/sdb, /dev/sdb with raid0

```
{
    "type": "raid0",
    "disks": {
        "/dev/sdb",
        "/dev/sdc"
    }
}
```

#### Discovery

Discovery object describes how to setup local volumes in discovery directory.

Examples:

1) For each block device from storage provider, link it into discovery directory

```
{
    "dir": "/mnt/blocks",
    "mode": "block"
}
```

2) For each block device, format and mount the filesystem in `/mnt/disks`

```
{
    "dir": "/mnt/disks",
    "mode": "filesystem",
    "fsType": "ext4"
}
```

3) For each block device, format, mount and share by creating self-bind-mounted
directories under it.

```
{
    "dir": "/mnt/disks",
    "mode": "filesystem",
    "fsType": "ext4",
    "shares": 10
}
```

By default, configurator will follow our [best practices](best-practices.md) to
setup local volumes, e.g. use UUID in volume path.

### How to configure

#### metadata server

In cloud environments, it's best to configure node-specific data in cloud
metadata server (e.g. [GCP metadata
server](https://cloud.google.com/compute/docs/storing-retrieving-metadata#custom).

The reserved key is `local-volume-configurator`.

#### node annotations

When metadata server is not available (e.g. non-cloud environments), it is
convenient to configure node-specific data in the node annotations.

The reserved key is `configurator.local.storage.k8s.io`.

The value is a JSON array which contains a list of rules.

```
[
  {
    // rule 1
  },
  {
    // rule 2
  },
  // ...
]
```

## Use Cases

### Format and mount multiple local SSD devices into a single logical volume in GKE

On GCP/GKE, local SSDs have a fixed 375 GB capacity for each device that you
attach to the instance. If you want to combine multiple local SSD devices into
a single logical volume, [you can setup
manually](https://cloud.google.com/compute/docs/disks/local-ssd#formatmultiple)
or with this configurator automatically.

For example, by adding this rule local-volume-configurator will combine
the disks, format and mount it into discovery directory automatically.

```
{
    "storage": {
        "provider": "raid0",
        "disks": {
            "/dev/disk/by-id/google-local-ssd-0",
            "/dev/disk/by-id/google-local-ssd-1"
        }
    },
    "discovery": {
        "dir": "/mnt/disks",
        "mode": "filesystem",
        "fsType": "ext4"
    }
}
```

Note that in GKE, with `--local-ssd-count` flag, local SSDs are formatted and
mounted at `/mnt/disks/ssdX` automatically. You need to unmount and wipe old
filesystems. You can configure an extra init container before configurator to
cleanup automatically. Here is [an example](TODO).

## Alternatives

### Cloud startup-script

In most cloud environments, platform provides a startup script which can be
executed to provision the node. But it is hard to be a universal solution and
in some environments, startup-script is not available.

### Implement as an operator

Before we implement an operator to configure local volumes for us. We need
abstract the local volume setup work and implement a basic utility to do the
basic work.

## Plan

### Phase 1 Run in init container and read rules from metadata server in GKE

It's easy to prepare local volumes in managed clusters, so at the beginning we
implement it as a init container of provisioner and setup local volumes once
before provisioner to discovery local volumes.
