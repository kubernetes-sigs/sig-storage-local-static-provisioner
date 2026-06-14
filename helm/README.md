# Install local-volume-provisioner with helm

Here is a [helm chart](./provisioner) for local-volume-provisioner. You can
easily generate yaml files with `helm template` or install
local-volume-provisioner in your Kubernetes with `helm install` directly.

## Table of Contents

- [Install local-volume-provisioner with helm](#install-local-volume-provisioner-with-helm)
  - [Table of Contents](#table-of-contents)
  - [Install Helm](#install-helm)
  - [Customize your deployment with values file](#customize-your-deployment-with-values-file)
  - [Install local-volume-provisioner](#install-local-volume-provisioner)
    - [Generate yaml files with `helm template` and install with `kubectl`](#generate-yaml-files-with-helm-template-and-install-with-kubectl)
    - [Install using helm repo](#install-using-helm-repo)
    - [Install with `helm install` directly](#install-with-helm-install-directly)
  - [Discovery Directory and Storage Classes](#discovery-directory-and-storage-classes)
  - [Configurations](#configurations)
  - [Examples](#examples)

## Install Helm

Please follow [official
instructions](https://helm.sh/docs/intro/install/) to install `helm`.

## Customize your deployment with values file

Our chart provides a variety of options to configure deployment, see [provisioner/values.yaml](./provisioner/values.yaml).

And there are [a lot of examples](./examples) to help you get started quickly.

## Install local-volume-provisioner

### Generate yaml files with `helm template` and install with `kubectl`

Helm templating is used to generate the provisioner's DaemonSet, ConfigMap and
other necessary objects' specs.  The generated specs can be further customized
as needed (usually not necessary), and then deployed using kubectl.

Here is basic workflow:

### Install using helm repo

Install by adding the repo as a Helm repo:

```sh
helm repo add sig-storage-local-static-provisioner https://kubernetes-sigs.github.io/sig-storage-local-static-provisioner
helm template --debug sig-storage-local-static-provisioner/local-static-provisioner --version <version> --namespace <namespace> > local-volume-provisioner.generated.yaml
# edit local-volume-provisioner.generated.yaml if necessary
kubectl create -f local-volume-provisioner.generated.yaml
```

Or install by cloning the repo locally:

```console
git clone --depth=1 https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner.git
helm template -f <path-to-your-values-file> <release-name> --namespace <namespace> ./helm/provisioner > local-volume-provisioner.generated.yaml
# edit local-volume-provisioner.generated.yaml if necessary
kubectl create -f local-volume-provisioner.generated.yaml
```

Delete:

```console
kubectl delete -f local-volume-provisioner.generated.yaml
```

Upgrade:

**Update your custom values to match the new chart parameter**

```console
# Teardown the old provisioner
kubectl delete -f local-volume-provisioner.generated.yaml

# Update your custom values to match the new chart parameter

# Apply the new generated.yaml
kubectl create -f local-volume-provisioner.generated.yaml
```

### Install with `helm install` directly

Helm provides an easy interface to install applications and sources into
Kubernetes cluster. You can install local-volume-provisioner with `helm
install` command directly. Here is basic workflow:

Install:

```console
helm install -f <path-to-your-values-file> <release-name> --namespace <namespace> ./helm/provisioner
```

Note: set your preferred namespace and release name, e.g. `helm install -f helm/examples/gke.yaml local-volume-provisioner --namespace kube-system ./helm/provisioner`

Delete:

```console
helm uninstall <release-name> --namespace
```

Upgrade: **This action will `recreate` the running pod**

**Update your custom values to match the new chart parameter**

```console
helm upgrade --reset-value -f <path-to-your-values-file> <release-name> --namespace <namespace> ./helm/provisioner
```

`--reset-values` will reset custom values to the values from the new chart version. `-f` apply the custom values file on top.

Please refer [helm docs](https://helm.sh/docs/intro/using_helm/) for more
information.

## Discovery Directory and Storage Classes

The provisioner discovers local volumes by scanning **discovery directories** on
each node. Each storage class maps to one discovery directory via the `hostDir`
field in the `classes` list. The provisioner discovers:

- **Mount points** for Filesystem mode volumes
- **Symbolic links** to block devices (for Block mode volumes, or for
  Filesystem mode when you want Kubernetes to format a raw device with `fsType`)

For directory-based local volumes, they must be **bind-mounted** into the
discovery directory.

### Basic setup: one disk per PV

Mount each disk into the discovery directory:

```bash
sudo mkfs.ext4 /dev/sdb
DISK_UUID=$(sudo blkid -s UUID -o value /dev/sdb)
sudo mkdir -p /mnt/disks/$DISK_UUID
sudo mount -t ext4 /dev/sdb /mnt/disks/$DISK_UUID
```

Then configure the Helm values:

```yaml
classes:
  - name: local-storage
    hostDir: /mnt/disks
    volumeMode: Filesystem
    fsType: ext4
    storageClass: true
```

### Sharing a disk filesystem by multiple PVs

You can split a single disk into multiple PVs by creating subdirectories and
bind-mounting them into the discovery directory. The provisioner discovers each
bind mount as a separate PV.

On the node:

```bash
# Format and mount the disk (NOT into the discovery directory)
sudo mkfs.ext4 /dev/sdb
DISK_UUID=$(sudo blkid -s UUID -o value /dev/sdb)
sudo mkdir -p /mnt/$DISK_UUID
sudo mount -t ext4 /dev/sdb /mnt/$DISK_UUID

# Create subdirectories and bind-mount them into the discovery directory
for i in $(seq 1 10); do
  sudo mkdir -p /mnt/${DISK_UUID}/vol${i} /mnt/disks/${DISK_UUID}_vol${i}
  sudo mount --bind /mnt/${DISK_UUID}/vol${i} /mnt/disks/${DISK_UUID}_vol${i}
done
```

The Helm values are the same — `hostDir` points to the discovery directory:

```yaml
classes:
  - name: local-storage
    hostDir: /mnt/disks
    volumeMode: Filesystem
    fsType: ext4
    storageClass: true
```

Each bind mount under `/mnt/disks` is discovered as a separate PV.

### Block mode volumes

For block devices, create symbolic links in the discovery directory using the
stable device path:

```bash
sudo ln -s /dev/disk/by-id/<unique-disk-id> /mnt/disks/
```

```yaml
classes:
  - name: local-block
    hostDir: /mnt/disks
    volumeMode: Block
    storageClass: true
```

### Key points

- `hostDir` is the discovery directory path on the host node
- **Discovery only scans the first level** of `hostDir` — it does not recurse
  into subdirectories. All mount points, bind mounts, or symlinks must appear
  directly under `hostDir`.
- `mountDir` (optional) overrides the mount path inside the container; defaults
  to the same value as `hostDir`
- Each storage class must have its own unique discovery directory
- Set `storageClass: true` (or configure it as a map) to have Helm create the
  StorageClass object automatically; otherwise you must create it separately
- **Mounts must be persistent across reboots.** Add entries to `/etc/fstab` for
  all mounts and bind mounts. Without persistent mount entries, PV discovery
  will be lost after a node restart.
- See [operations.md](../docs/operations.md) for full details on preparing
  local volumes

## Configurations

The following table lists the configurable parameters of the local volume
provisioner chart and their default values.

| Parameter                               | Description                                                                                                                    | Type     | Default                                                       |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ | -------- | ------------------------------------------------------------- |
| nameOverride                            | Override default chartname                                                                                                     | str      | `""`                                                          |
| fullnameOverride                        | Override default releasename                                                                                                   | str      | `""`                                                          |
| rbac.create                             | if `true`, create and use RBAC resources                                                                                       | bool     | `true`                                                        |
| serviceAccount.create                   | if `true`, create serviceaccount in .Release.Namespace                                                                         | bool     | `true`                                                        |
| serviceAccount.name                     | if set serviceaccount if the given name will be created                                                                        | str      | `""`                                                          |
| useJobForCleaning                       | If set to true, provisioner will use jobs-based block cleaning.                                                                | bool     | `false`                                                       |
| useNodeNameOnly                         | If set to true, provisioner name will only use Node.Name and not Node.UID.                                                     | bool     | `false`                                                       |
| minResyncPeriod                         | Resync period in reflectors will be random between `minResyncPeriod` and `2*minResyncPeriod`.                                  | str      | `5m0s`                                                        |
| setPVOwnerRef                           | If set to true, PVs are set to be dependents of the owner Node.                                                                | bool     | `false`                                                       |
| additionalVolumes                       | Additional volumes to create, for the default container and init containers to consume.                                        | list     | `-`                                                           |
| mountDevVolume                          | If set to false, the node's `/dev` path will not be mounted into containers.                                                   | bool     | `true`                                                        |
| additionalVolumeMounts                  | Additional volumes to mount to the default container, the volumes should either be host paths or defined by additionalVolumes. | list     | `-`                                                           |
| labelsForPV                             | Map of label key-value pairs to apply to the PVs created by the provisioner.                                                   | map      | `-`                                                           |
| enableWindows                           | If `true`, Windows DaemonSet will be created by the provisioner.                                                               | bool     | `false`                                                       |
| classes.[n].name                        | StorageClass name.                                                                                                             | str      | `-`                                                           |
| classes.[n].hostDir                     | Path on the host where local volumes of this storage class are mounted under.                                                  | str      | `-`                                                           |
| classes.[n].mountDir                    | Optionally specify mount path of local volumes. By default, we use same path as hostDir in container.                          | str      | `-`                                                           |
| classes.[n].blockCleanerCommand         | List of command and arguments of block cleaner command.                                                                        | list     | `-`                                                           |
| classes.[n].volumeMode                  | Optionally specify volume mode of created PersistentVolume object. By default, we use Filesystem.                              | str      | `-`                                                           |
| classes.[n].fsType                      | Filesystem type to mount. Only applies when source is block while volume mode is Filesystem.                                   | str      | `-`                                                           |
| classes.[n].namePattern                 | File name pattern to discover. By default, discover all file names.                                                            | str      | `*`                                                           |
| classes.[n].storageClass                | Create storage class for this class and configure it optionally.                                                               | bool/map | `false`                                                       |
| classes.[n].storageClass.reclaimPolicy  | Specify reclaimPolicy of storage class, available: Delete/Retain.                                                              | str      | `Delete`                                                      |
| classes.[n].storageClass.isDefaultClass | Set storage class as default                                                                                                   | bool     | `false`                                                       |
| classes.[n].storageClass.provisioner    | Specify provisioner of storage class.                                                                                          | str      | `kubernetes.io/no-provisioner`                                |
| podAnnotations                          | Annotations for each Pod in the DaemonSet.                                                                                     | map      | `-`                                                           |
| podLabels                               | Labels for each Pod in the DaemonSet.                                                                                          | map      | `-`                                                           |
| hostPID                                 | Host PID set in the linux daemonset container spec. When set to true allows a pod to have access to the host process ID namespace | bool     | `false`                                                       |
| image                                   | Provisioner image.                                                                                                             | str      | `registry.k8s.io/sig-storage/local-volume-provisioner:v2.7.0` |
| imagePullPolicy                         | Provisioner DaemonSet image pull policy.                                                                                       | str      | `-`                                                           |
| imagePullSecrets                        | Provisioner image pull secrets.                                                                                                | list     | `-`                                                           |
| priorityClassName                       | Provisioner DaemonSet Pod Priority Class name.                                                                                 | str      | ``                                                            |
| kubeConfigEnv                           | Specify the location of kubernetes config file.                                                                                | str      | `-`                                                           |
| nodeLabels                              | List of node labels to be copied to the PVs created by the provisioner.                                                        | list     | `-`                                                           |
| nodeSelector                            | NodeSelector constraint on nodes eligible to run the provisioner.                                                              | map      | `-`                                                           |
| tolerations                             | List of tolerations to be applied to the Provisioner DaemonSet.                                                                | list     | `-`                                                           |
| resources                               | Map of resource request and limits to be applied to the Provisioner Daemonset.                                                 | map      | `-`                                                           |
| affinity                                | List of affinity to be applied to the provisioner Daemonset.                                                                   | list     | `-`                                                           |
| privileged                              | If set to false, containers created by the Provisioner Daemonset will run without extra privileges.                            | bool     | `true`                                                        |
| initContainers                          | Init containers.                                                                                                               | list     | `-`                                                           |
| serviceMonitor.enabled                  | If set to true, Prometheus servicemonitor will be applied                                                                      | bool     | `false`                                                       |
| serviceMonitor.interval                 | Interval at which Prometheus scrapes the provisioner                                                                           | str      | `10s`                                                         |
| serviceMonitor.namespace                | The namespace Prometheus servicemonitor will be installed                                                                      | str      | `.Release.Namespace`                                          |
| serviceMonitor.additionalLabels         | Additional labels for the servicemonitor                                                                                       | map      | `-`                                                           |
| serviceMonitor.relabelings              | Additional metrics relabel_config                                                                                              | lists    | `-`                                                           |

Note: `classes` is a list of objects, you can specify one or more classes.

## Examples

Here are a list of examples for various environments:

* [examples/baremetal-cleanbyjobs.yaml](examples/baremetal-cleanbyjobs.yaml)
* [examples/baremetal-resyncperiod.yaml](examples/baremetal-resyncperiod.yaml)
* [examples/baremetal-tolerations.yaml](examples/baremetal-tolerations.yaml)
* [examples/baremetal-provisioner.yaml](examples/baremetal-provisioner.yaml)
* [examples/baremetal-with-resource-limits.yaml](examples/baremetal-with-resource-limits.yaml)
* [examples/baremetal-without-rbac.yaml](examples/baremetal-without-rbac.yaml)
* [examples/baremetal.yaml](examples/baremetal.yaml)
* [examples/gce-pre1.9.yaml](examples/gce-pre1.9.yaml)
* [examples/gce-retain.yaml](examples/gce-retain.yaml)
* [examples/gce.yaml](examples/gce.yaml)
* [examples/gke.yaml](examples/gke.yaml)
* [examples/eks-nvme-ssd.yaml](example/eks-nvme-ssd.yaml)
* [more...](examples/)
