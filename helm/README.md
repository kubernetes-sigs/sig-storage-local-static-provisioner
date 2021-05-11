# Install local-volume-provisioner with helm

Here is a [helm chart](./provisioner) for local-volume-provisioner. You can
easily generate yaml files with `helm template` or install
local-volume-provisioner in your Kubernetes with `helm install` directly.

## Table of Contents

- [Install Helm](#install-helm)
- [Custom your deployment with values file](#custom-your-deployment-with-values-file)
- [Install local-volume-provisioner](#install-local-volume-provisioner)
- [helm version < v3.0.0](#helm-version-lt-v3.0.0)
    * [Generate yaml files with `helm template` and install with `kubectl`](#generate-yaml-files-with-helm-template-and-install-with-kubectl)
    * [Install with `helm install` directly](#install-with-helm-install-directly)
- [helm version >= v3.0.0](#helm-version-gt-v3.0.0)
    * [Generate yaml files with `helm template` and install with `kubectl`](#generate-yaml-files-with-helm-template-and-install-with-kubectl1)
    * [Install with `helm install` directly](#install-with-helm-install-directly1)
- [Configurations](#configurations)
- [Examples](#examples)

## Install Helm

Please follow [official
instructions](https://helm.sh/docs/using_helm/#installing-helm) to install
`helm` client and server in your Kubernetes cluster.

Required helm version: >= 2.7.2+

## Custom your deployment with values file

Our chart provides a variety of options to configure deployment, see [provisioner/values.yaml](./provisioner/values.yaml).

And there are [a lot of examples](./helm) to help you get started quickly.

## Install local-volume-provisioner

### Generate yaml files with `helm template` and install with `kubectl`

Helm templating is used to generate the provisioner's DaemonSet, ConfigMap and
other necessary objects' specs.  The generated specs can be further customized
as needed (usually not necessary), and then deployed using kubectl.

**helm template** uses 3 sources of information:

1. Provisioner's chart templates
2. Provisioner's default values.yaml which contains variables used for rendering a template.
3. (Optional) User's customized values.yaml as a part of helm template command. User's provided
   values will override default values of Provisioner's values.yaml.

Here is basic workflow:

### helm version < v3.0.0

Install via helm template:

```console
git clone --depth=1 https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner.git
helm template ./helm/provisioner -f <path-to-your-values-file> --name <release-name> --namespace <namespace> > local-volume-provisioner.generated.yaml
edit local-volume-provisioner.generated.yaml if necessary
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
helm install ./helm/provisioner -f <path-to-your-values-file> --namespace <namespace> --name <release-name>
```

Note: set your preferred namespace and release name, e.g. `helm install ./helm/provisioner -f helm/examples/gke.yaml --namespace kube-system --name local-volume-provisioner`

Delete:

```console
helm delete --purge <release-name>
```

Upgrade: **This action will `recreate` the running pod**

**Update your custom values to match the new chart parameter**

```console
helm upgrade ./helm/provisioner -f <path-to-your-values-file> <release-name>
```

### helm version  >= v3.0.0

Install via helm template:

```console
git clone --depth=1 https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner.git
helm template -f <path-to-your-values-file> <release-name> --namespace <namespace> ./helm/provisioner > local-volume-provisioner.generated.yaml
edit local-volume-provisioner.generated.yaml if necessary
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

Please refer [helm docs](https://helm.sh/docs/using_helm/#using-helm) for more
information.

## Configurations

The following table lists the configurable parameters of the local volume
provisioner chart and their default values.

| Parameter                                    | Description                                                                                           | Type     | Default                                                    |
| ---                                          | ---                                                                                                   | ---      | ---                                                        |
| common.rbac.create                           | if `true`, create and use RBAC resources                                                              | bool     | `true`                                                     |
| common.rbac.pspEnabled                       | If `true`, create and use a restricted pod security policy                                            | bool     | `false`                                                    |
| common.serviceAccount.create                 | if `true`, create serviceaccount in .Release.Namespace                                                | bool     | `true`                                                     |
| common.serviceAccount.name                   | if set serviceaccount if the given name will be created                                               | str      | `""`                                                       |
| common.useAlphaAPI                           | If running against pre-1.10 k8s version, the `useAlphaAPI` flag must be enabled.                      | bool     | `false`                                                    |
| common.useJobForCleaning                     | If set to true, provisioner will use jobs-based block cleaning.                                       | bool     | `false`                                                    |
| common.useNodeNameOnly                       | If set to true, provisioner name will only use Node.Name and not Node.UID.                            | bool     | `false`                                                    |
| common.minResyncPeriod                       | Resync period in reflectors will be random between `minResyncPeriod` and `2*minResyncPeriod`.         | str      | `5m0s`                                                     |
| common.setPVOwnerRef                         | If set to true, PVs are set to be dependents of the owner Node.                                       | bool     | `false`                                                    |
| common.mountDevVolume                        | If set to false, the node's `/dev` path will not be mounted into containers.                          | bool     | `true`                                                     |
| common.labelsForPV                           | Map of label key-value pairs to apply to the PVs created by the provisioner.                          | map      | `-`                                                        |
| classes.[n].name                             | StorageClass name.                                                                                    | str      | `-`                                                        |
| classes.[n].hostDir                          | Path on the host where local volumes of this storage class are mounted under.                         | str      | `-`                                                        |
| classes.[n].mountDir                         | Optionally specify mount path of local volumes. By default, we use same path as hostDir in container. | str      | `-`                                                        |
| classes.[n].blockCleanerCommand              | List of command and arguments of block cleaner command.                                               | list     | `-`                                                        |
| classes.[n].volumeMode                       | Optionally specify volume mode of created PersistentVolume object. By default, we use Filesystem.     | str      | `-`                                                        |
| classes.[n].fsType                           | Filesystem type to mount. Only applies when source is block while volume mode is Filesystem.          | str      | `-`                                                        |
| classes.[n].namePattern                      | File name pattern to discover. By default, discover all file names.                                    | str      | `*`                                                        |
| classes.[n].storageClass                     | Create storage class for this class and configure it optionally.                                      | bool/map | `false`                                                    |
| classes.[n].storageClass.reclaimPolicy       | Specify reclaimPolicy of storage class, available: Delete/Retain.                                     | str      | `Delete`                                                   |
| classes.[n].storageClass.isDefaultClass      | Set storage class as default                                                                          | bool     | `false`                                                    |
| daemonset.podAnnotations                     | Annotations for each Pod in the DaemonSet.                                                            | map      | `-`                                                        |
| daemonset.podLabels                          | Labels for each Pod in the DaemonSet.                                                                 | map      | `-`                                                        |
| daemonset.image                              | Provisioner image.                                                                                    | str      | `quay.io/external_storage/local-volume-provisioner:v2.1.0` |
| daemonset.imagePullPolicy                    | Provisioner DaemonSet image pull policy.                                                              | str      | `-`                                                        |
| daemonset.priorityClassName                  | Provisioner DaemonSet Pod Priority Class name.                                                        | str      | ``                                                         |
| daemonset.kubeConfigEnv                      | Specify the location of kubernetes config file.                                                       | str      | `-`                                                        |
| daemonset.nodeLabels                         | List of node labels to be copied to the PVs created by the provisioner.                               | list     | `-`                                                        |
| daemonset.nodeSelector                       | NodeSelector constraint on nodes eligible to run the provisioner.                                     | map      | `-`                                                        |
| daemonset.tolerations                        | List of tolerations to be applied to the Provisioner DaemonSet.                                       | list     | `-`                                                        |
| daemonset.resources                          | Map of resource request and limits to be applied to the Provisioner Daemonset.                        | map      | `-`                                                 |
| daemonset.affinity                           | List of affinity to be applied to the provisioner Daemonset.                                                 | list     | `-`
| daemonset.privileged                         | If set to false, containers created by the Provisioner Daemonset will run without extra privileges.   | bool     | `true`                                                     |
| serviceMonitor.enabled                       | If set to true, Prometheus servicemonitor will be applied                                             | bool     | `false`                                                    |
| serviceMonitor.interval                      | Interval at which Prometheus scrapes the provisioner                                                  | str      | `10s`                                                      |
| serviceMonitor.namespace                     | The namespace Prometheus servicemonitor will be installed                                             | str      | `.Release.Namespace`                                       |
| serviceMonitor.additionalLabels              | Additional labels for the servicemonitor                                                              | map      | `-`                                                        |
| serviceMonitor.relabelings                   | Additional metrics relabel_config                                                                     | lists    | `-`                                                        |
| nameOverride                                 | Override default chartname                                                                            | str      | `""`                                                       |
| fullnameOverride                             | Override default releasename                                                                          | str      | `""`                                                       |

Note: `classes` is a list of objects, you can specify one or more classes.

## Examples

Here are a list of examples for various environments:

* [examples/baremetal-cleanbyjobs.yaml](examples/baremetal-cleanbyjobs.yaml)
* [examples/baremetal-resyncperiod.yaml](examples/baremetal-resyncperiod.yaml)
* [examples/baremetal-tolerations.yaml](examples/baremetal-tolerations.yaml)
* [examples/baremetal-with-resource-limits.yaml](examples/baremetal-with-resource-limits.yaml)
* [examples/baremetal-without-rbac.yaml](examples/baremetal-without-rbac.yaml)
* [examples/baremetal.yaml](examples/baremetal.yaml)
* [examples/gce-pre1.9.yaml](examples/gce-pre1.9.yaml)
* [examples/gce-retain.yaml](examples/gce-retain.yaml)
* [examples/gce.yaml](examples/gce.yaml)
* [examples/gke.yaml](examples/gke.yaml)
* [examples/eks-nvme-ssd.yaml](example/eks-nvme-ssd.yaml)
* [more...](examples/)
