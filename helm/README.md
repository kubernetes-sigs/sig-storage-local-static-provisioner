# Install local-volume-provisioner with helm

Here is a [helm chart](./provisioner) for local-volume-provisioner. You can
easily generate yaml files with `helm template` or install
local-volume-provisioner in your Kubernetes with `helm install` directly.

## Table of Contents

- [Install Helm](#install-helm)
- [Custom your deployment with values file](#custom-your-deployment-with-values-file)
- [Install local-volume-provisioner](#install-local-volume-provisioner)
  * [Generate yaml files with `helm template` and install with `kubectl`](#generate-yaml-files-with-helm-template-and-install-with-kubectl)
  * [Install with `helm install` directly](#install-with-helm-install-directly)
- [Configurations](#configurations)
- [Examples](#examples)

## Install Helm

Please follow [official
instructions](https://helm.sh/docs/using_helm/#installing-helm) to install
`helm` client and server in your Kubernetes cluster.

Required helm version: >= 2.7.2+, < 3.0.0

## Custom your deployment with values file

Our chart provides a variety of options to configure deployment, see [a
full list of them](TODO).

And there are [a lot of examples](TODO) to help you get started quickly.

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

Install:

```console
$ git clone --depth=1 https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner.git
$ helm template ./helm/provisioner -f <path-to-your-values-file> > local-volume-provisioner.generated.yaml
# edit local-volume-provisioner.generated.yaml if necessary
$ kubectl create -f local-volume-provisioner.generated.yaml
```

Delete:

```console
$ kubectl delete -f local-volume-provisioner.generated.yaml
```

### Install with `helm install` directly

Helm provides an easy interface to install applications and sources into
Kubernetes cluster. You can install local-volume-provisioner with `helm
install` command directly. Here is basic workflow:

Install:

```console
$ helm install ./helm/provisioner -f <path-to-your-values-file> --namespace <namespace> --name <release-name>
```

Note: set your preferred namespace and release name, e.g. `helm install ./helm/provisioner -f helm/examples/gke.yaml --namespace kube-system --name local-volume-provisioner`

Delete:

```
$ helm delete --purge <release-name>
```

Please refer [helm docs](https://helm.sh/docs/using_helm/#using-helm) for more
information.

## Configurations

The following table lists the configurable parameters of the local volume
provisioner chart and their default values.

| Parameter                                    | Description                                                                                           | Type     | Default                                                    |
| ---                                          | ---                                                                                                   | ---      | ---                                                        |
| common.rbac                                  | Generating RBAC (Role Based Access Control) objects.                                                  | bool     | `true`                                                     |
| common.namespace                             | Namespace where provisioner runs.                                                                     | str      | `default`                                                  |
| common.createNamespace                       | Whether to create namespace for provisioner.                                                          | bool     | `false`                                                    |
| common.useAlphaAPI                           | If running against pre-1.10 k8s version, the `useAlphaAPI` flag must be enabled.                      | bool     | `false`                                                    |
| common.useJobForCleaning                     | If set to true, provisioner will use jobs-based block cleaning.                                       | bool     | `false`                                                    |
| common.useNodeNameOnly                       | If set to true, provisioner name will only use Node.Name and not Node.UID.                            | bool     | `false`                                                    |
| common.minResyncPeriod                       | Resync period in reflectors will be random between `minResyncPeriod` and `2*minResyncPeriod`.         | str      | `5m0s`                                                     |
| common.configMapName                         | Provisioner ConfigMap name.                                                                           | str      | `local-provisioner-config`                                 |
| common.podSecurityPolicy                     | Whether to create pod security policy or not.                                                         | bool     | `false`                                                    |
| common.setPVOwnerRef                         | If set to true, PVs are set to be dependents of the owner Node.                                       | bool     | `false`                                                    |
| classes.[n].name                             | StorageClass name.                                                                                    | str      | `-`                                                        |
| classes.[n].hostDir                          | Path on the host where local volumes of this storage class are mounted under.                         | str      | `-`                                                        |
| classes.[n].mountDir                         | Optionally specify mount path of local volumes. By default, we use same path as hostDir in container. | str      | `-`                                                        |
| classes.[n].blockCleanerCommand              | List of command and arguments of block cleaner command.                                               | list     | `-`                                                        |
| classes.[n].volumeMode                       | Optionally specify volume mode of created PersistentVolume object. By default, we use Filesystem.     | str      | `-`                                                        |
| classes.[n].fsType                           | Filesystem type to mount. Only applies when source is block while volume mode is Filesystem.          | str      | `-`                                                        |
| classes.[n].storageClass                     | Create storage class for this class and configure it optionally.                                      | bool/map | `false`                                                    |
| classes.[n].storageClass.reclaimPolicy       | Specify reclaimPolicy of storage class, available: Delete/Retain.                                     | str      | `Delete`                                                   |
| classes.[n].storageClass.isDefaultClass      | Set storage class as default                                                                          | bool     | `false`                                                    |
| daemonset.name                               | Provisioner DaemonSet name.                                                                           | str      | `local-volume-provisioner`                                 |
| daemonset.image                              | Provisioner image.                                                                                    | str      | `quay.io/external_storage/local-volume-provisioner:v2.1.0` |
| daemonset.imagePullPolicy                    | Provisioner DaemonSet image pull policy.                                                              | str      | `-`                                                        |
| daemonset.serviceAccount                     | Provisioner DaemonSet service account.                                                                | str      | `local-storage-admin`                                      |
| daemonset.priorityClassName                  | Provisioner DaemonSet Pod Priority Class name.                                                        | str      | ``                                                         |
| daemonset.kubeConfigEnv                      | Specify the location of kubernetes config file.                                                       | str      | `-`                                                        |
| daemonset.nodeLabels                         | List of node labels to be copied to the PVs created by the provisioner.                               | list     | `-`                                                        |
| daemonset.nodeSelector                       | NodeSelector constraint on nodes eligible to run the provisioner.                                     | map      | `-`                                                        |
| daemonset.tolerations                        | List of tolerations to be applied to the Provisioner DaemonSet.                                       | list     | `-`                                                        |
| daemonset.resources                          | Map of resource request and limits to be applied to the Provisioner Daemonset.                        | map      | `-`                                                        |
| prometheus.operator.enabled                  | If set to true, will configure Prometheus monitoring                                                  | bool     | `false`                                                    |
| prometheus.operator.serviceMonitor.interval  | Interval at which Prometheus scrapes the provisioner                                                  | str      | `10s`                                                      |
| prometheus.operator.serviceMonitor.namespace | The namespace Prometheus is installed in                                                              | str      | `monitoring`                                               |
| prometheus.operator.serviceMonitor.selector  | The Prometheus selector label                                                                         | map      | `prometheus: kube-prometheus`                              |

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
* [more...](examples/)
