## Getting started

These instructions reflect the latest version of the codebase.  For instructions
on older versions, please see version links under
[Version Compatibility](../README.md#version-compatibility).

## Table of Contents

- [Step 1: Bringing up a cluster with local disks](#step-1-bringing-up-a-cluster-with-local-disks)
  * [Enabling the alpha feature gates](#enabling-the-alpha-feature-gates)
    + [1.10-1.12](#110-112)
  * [Option 1: GCE](#option-1-gce)
  * [Option 2: GKE](#option-2-gke)
  * [Option 3: Baremetal environments](#option-3-baremetal-environments)
  * [Option 4: Local test cluster](#option-4-local-test-cluster)
  * [Option 5: EKS (experimental)](#option-5-eks-experimental)
  * [Option 6: AKS](#option-6-aks)
  * [Option 7: LKE](#option-7-lke)
- [Step 2: Creating a StorageClass (1.9+)](#step-2-creating-a-storageclass-19)
- [Step 3: Creating local persistent volumes](#step-3-creating-local-persistent-volumes)
  * [Option 1: Using the local volume static provisioner](#option-1-using-the-local-volume-static-provisioner)
  * [Option 2: Manually create local persistent volume](#option-2-manually-create-local-persistent-volume)
- [Step 4: Create local persistent volume claim](#step-4-create-local-persistent-volume-claim)

### Step 1: Bringing up a cluster with local disks

#### Enabling the alpha feature gates

##### 1.10-1.12

If raw local block feature is needed,
```
$ export KUBE_FEATURE_GATES="BlockVolume=true"
```

Note: Kubernetes versions prior to 1.10 require [several additional
feature-gates](https://github.com/kubernetes-incubator/external-storage/tree/local-volume-provisioner-v2.0.0/local-volume#enabling-the-alpha-feature-gates)
be enabled on all Kubernetes components, because the persistent local volumes and other features were in alpha.

#### Option 1: GCE

GCE clusters brought up with `clusters/kube-up.sh` script in [Kubernetes
repository](https://github.com/kubernetes/kubernetes) will automatically format
and mount the requested Local SSDs, so you can deploy the provisioner with the
pre-generated deployment spec and skip to [step
4](#step-4-create-local-persistent-volume-claim), unless you want to customize
the provisioner spec or storage classes.

``` console
$ git clone --depth=1 https://github.com/kubernetes/kubernetes
$ cd kubernetes
$ NODE_LOCAL_SSDS_EXT=<n>,<scsi|nvme>,fs cluster/kube-up.sh
$ cd ../
$ kubectl create -f helm/generated_examples/gce.yaml
```

#### Option 2: GKE

GKE clusters will automatically format and mount the
requested Local SSDs. Please see
[GKE
documentation](https://cloud.google.com/kubernetes-engine/docs/how-to/persistent-volumes/local-ssd)
for instructions for how to create a cluster with Local SSDs.

Then skip to [step 4](#step-4-create-local-persistent-volume-claim).

**Note:** The raw block feature is only supported on GKE Kubernetes alpha clusters.

#### Option 3: Baremetal environments

1. Partition and format the disks on each node according to your application's
   requirements.
2. Mount all the filesystems under one directory per StorageClass. The directories
   are specified in a configmap, see below.
3. Configure the Kubernetes API Server, controller-manager, scheduler, and all kubelets
   with `KUBE_FEATURE_GATES` as described [above](#enabling-the-alpha-feature-gates).
4. If not using the default Kubernetes scheduler policy, the following
   predicates must be enabled:
   * Pre-1.9: `NoVolumeBindConflict`
   * 1.9+: `VolumeBindingChecker`

#### Option 4: Local test cluster

Kubernetes provides a script to build and start a lightweight local cluster on
Linux. You can try to deploy local-volume-provisioner in this cluster and
discovery local volumes on your local machine.

1. Create `/mnt/disks` directory and mount several volumes into its subdirectories.
   The example below uses three ram disks to simulate real local volumes:

```console
$ mkdir /mnt/disks
$ for vol in vol1 vol2 vol3; do
    mkdir /mnt/disks/$vol
    mount -t tmpfs $vol /mnt/disks/$vol
done
```

2. Run the local cluster.

```console
$ git clone --depth=1 https://github.com/kubernetes/kubernetes
$ cd kubernetes
$ ALLOW_PRIVILEGED=true LOG_LEVEL=5 FEATURE_GATES=$KUBE_FEATURE_GATES hack/local-up-cluster.sh
```

See [running Kubernetes
locally](https://github.com/kubernetes/community/blob/master/contributors/devel/running-locally.md)
for more information.

#### Option 5: EKS (experimental)

##### eks-nvme-ssd-provisioner
[eks-nvme-ssd-provisioner](https://github.com/brunsgaard/eks-nvme-ssd-provisioner)
runs as a DaemonSet and will automatically format and mount the requested local
NVMe SSDs.

**Note:** This project mounts disks in `/pv-disks/$uuid`. There is a
working example of storage local static provisioner resources in the
eks-nvme-ssd-provisioner repo.

##### Using raw block devices directly

You can  also mount the nvme instance storage disks directly.  You can do this
by symlinking the Instance Storage disks for discovery using udev automatically.
This has the benefit of not needing an additional component like
`eks-nvme-ssd-provisioner` to be deployed.

The following udev rule will symlink all Instance Storage disks under `/dev/disk/kubernetes/<uniqe id>`:
```
# /etc/udev/rules.d/90-kubernetes-discovery.rules

# Discover Instance Storage disks so kubernetes local provisioner can pick them up from /dev/disk/kubernetes
KERNEL=="nvme[0-9]*n[0-9]*", ENV{DEVTYPE}=="disk", ATTRS{model}=="Amazon EC2 NVMe Instance Storage", ATTRS{serial}=="?*", SYMLINK+="disk/kubernetes/nvme-$attr{model}_$attr{serial}", OPTIONS="string_escape=replace"

```

e.g. you could bring up an eks cluster using [eksctl](https://eksctl.io) that sets up these udev rules on startup as follows:

```yaml
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig
metadata:
  name: cluster-with-storage
  region: eu-central-1
managedNodeGroups:
  - name: storage-nvme
    desiredCapacity: 4
    instanceType: i3.large
    preBootstrapCommands:
      - |
          cat <<EOF > /etc/udev/rules.d/90-kubernetes-discovery.rules
          # Discover Instance Storage disks so kubernetes local provisioner can pick them up from /dev/disk/kubernetes
          KERNEL=="nvme[0-9]*n[0-9]*", ENV{DEVTYPE}=="disk", ATTRS{model}=="Amazon EC2 NVMe Instance Storage", ATTRS{serial}=="?*", SYMLINK+="disk/kubernetes/nvme-\\\$attr{model}_\\\$attr{serial}", OPTIONS="string_escape=replace"
          EOF
      - udevadm control --reload && udevadm trigger
```


You can then use

```
$ kubectl create -f helm/generated_examples/eks-nvme-ssd.yaml
```

or use helm with `helm/examples/eks-nvme-ssd.yaml`

to setup provisioning.

#### Option 6: AKS
See [Local Persistent Volume support on Azure](https://github.com/Azure/kubernetes-volume-drivers/tree/master/local) for more information.

### Option 7: LKE

LKE clusters can be created with custom Node Pools using the [Linode API](https://www.linode.com/docs/products/tools/linode-api/). For more information, see the [LKE Endpoints Collection](https://www.linode.com/docs/api/linode-kubernetes-engine-lke).

### Step 2: Creating a StorageClass (1.9+)

To delay volume binding until pod scheduling and to handle multiple local PVs in
a single pod, a StorageClass must to be created with `volumeBindingMode` set to
`WaitForFirstConsumer`.

```console
$ kubectl create -f deployment/kubernetes/example/default_example_storageclass.yaml
```

### Step 3: Creating local persistent volumes

#### Option 1: Using the local volume static provisioner

1. Generate Provisioner's ServiceAccount, Roles, DaemonSet, and ConfigMap spec, and customize it.

    This step uses helm templates to generate the specs.  See the [helm README](/helm/README.md) for setup instructions.
    To generate the provisioner's specs using the [default values](../helm/provisioner/values.yaml), run:

    ``` console
    helm template ./helm/provisioner > deployment/kubernetes/provisioner_generated.yaml
    ```

    You can also provide a custom values file instead:

    ``` console
    helm template ./helm/provisioner --values custom-values.yaml > deployment/kubernetes/provisioner_generated.yaml
    ```

2. Deploy Provisioner

    Once a user is satisfied with the content of Provisioner's yaml file, **kubectl** can be used
    to create Provisioner's DaemonSet and ConfigMap.

    ``` console
    $ kubectl create -f deployment/kubernetes/provisioner_generated.yaml
    ```

3. Check discovered local volumes

    Once launched, the external static provisioner will discover and create local-volume PVs.

    For example, if the directory `/mnt/disks/` contained one directory `/mnt/disks/vol1` then the following
    local-volume PV would be created by the static provisioner:

    ```
    $ kubectl get pv
    NAME                CAPACITY    ACCESSMODES   RECLAIMPOLICY   STATUS      CLAIM     STORAGECLASS    REASON    AGE
    local-pv-ce05be60   1024220Ki   RWO           Delete          Available             local-storage             26s

    $ kubectl describe pv local-pv-ce05be60
    Name:		local-pv-ce05be60
    Labels:		<none>
    Annotations:	pv.kubernetes.io/provisioned-by=local-volume-provisioner-minikube-18f57fb2-a186-11e7-b543-080027d51893
    StorageClass:	local-storage
    Status:		Available
    Claim:
    Reclaim Policy:	Delete
    Access Modes:	RWO
    Capacity:	1024220Ki
    NodeAffinity:
      Required Terms:
          Term 0:  kubernetes.io/hostname in [my-node]
    Message:
    Source:
        Type:	LocalVolume (a persistent volume backed by local storage on a node)
        Path:	/mnt/disks/vol1
    Events:		<none>
    ```

    The PV described above can be claimed and bound to a PVC by referencing the `local-storage` storageClassName.

#### Option 2: Manually create local persistent volume

See [Kubernetes documentation](https://kubernetes.io/docs/concepts/storage/volumes/#local)
for an example PersistentVolume spec.

### Step 4: Create local persistent volume claim

``` yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: example-local-claim
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
  storageClassName: local-storage
```
Please replace the following elements to reflect your configuration:

  * "5Gi" with required size of storage volume
  * "local-storage" with the name of storage class associated with the
  local PVs that should be used for satisfying this PVC

For "Block" volumeMode PVC, which tries to claim a "Block" PV, the following
example can be used:

``` yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: example-local-claim
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
  volumeMode: Block
  storageClassName: local-storage
```
Note that the only additional field of interest here is volumeMode, which has been set
to "Block".

