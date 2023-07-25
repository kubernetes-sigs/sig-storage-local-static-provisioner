# Release Process

local-volume-provisioner is released on an as-needed basis. The process is as follows:

## Image release

1. Create a PR with the CHANGELOG contents, to generate a CHANGELOG follow the steps
   in https://github.com/kubernetes-csi/csi-release-tools/blob/master/SIDECAR_RELEASE_PROCESS.md#release-process
   1. Compare the generated output to the new commits for the release to check if any notable change missed a release note.
   1. Reword release notes as needed. Make sure to check notes for breaking changes and deprecations.
   1. If release is a new major/minor version, create a new CHANGELOG-<major>.<minor>.md file. Otherwise, add the release notes to the top of the existing CHANGELOG file for that minor version.
   1. Submit a PR for the CHANGELOG changes and wait for it to be merged.
   1. Make sure that no new PRs have merged in the meantime, and no PRs are in flight and soon to be merged.
1. Check CI job at https://testgrid.k8s.io/sig-storage-local-static-provisioner#master-gce-lastest
1. An OWNER runs `git tag -a $VERSION` and pushes the tag with `git push $VERSION`.
1. Create a new release following a previous release as a template. Be sure to select the correct branch. This requires Github release permissions as required by the prerequisites.
1. If release was a new major/minor version, create a new release-<minor> branch at that commit.
1. On git tag push, a [post-submit Prow job](https://testgrid.k8s.io/sig-storage-image-build#post-sig-storage-local-static-provisioner-push-images) will push the local volume provisioner image to k8s-staging-sig-storage.
1. The image needs to be promoted to registry.k8s.io by running
   [generate.sh](https://github.com/kubernetes/k8s.io/tree/main/registry.k8s.io/images/k8s-staging-sig-storage)
   and creating a PR.

## Helm chart release

1. An OWNER submits a PR to bump the helm chart version to a stable version
   1. In `helm/provisioner/Chart.yaml` bump the `version` and `appVersion` to the next major/minor/patch version
   1. In `helm/provisioner/values.yaml` bump the image version to the next major/minor/patch version
   1. Run `./hack/update-generated.sh`
   1. Submit a PR
1. TODO create a tag for the helm chart release.
1. On git tag push.
   1. The helm-chart-release Github Action will create a Github Release with the contents of the chart, it'll also recreate
      the contents of gh-pages with a manifest that the helm cli can use to download a specific version of a release.

