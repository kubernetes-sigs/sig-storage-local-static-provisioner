# FAQs

## Table of Contents

- [I updated provisioner configuration but volumes are not discovered](#i-updated-provisioner-configuration-but-volumes-are-not-discovered)
- [I bind mounted a directory into sub-directory of discovery directory, but no PVs created](#i-bind-mounted-a-directory-into-sub-directory-of-discovery-directory-but-no-pvs-created)
- [Failed to start when docker --init flag is enabled.](#failed-to-start-when-docker---init-flag-is-enabled)

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
