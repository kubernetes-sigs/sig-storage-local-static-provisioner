# FAQs

## Table of Contents

- [I updated provisioner configuration but volumes are not discovered](#i-updated-provisioner-configuration-but-volumes-are-not-discovered)
- [I bind mounted a directory into sub-directory of discovery directory, but no PVs created](#i-bind-mounted-a-directory-into-sub-directory-of-discovery-directory-but-no-pvs-created)
- [Failed to start when docker --init flag is enabled.](#failed-to-start-when-docker---init-flag-is-enabled)
- [Can I clean volume data by deleting PV object?](#can-i-clean-volume-data-by-deleting-pv-object)
- [PV with delete reclaimPolicy is released but not going to be reclaimed](#pv-with-delete-reclaimpolicy-is-released-but-not-going-to-be-reclaimed)
- [Why my application uses an empty volume when node gets recreated in GCP](#why-my-application-uses-an-empty-volume-when-node-gets-recreated-in-gcp)
- [Can I change storage class name after some volumes has been provisioned](#can-i-change-storage-class-name-after-some-volumes-has-been-provisioned)

## I updated provisioner configuration but volumes are not discovered

Currently, provisioner will not reload configuration automatically. You need to restart them.


Check your local volume provisioners:

```
kubectl -n <provisioner-namespace> get pods -l app=local-volume-provisioner
```

Delete them:

```
kubectl -n <provisioner-namespace> delete pods -l app=local-volume-provisioner
```

Check new pods are created by daemon set, also please make sure they are running in the last.

```
kubectl -n <provisioner-namespace> get pods -l app=local-volume-provisioner
```
 
## I bind mounted a directory into sub-directory of discovery directory, but no PVs created

Provisioner only creates local PVs for mounted directories in the first level
of discovery directory. This is because there is no way to detect whether the
sub-directory is created by system admin for discovering as local PVs or by
user applications.

## Failed to start when docker --init flag is enabled.

We need to mount `/dev` into container for device discovery, it conflicts when
`docker --init` flag is enabled because docker will mount [`init` process](https://docs.docker.com/engine/reference/run/#specify-an-init-process) at `/dev/init`.

This has been fixed in
[moby/moby#37665](https://github.com/moby/moby/pull/37665).

Workarounds before the fix is released:

- do not use docker `--init`, packs [tini](https://github.com/krallin/tini) in your docker image
- do not use docker `--init`, [share process namespace](https://kubernetes.io/docs/tasks/configure-pod-container/share-process-namespace/) in your pod and use [pause:3.1+](https://github.com/kubernetes/kubernetes/blob/master/build/pause/CHANGELOG.md) to clean up orphaned zombie processes
- do not mount `/dev`, provisioner will discover current mounted devices, but it cannot discovery the newly mounted (see [why](https://github.com/kubernetes-incubator/external-storage/issues/783#issuecomment-395013458))

## Can I clean volume data by deleting PV object?

No, there is no reliable mechanism in provisioner to detect the PV object of
volume is deleted by the user or not created yet. This is because delete event
will not be delivered when provisioner is not running and provisioner don't
know whether the volume has been discovered before.

So provisioner will always discover volume as a new PV if no existing PV is
associated with it. The volume data will not be cleaned in this phrase, and old
data in it may leak other applications.

So you must not delete PV objects by yourself, always delete PVC objects and
set `PersistentVolumeReclaimPolicy` to `Delete` to clean volume data of
associated PVs.

## PV with delete reclaimPolicy is released but not going to be reclaimed

At first, please check provisioner is running on the node.  If provisioner is
running, please check whether the volume exists on the node. If the volume is
missing, provisioner can not clean the volume data. For safety, it will not
clean the associated PV object. This is to prevent old volume from being used
by other programs if it recovers later.

It’s up to the system administrator to fix this:

- If the volume has been decommissioned, you can delete the PV object manually
  (e.g. `kubectl delete <pv-name>`)
- If the volume is missing because of the invalid `/etc/fstab`, setup script or
  hardware failures, please fix the volume. If the volume can be recovered,
  provisioner will continue to clean the volume data and reclaim the PV object.
  If the volume cannot be recovered, you can remove the volume (or disk) from
  the node and delete the PV object manually.

Of course, on a specific platform if you have a reliable mechanism to detect if
a volume is permanently deleted or cannot recover, you can write an operator or
sidecar to automate this.

## Why my application uses an empty volume when node gets recreated in GCP

Please check `spec.local.path` field of local PV object. If it is a non-unique
path (e.g. without UUID in it) e.g. `/mnt/disks/ssd0`, newly created disks may be
mounted at the same path.

For example, in GKE when nodes get recreated, local SSDs will be recreated and
mounted at the same paths (note: `--keep-disks` cannot be used to keep local
SSDs because `autoDelete` cannot be set to false on local SSDs). Pods using old
volumes will start with empty volumes because paths of PV objects will get
mounted with newly created disks.

If your application does not expect this behavior, you should use
[`--local-ssd-volumes`](https://cloud.google.com/sdk/gcloud/reference/alpha/container/node-pools/create)
and configure provisioner to discover volumes under `/mnt/disks/by-uuid/google-local-ssds-scsi-fs` or
`/mnt/disks/by-uuid/google-local-ssds-nvme-fs`. Here is [an
example](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/blob/master/helm/generated_examples/gce.yaml).

This applies in other environments if local paths you configured are not
stable. See our [operations guide](operations.md) and [best
practices](best-practices.md) in production.

## Can I change storage class name after some volumes has been provisioned

Basically, you can't. When a discovery directory is configured in a storage
class, it cannot be configured in another storage class, otherwise, volumes
will be discovered again under different storage class. Pods which request PVs
from different storage classes can mount the same volume. Once a directory is
configured in a storage class, it's better to not change.

For now, we don't support migrating volumes to another storage class. If you
really need to do this, the only way is to clean all volumes under old
storage class, configure discovery directory under new storage class and
restart all provisioners.
