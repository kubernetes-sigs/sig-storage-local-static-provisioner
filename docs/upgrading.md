# Upgrading

This document provides guides on how to upgrade provisioner version or update
configuration in Kubernetes cluster.

## Table of Contents

- [Upgrading provisioner version only](#upgrading-provisioner-version-only)
- [Updating provisioner configuration](#updating-provisioner-configuration)
  * [Limitations on updating configuration](#limitations-on-updating-configuration)
    + [useNodeNameOnly](#usenodenameonly)
    + [storageClassMap](#storageclassmap)
  * [How to update](#how-to-update)

## Upgrading provisioner version only

At first, you must take a look at [change logs](/CHANGELOG.md) between current version and the
version you are upgrading to. If there are some ACTION REQUIRED items, you must
follow them in upgrading.

Then, you can update the image tag and deploy the provisioner again.

## Updating provisioner configuration

*WARNING* Not all configs cannot be changed when PVs already been provisioned
in production. Please fully understand all fields in provisioner
[configuration](/docs/provisioner.md#configuration) and limitations in
here. We're trying to remove limitations as possible as we can.

### Limitations on updating configuration

#### useNodeNameOnly

`useNodeNameOnly` will change provisioner name which is set in PV annotations
`pv.kubernetes.io/provisioned-by`. The key is used to select PVs the
provisioner managed. So if you changed this, you need to update the value of
current PVs to the new provisioner name.

The provisioner name is

- `local-volume-provisioner-<node-name>` if `useNodeNameOnly` is true
- `local-volume-provisioner-<node-name>-<node-uid>` if `useNodeNameOnly` is false

#### storageClassMap

In an existing configuration of storage class, only a few fields can be
changed.

- `mountDir` can be change if you changed mount path in provisioner pod spec. 
- `blockCleanerCommand` is safe to change.

However, it's safe to add a new storage class.

### How to update

Currently, the provisioner does not reload configuration automatically. When
you finish updating the ConfigMap, you must delete the pods of provisioner
DaemonSet to take effect.

Note that if you add new discovery directory in provisioner configuration, you
must update provisioner pod template spec too. This is not necessary if you
deploy provisioner with our [helm chart](/helm/provisioner).
