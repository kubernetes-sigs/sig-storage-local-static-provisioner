#! /bin/bash

# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


# This script runs inside a Prow job. It can run unit tests ("make test")
# and E2E testing. This E2E testing covers different scenarios (see
# https://github.com/kubernetes/enhancements/pull/807):
# - running the stable hostpath example against a Kubernetes release
# - running the canary hostpath example against a Kubernetes release
# - building the component in the current repo and running the
#   stable hostpath example with that one component replaced against
#   a Kubernetes release
#
# The intended usage of this script is that individual repos import
# csi-release-tools, then link their top-level prow.sh to this or
# include it in that file. When including it, several of the variables
# can be overridden in the top-level prow.sh to customize the script
# for the repo.
#
# The expected environment is:
# - $GOPATH/src/<import path> for the repository that is to be tested,
#   with PR branch merged (when testing a PR)
# - running on linux-amd64
# - kind (https://github.com/kubernetes-sigs/kind) installed
# - optional: Go already installed

RELEASE_TOOLS_ROOT="$(realpath "$(dirname "${BASH_SOURCE[0]}")")"
REPO_DIR="$(pwd)"

# Sets the default value for a variable if not set already and logs the value.
# Any variable set this way is usually something that a repo's .prow.sh
# or the job can set.
configvar () {
    # Ignore: Word is of the form "A"B"C" (B indicated). Did you mean "ABC" or "A\"B\"C"?
    # shellcheck disable=SC2140
    eval : \$\{"$1":="\$2"\}
    eval echo "\$3:" "$1=\${$1}"
}

# Prints the value of a variable + version suffix, falling back to variable + "LATEST".
get_versioned_variable () {
    local var="$1"
    local version="$2"
    local value

    eval value="\${${var}_${version}}"
    if ! [ "$value" ]; then
        eval value="\${${var}_LATEST}"
    fi
    echo "$value"
}

# This takes a version string like CSI_PROW_KUBERNETES_VERSION and
# maps it to the corresponding git tag, branch or commit.
version_to_git () {
    version="$1"
    shift
    case "$version" in
        latest|master) echo "master";;
        release-*) echo "$version";;
        *) echo "v$version";;
    esac
}

# the list of windows versions was matched from:
# - https://hub.docker.com/_/microsoft-windows-nanoserver
# - https://hub.docker.com/_/microsoft-windows-servercore
configvar CSI_PROW_BUILD_PLATFORMS "linux amd64 amd64; linux ppc64le ppc64le -ppc64le; linux s390x s390x -s390x; linux arm arm -arm; linux arm64 arm64 -arm64; linux arm arm/v7 -armv7; windows amd64 amd64 .exe nanoserver:1809 servercore:ltsc2019; windows amd64 amd64 .exe nanoserver:20H2 servercore:20H2; windows amd64 amd64 .exe nanoserver:ltsc2022 servercore:ltsc2022" "Go target platforms (= GOOS + GOARCH) and file suffix of the resulting binaries"

# If we have a vendor directory, then use it. We must be careful to only
# use this for "make" invocations inside the project's repo itself because
# setting it globally can break other go usages (like "go get <some command>"
# which is disabled with GOFLAGS=-mod=vendor).
configvar GOFLAGS_VENDOR "$( [ -d vendor ] && echo '-mod=vendor' )" "Go flags for using the vendor directory"

configvar CSI_PROW_GO_VERSION_BUILD "1.18" "Go version for building the component" # depends on component's source code
configvar CSI_PROW_GO_VERSION_E2E "" "override Go version for building the Kubernetes E2E test suite" # normally doesn't need to be set, see install_e2e
configvar CSI_PROW_GO_VERSION_SANITY "${CSI_PROW_GO_VERSION_BUILD}" "Go version for building the csi-sanity test suite" # depends on CSI_PROW_SANITY settings below
configvar CSI_PROW_GO_VERSION_KIND "${CSI_PROW_GO_VERSION_BUILD}" "Go version for building 'kind'" # depends on CSI_PROW_KIND_VERSION below
configvar CSI_PROW_GO_VERSION_GINKGO "${CSI_PROW_GO_VERSION_BUILD}" "Go version for building ginkgo" # depends on CSI_PROW_GINKGO_VERSION below

# ginkgo test runner version to use. If the pre-installed version is
# different, the desired version is built from source.
configvar CSI_PROW_GINKGO_VERSION v1.7.0 "Ginkgo"

# Ginkgo runs the E2E test in parallel. The default is based on the number
# of CPUs, but typically this can be set to something higher in the job.
configvar CSI_PROW_GINKO_PARALLEL "-p" "Ginko parallelism parameter(s)"

# Enables building the code in the repository. On by default, can be
# disabled in jobs which only use pre-built components.
configvar CSI_PROW_BUILD_JOB true "building code in repo enabled"

# Kubernetes version to test against. This must be a version number
# (like 1.13.3), "latest" (builds Kubernetes from the master branch)
# or "release-x.yy" (builds Kubernetes from a release branch).
#
# The patch version is only relevant for picking the E2E test suite
# that is used for testing. The script automatically picks
# the kind images for the major/minor version of Kubernetes
# that the kind release supports.
#
# This can also be a version that was not released yet at the time
# that the settings below were chose. The script will then
# use the same settings as for "latest" Kubernetes. This works
# as long as there are no breaking changes in Kubernetes, like
# deprecating or changing the implementation of an alpha feature.
configvar CSI_PROW_KUBERNETES_VERSION 1.17.0 "Kubernetes"

# CSI_PROW_KUBERNETES_VERSION reduced to first two version numbers and
# with underscore (1_13 instead of 1.13.3) and in uppercase (LATEST
# instead of latest).
#
# This is used to derive the right defaults for the variables below
# when a Prow job just defines the Kubernetes version.
csi_prow_kubernetes_version_suffix="$(echo "${CSI_PROW_KUBERNETES_VERSION}" | tr . _ | tr '[:lower:]' '[:upper:]' | sed -e 's/^RELEASE-//' -e 's/\([0-9]*\)_\([0-9]*\).*/\1_\2/')"

# Only the latest KinD is (eventually) guaranteed to work with the
# latest Kubernetes. For example, KinD 0.10.0 failed with Kubernetes
# 1.21.0-beta1.  Therefore the default version of KinD is "main"
# for that, otherwise the latest stable release for which we then
# list the officially supported images below.
kind_version_default () {
    case "${CSI_PROW_KUBERNETES_VERSION}" in
        latest|master)
            echo main;;
        *)
            echo v0.11.1;;
    esac
}

# kind version to use. If the pre-installed version is different,
# the desired version is downloaded from https://github.com/kubernetes-sigs/kind/releases
# (if available), otherwise it is built from source.
configvar CSI_PROW_KIND_VERSION "$(kind_version_default)" "kind"

# kind images to use. Must match the kind version.
# The release notes of each kind release list the supported images.
configvar CSI_PROW_KIND_IMAGES "kindest/node:v1.23.0@sha256:49824ab1727c04e56a21a5d8372a402fcd32ea51ac96a2706a12af38934f81ac
kindest/node:v1.22.0@sha256:b8bda84bb3a190e6e028b1760d277454a72267a5454b57db34437c34a588d047
kindest/node:v1.21.1@sha256:69860bda5563ac81e3c0057d654b5253219618a22ec3a346306239bba8cfa1a6
kindest/node:v1.20.7@sha256:cbeaf907fc78ac97ce7b625e4bf0de16e3ea725daf6b04f930bd14c67c671ff9
kindest/node:v1.19.11@sha256:07db187ae84b4b7de440a73886f008cf903fcf5764ba8106a9fd5243d6f32729
kindest/node:v1.18.19@sha256:7af1492e19b3192a79f606e43c35fb741e520d195f96399284515f077b3b622c
kindest/node:v1.17.17@sha256:66f1d0d91a88b8a001811e2f1054af60eef3b669a9a74f9b6db871f2f1eeed00
kindest/node:v1.16.15@sha256:83067ed51bf2a3395b24687094e283a7c7c865ccc12a8b1d7aa673ba0c5e8861
kindest/node:v1.15.12@sha256:b920920e1eda689d9936dfcf7332701e80be12566999152626b2c9d730397a95
kindest/node:v1.14.10@sha256:f8a66ef82822ab4f7569e91a5bccaf27bceee135c1457c512e54de8c6f7219f8" "kind images"

# By default, this script tests sidecars with the CSI hostpath driver,
# using the install_csi_driver function. That function depends on
# a deployment script that it searches for in several places:
#
# - The "deploy" directory in the current repository: this is useful
#   for the situation that a component becomes incompatible with the
#   shared deployment, because then it can (temporarily!) provide its
#   own example until the shared one can be updated; it's also how
#   csi-driver-host-path itself provides the example.
#
# - CSI_PROW_DRIVER_VERSION of the CSI_PROW_DRIVER_REPO is checked
#   out: this allows other repos to reference a version of the example
#   that is known to be compatible.
#
# - The <driver repo>/deploy directory can have multiple sub-directories,
#   each with different deployments (stable set of images for Kubernetes 1.13,
#   stable set of images for Kubernetes 1.14, canary for latest Kubernetes, etc.).
#   This is necessary because there may be incompatible changes in the
#   "API" of a component (for example, its command line options or RBAC rules)
#   or in its support for different Kubernetes versions (CSIDriverInfo as
#   CRD in Kubernetes 1.13 vs builtin API in Kubernetes 1.14).
#
#   When testing an update for a component in a PR job, the
#   CSI_PROW_DEPLOYMENT variable can be set in the
#   .prow.sh of each component when there are breaking changes
#   that require using a non-default deployment. The default
#   is a deployment named "kubernetes-x.yy${CSI_PROW_DEPLOYMENT_SUFFIX}" (if available),
#   otherwise "kubernetes-latest${CSI_PROW_DEPLOYMENT_SUFFIX}".
#   "none" disables the deployment of the hostpath driver.
#
# When no deploy script is found (nothing in `deploy` directory,
# CSI_PROW_DRIVER_REPO=none), nothing gets deployed.
#
# If the deployment script is called with CSI_PROW_TEST_DRIVER=<file name> as
# environment variable, then it must write a suitable test driver configuration
# into that file in addition to installing the driver.
configvar CSI_PROW_DRIVER_VERSION "v1.3.0" "CSI driver version"
configvar CSI_PROW_DRIVER_REPO https://github.com/kubernetes-csi/csi-driver-host-path "CSI driver repo"
configvar CSI_PROW_DEPLOYMENT "" "deployment"
configvar CSI_PROW_DEPLOYMENT_SUFFIX "" "additional suffix in kubernetes-x.yy[suffix].yaml files"

# The install_csi_driver function may work also for other CSI drivers,
# as long as they follow the conventions of the CSI hostpath driver.
# If they don't, then a different install function can be provided in
# a .prow.sh file and this config variable can be overridden.
configvar CSI_PROW_DRIVER_INSTALL "install_csi_driver" "name of the shell function which installs the CSI driver"

# If CSI_PROW_DRIVER_CANARY is set (typically to "canary", but also
# version tag. Usually empty. CSI_PROW_HOSTPATH_CANARY is
# accepted as alternative name because some test-infra jobs
# still use that name.
configvar CSI_PROW_DRIVER_CANARY "${CSI_PROW_HOSTPATH_CANARY}" "driver image override for canary images"

# Image registry to use for canary images.
# Only valid if CSI_PROW_DRIVER_CANARY == "canary".
configvar CSI_PROW_DRIVER_CANARY_REGISTRY "gcr.io/k8s-staging-sig-storage" "registry for canary images"

# The E2E testing can come from an arbitrary repo. The expectation is that
# the repo supports "go test ./test/e2e -args --storage.testdriver" (https://github.com/kubernetes/kubernetes/pull/72836)
# after setting KUBECONFIG. As a special case, if the repository is Kubernetes,
# then `make WHAT=test/e2e/e2e.test` is called first to ensure that
# all generated files are present.
#
# CSI_PROW_E2E_REPO=none disables E2E testing.
configvar CSI_PROW_E2E_VERSION "$(version_to_git "${CSI_PROW_KUBERNETES_VERSION}")"  "E2E version"
configvar CSI_PROW_E2E_REPO "https://github.com/kubernetes/kubernetes" "E2E repo"
configvar CSI_PROW_E2E_IMPORT_PATH "k8s.io/kubernetes" "E2E package"

# csi-sanity testing from the csi-test repo can be run against the installed
# CSI driver. For this to work, deploying the driver must expose the Unix domain
# csi.sock as a TCP service for use by the csi-sanity command, which runs outside
# of the cluster. The alternative would have been to (cross-)compile csi-sanity
# and install it inside the cluster, which is not necessarily easier.
configvar CSI_PROW_SANITY_REPO https://github.com/kubernetes-csi/csi-test "csi-test repo"
configvar CSI_PROW_SANITY_VERSION v4.3.0 "csi-test version"
configvar CSI_PROW_SANITY_PACKAGE_PATH github.com/kubernetes-csi/csi-test "csi-test package"
configvar CSI_PROW_SANITY_SERVICE "hostpath-service" "Kubernetes TCP service name that exposes csi.sock"
configvar CSI_PROW_SANITY_POD "csi-hostpathplugin-0" "Kubernetes pod with CSI driver"
configvar CSI_PROW_SANITY_CONTAINER "hostpath" "Kubernetes container with CSI driver"

# The version of dep to use for 'make test-vendor'. Ignored if the project doesn't
# use dep. Only binary releases of dep are supported (https://github.com/golang/dep/releases).
configvar CSI_PROW_DEP_VERSION v0.5.1 "golang dep version to be used for vendor checking"

# Each job can run one or more of the following tests, identified by
# a single word:
# - unit testing
# - parallel excluding alpha features
# - serial excluding alpha features
# - parallel, only alpha feature
# - serial, only alpha features
# - sanity
#
# Unknown or unsupported entries are ignored.
#
# Testing of alpha features is only supported for CSI_PROW_KUBERNETES_VERSION=latest
# because CSI_PROW_E2E_ALPHA and CSI_PROW_E2E_ALPHA_GATES are not set for
# older Kubernetes releases. The script supports that, it just isn't done because
# it is not needed and would cause additional maintenance effort.
#
# Sanity testing with csi-sanity only covers the CSI driver itself and
# thus only makes sense in repos which provide their own CSI
# driver. Repos can enable sanity testing by setting
# CSI_PROW_TESTS_SANITY=sanity.
configvar CSI_PROW_TESTS "unit parallel serial $(if [ "${CSI_PROW_KUBERNETES_VERSION}" = "latest" ]; then echo parallel-alpha serial-alpha; fi) sanity" "tests to run"
tests_enabled () {
    local t1 t2
    # We want word-splitting here, so ignore: Quote to prevent word splitting, or split robustly with mapfile or read -a.
    # shellcheck disable=SC2206
    local tests=(${CSI_PROW_TESTS})
    for t1 in "$@"; do
        for t2 in "${tests[@]}"; do
            if [ "$t1" = "$t2" ]; then
                return
            fi
        done
    done
    return 1
}
sanity_enabled () {
    [ "${CSI_PROW_TESTS_SANITY}" = "sanity" ] && tests_enabled "sanity"
}
tests_need_kind () {
    tests_enabled "parallel" "serial" "serial-alpha" "parallel-alpha" ||
        sanity_enabled
}
tests_need_non_alpha_cluster () {
    tests_enabled "parallel" "serial" ||
        sanity_enabled
}
tests_need_alpha_cluster () {
    tests_enabled "parallel-alpha" "serial-alpha"
}

# Enabling mock tests adds the "CSI mock volume" tests from https://github.com/kubernetes/kubernetes/blob/HEAD/test/e2e/storage/csi_mock_volume.go
# to the e2e.test invocations (serial, parallel, and the corresponding alpha variants).
# When testing canary images, those get used instead of the images specified
# in the e2e.test's normal YAML files.
#
# The default is to enable this for all jobs which use canary images
# and the latest Kubernetes because those images will be used for mock
# testing once they are released. Using them for mock testing with
# older Kubernetes releases is too risky because the deployment files
# can be very old (for example, still using a removed -provisioner
# parameter in external-provisioner).
configvar CSI_PROW_E2E_MOCK "$(if [ "${CSI_PROW_DRIVER_CANARY}" = "canary" ] && [ "${CSI_PROW_KUBERNETES_VERSION}" = "latest" ]; then echo true; else echo false; fi)" "enable CSI mock volume tests"

# Regex for non-alpha, feature-tagged tests that should be run.
#
configvar CSI_PROW_E2E_FOCUS_LATEST '\[Feature:VolumeSnapshotDataSource\]' "non-alpha, feature-tagged tests for latest Kubernetes version"
configvar CSI_PROW_E2E_FOCUS "$(get_versioned_variable CSI_PROW_E2E_FOCUS "${csi_prow_kubernetes_version_suffix}")" "non-alpha, feature-tagged tests"

# Serial vs. parallel is always determined by these regular expressions.
# Individual regular expressions are separated by spaces for readability
# and expected to not contain spaces. Use dots instead. The complete
# regex for Ginkgo will be created by joining the individual terms.
configvar CSI_PROW_E2E_SERIAL '\[Serial\] \[Disruptive\]' "tags for serial E2E tests"
regex_join () {
    echo "$@" | sed -e 's/  */|/g' -e 's/^|*//' -e 's/|*$//' -e 's/^$/this-matches-nothing/g'
}

# Which tests are alpha depends on the Kubernetes version. We could
# use the same E2E test for all Kubernetes version. This would have
# the advantage that new tests can be applied to older versions
# without having to backport tests.
#
# But the feature tag gets removed from E2E tests when the corresponding
# feature becomes beta, so we would have to track which tests were
# alpha in previous Kubernetes releases. This was considered too
# error prone. Therefore we use E2E tests that match the Kubernetes
# version that is getting tested.
configvar CSI_PROW_E2E_ALPHA_LATEST '\[Feature:' "alpha tests for latest Kubernetes version" # there's no need to update this, adding a new case for CSI_PROW_E2E for a new Kubernetes is enough
configvar CSI_PROW_E2E_ALPHA "$(get_versioned_variable CSI_PROW_E2E_ALPHA "${csi_prow_kubernetes_version_suffix}")" "alpha tests"

# After the parallel E2E test without alpha features, a test cluster
# with alpha features is brought up and tests that were previously
# disabled are run. The alpha gates in each release have to be listed
# explicitly. If none are set (= variable empty), alpha testing
# is skipped.
#
# Testing against "latest" Kubernetes is problematic because some alpha
# feature which used to work might stop working or change their behavior
# such that the current tests no longer pass. If that happens,
# kubernetes-csi components must be updated, either by disabling
# the failing test for "latest" or by updating the test and not running
# it anymore for older releases.
configvar CSI_PROW_E2E_ALPHA_GATES_LATEST 'GenericEphemeralVolume=true,CSIStorageCapacity=true' "alpha feature gates for latest Kubernetes"
configvar CSI_PROW_E2E_ALPHA_GATES "$(get_versioned_variable CSI_PROW_E2E_ALPHA_GATES "${csi_prow_kubernetes_version_suffix}")" "alpha E2E feature gates"

# Which external-snapshotter tag to use for the snapshotter CRD and snapshot-controller deployment
default_csi_snapshotter_version () {
	if [ "${CSI_PROW_KUBERNETES_VERSION}" = "latest" ] || [ "${CSI_PROW_DRIVER_CANARY}" = "canary" ]; then
		echo "master"
	else
		echo "v3.0.2"
	fi
}
configvar CSI_SNAPSHOTTER_VERSION "$(default_csi_snapshotter_version)" "external-snapshotter version tag"

# Some tests are known to be unusable in a KinD cluster. For example,
# stopping kubelet with "ssh <node IP> systemctl stop kubelet" simply
# doesn't work. Such tests should be written in a way that they verify
# whether they can run with the current cluster provider, but until
# they are, we filter them out by name. Like the other test selection
# variables, this is again a space separated list of regular expressions.
configvar CSI_PROW_E2E_SKIP 'Disruptive' "tests that need to be skipped"

# This creates directories that are required for testing.
ensure_paths () {
    # Work directory. It has to allow running executables, therefore /tmp
    # is avoided. Cleaning up after the script is intentionally left to
    # the caller.
    configvar CSI_PROW_WORK "$(mkdir -p "$GOPATH/pkg" && mktemp -d "$GOPATH/pkg/csiprow.XXXXXXXXXX")" "work directory"

    # This is the directory for additional result files. Usually set by Prow, but
    # if not (for example, when invoking manually) it defaults to the work directory.
    configvar ARTIFACTS "${CSI_PROW_WORK}/artifacts" "artifacts"
    mkdir -p "${ARTIFACTS}"

    # For additional tools.
    CSI_PROW_BIN="${CSI_PROW_WORK}/bin"
    mkdir -p "${CSI_PROW_BIN}"
    PATH="${CSI_PROW_BIN}:$PATH"
}

run () {
    echo "$(date) $(go version | sed -e 's/.*version \(go[^ ]*\).*/\1/') $(if [ "$(pwd)" != "${REPO_DIR}" ]; then pwd; fi)\$" "$@" >&2
    "$@"
}

info () {
    echo >&2 INFO: "$@"
}

warn () {
    echo >&2 WARNING: "$@"
}

die () {
    echo >&2 ERROR: "$@"
    exit 1
}

# Ensure that PATH has the desired version of the Go tools, then run command given as argument.
# Empty parameter uses the already installed Go. In Prow, that version is kept up-to-date by
# bumping the container image regularly.
run_with_go () {
    local version
    version="$1"
    shift

    if ! [ "$version" ] || go version 2>/dev/null | grep -q "go$version"; then
        run "$@"
    else
        if ! [ -d "${CSI_PROW_WORK}/go-$version" ];  then
            run curl --fail --location "https://dl.google.com/go/go$version.linux-amd64.tar.gz" | tar -C "${CSI_PROW_WORK}" -zxf - || die "installation of Go $version failed"
            mv "${CSI_PROW_WORK}/go" "${CSI_PROW_WORK}/go-$version"
        fi
        PATH="${CSI_PROW_WORK}/go-$version/bin:$PATH" run "$@"
    fi
}

# Ensure that we have the desired version of kind.
install_kind () {
    if kind --version 2>/dev/null | grep -q " ${CSI_PROW_KIND_VERSION}$"; then
        return
    fi
    if run curl --fail --location -o "${CSI_PROW_WORK}/bin/kind" "https://github.com/kubernetes-sigs/kind/releases/download/${CSI_PROW_KIND_VERSION}/kind-linux-amd64"; then
        chmod u+x "${CSI_PROW_WORK}/bin/kind"
    else
        git_checkout https://github.com/kubernetes-sigs/kind "${GOPATH}/src/sigs.k8s.io/kind" "${CSI_PROW_KIND_VERSION}" --depth=1 &&
        (cd "${GOPATH}/src/sigs.k8s.io/kind" && run_with_go "$CSI_PROW_GO_VERSION_KIND" make install INSTALL_DIR="${CSI_PROW_WORK}/bin")
    fi
}

# Ensure that we have the desired version of the ginkgo test runner.
install_ginkgo () {
    # CSI_PROW_GINKGO_VERSION contains the tag with v prefix, the command line output does not.
    if [ "v$(ginkgo version 2>/dev/null | sed -e 's/.* //')" = "${CSI_PROW_GINKGO_VERSION}" ]; then
        return
    fi
    run_with_go "${CSI_PROW_GO_VERSION_GINKGO}" env GOBIN="${CSI_PROW_BIN}" go install "github.com/onsi/ginkgo/ginkgo@${CSI_PROW_GINKGO_VERSION}" || die "building ginkgo failed"
}

# Ensure that we have the desired version of dep.
install_dep () {
    if dep version 2>/dev/null | grep -q "version:.*${CSI_PROW_DEP_VERSION}$"; then
        return
    fi
    run curl --fail --location -o "${CSI_PROW_WORK}/bin/dep" "https://github.com/golang/dep/releases/download/v0.5.4/dep-linux-amd64" &&
        chmod u+x "${CSI_PROW_WORK}/bin/dep"
}

# This checks out a repo ("https://github.com/kubernetes/kubernetes")
# in a certain location ("$GOPATH/src/k8s.io/kubernetes") at
# a certain revision (a hex commit hash, v1.13.1, master). It's okay
# for that directory to exist already.
git_checkout () {
    local repo path revision
    repo="$1"
    shift
    path="$1"
    shift
    revision="$1"
    shift

    mkdir -p "$path"
    if ! [ -d "$path/.git" ]; then
        run git init "$path"
    fi
    if (cd "$path" && run git fetch "$@" "$repo" "$revision"); then
        (cd "$path" && run git checkout FETCH_HEAD) || die "checking out $repo $revision failed"
    else
        # Might have been because fetching by revision is not
        # supported by GitHub (https://github.com/isaacs/github/issues/436).
        # Fall back to fetching everything.
        (cd "$path" && run git fetch "$repo" '+refs/heads/*:refs/remotes/csiprow/heads/*' '+refs/tags/*:refs/tags/*') || die "fetching $repo failed"
        (cd "$path" && run git checkout "$revision") || die "checking out $repo $revision failed"
    fi
    # This is useful for local testing or when switching between different revisions in the same
    # repo.
    (cd "$path" && run git clean -fdx) || die "failed to clean $path"
}

# This clones a repo ("https://github.com/kubernetes/kubernetes")
# in a certain location ("$GOPATH/src/k8s.io/kubernetes") at
# a the head of a specific branch (i.e., release-1.13, master),
# tag (v1.20.0) or commit.
#
# The directory must not exist.
git_clone () {
    local repo path name parent
    repo="$1"
    shift
    path="$1"
    shift
    name="$1"
    shift

    parent="$(dirname "$path")"
    mkdir -p "$parent"
    (cd "$parent" && run git clone --single-branch --branch "$name" "$repo" "$path") || die "cloning $repo" failed
    # This is useful for local testing or when switching between different revisions in the same
    # repo.
    (cd "$path" && run git clean -fdx) || die "failed to clean $path"
}

list_gates () (
    set -f; IFS=','
    # Ignore: Double quote to prevent globbing and word splitting.
    # shellcheck disable=SC2086
    set -- $1
    while [ "$1" ]; do
        # Ignore: See if you can use ${variable//search/replace} instead.
        # shellcheck disable=SC2001
        echo "$1" | sed -e 's/ *\([^ =]*\) *= *\([^ ]*\) */      \1: \2/'
        shift
    done
)

# Turn feature gates in the format foo=true,bar=false into
# a YAML map with the corresponding API groups for use
# with https://kind.sigs.k8s.io/docs/user/configuration/#runtime-config
list_api_groups () (
    set -f; IFS=','
    # Ignore: Double quote to prevent globbing and word splitting.
    # shellcheck disable=SC2086
    set -- $1
    while [ "$1" ]; do
        if [ "$1" = 'CSIStorageCapacity=true' ]; then
            echo '   "storage.k8s.io/v1alpha1": "true"'
        fi
        shift
    done
)

go_version_for_kubernetes () (
    local path="$1"
    local version="$2"
    local go_version

    # We use the minimal Go version specified for each K8S release (= minimum_go_version in hack/lib/golang.sh).
    # More recent versions might also work, but we don't want to count on that.
    go_version="$(grep minimum_go_version= "$path/hack/lib/golang.sh" | sed -e 's/.*=go//')"
    if ! [ "$go_version" ]; then
        die "Unable to determine Go version for Kubernetes $version from hack/lib/golang.sh."
    fi
    # Strip the trailing .0. Kubernetes includes it, Go itself doesn't.
    # Ignore: See if you can use ${variable//search/replace} instead.
    # shellcheck disable=SC2001
    go_version="$(echo "$go_version" | sed -e 's/\.0$//')"
    echo "$go_version"
)

csi_prow_kind_have_kubernetes=false
# Brings up a Kubernetes cluster and sets KUBECONFIG.
# Accepts additional feature gates in the form gate1=true|false,gate2=...
start_cluster () {
    local image gates
    gates="$1"

    if kind get clusters | grep -q csi-prow; then
        run kind delete cluster --name=csi-prow || die "kind delete failed"
    fi

    # Try to find a pre-built kind image if asked to use a specific version.
    if ! [[ "${CSI_PROW_KUBERNETES_VERSION}" =~ ^release-|^latest$ ]]; then
        # Ignore: See if you can use ${variable//search/replace} instead.
        # shellcheck disable=SC2001
        major_minor=$(echo "${CSI_PROW_KUBERNETES_VERSION}" | sed -e 's/^\([0-9]*\)\.\([0-9]*\).*/\1.\2/')
        for i in ${CSI_PROW_KIND_IMAGES}; do
            if echo "$i" | grep -q "kindest/node:v${major_minor}"; then
                image="$i"
                break
            fi
        done
    fi

    # Need to build from source?
    if ! [ "$image" ]; then
        if ! ${csi_prow_kind_have_kubernetes}; then
            local version="${CSI_PROW_KUBERNETES_VERSION}"
            if [ "$version" = "latest" ]; then
                version=master
            fi
            git_clone https://github.com/kubernetes/kubernetes "${CSI_PROW_WORK}/src/kubernetes" "$(version_to_git "$version")" || die "checking out Kubernetes $version failed"

            go_version="$(go_version_for_kubernetes "${CSI_PROW_WORK}/src/kubernetes" "$version")" || die "cannot proceed without knowing Go version for Kubernetes"
            # Changing into the Kubernetes source code directory is a workaround for https://github.com/kubernetes-sigs/kind/issues/1910
            # shellcheck disable=SC2046
            (cd "${CSI_PROW_WORK}/src/kubernetes" && run_with_go "$go_version" kind build node-image --image csiprow/node:latest --kube-root "${CSI_PROW_WORK}/src/kubernetes") || die "'kind build node-image' failed"
            csi_prow_kind_have_kubernetes=true
        fi
        image="csiprow/node:latest"
    fi
    cat >"${CSI_PROW_WORK}/kind-config.yaml" <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
featureGates:
$(list_gates "$gates")
runtimeConfig:
$(list_api_groups "$gates")
EOF

    info "kind-config.yaml:"
    cat "${CSI_PROW_WORK}/kind-config.yaml"
    if ! run kind create cluster --name csi-prow --config "${CSI_PROW_WORK}/kind-config.yaml" --wait 5m --image "$image"; then
        warn "Cluster creation failed. Will try again with higher verbosity."
        info "Available Docker images:"
        docker image ls
        if ! run kind --loglevel debug create cluster --retain --name csi-prow --config "${CSI_PROW_WORK}/kind-config.yaml" --wait 5m --image "$image"; then
            run kind export logs --name csi-prow "$ARTIFACTS/kind-cluster"
            die "Cluster creation failed again, giving up. See the 'kind-cluster' artifact directory for additional logs."
        fi
    fi
    export KUBECONFIG="${HOME}/.kube/config"
}

# Deletes kind cluster inside a prow job
delete_cluster_inside_prow_job() {
    local name="$1"

    # Inside a real Prow job it is better to clean up at runtime
    # instead of leaving that to the Prow job cleanup code
    # because the later sometimes times out (https://github.com/kubernetes-csi/csi-release-tools/issues/24#issuecomment-554765872).
    #
    # This is also a good time to collect logs.
    if [ "$JOB_NAME" ]; then
        if kind get clusters | grep -q csi-prow; then
            run kind export logs --name=csi-prow "${ARTIFACTS}/cluster-logs/$name"
            run kind delete cluster --name=csi-prow || die "kind delete failed"
        fi
        unset KUBECONFIG
    fi
}

# Looks for the deployment as specified by CSI_PROW_DEPLOYMENT and CSI_PROW_KUBERNETES_VERSION
# in the given directory.
find_deployment () {
    local dir="$1"
    local file

    # major/minor without release- prefix.
    local k8sver
    # Ignore: See if you can use ${variable//search/replace} instead.
    # shellcheck disable=SC2001
    k8sver="$(echo "${CSI_PROW_KUBERNETES_VERSION}" | sed -e 's/^release-//' -e 's/\([0-9]*\)\.\([0-9]*\).*/\1.\2/')"

    # Desired deployment, either specified completely, including version, or derived from other variables.
    local deployment
    deployment=${CSI_PROW_DEPLOYMENT:-kubernetes-${k8sver}${CSI_PROW_DEPLOYMENT_SUFFIX}}

    # Fixed deployment name? Use it if it exists.
    if [ "${CSI_PROW_DEPLOYMENT}" ]; then
        file="$dir/${CSI_PROW_DEPLOYMENT}/deploy.sh"
        if [ -e "$file" ]; then
            echo "$file"
            return 0
        fi

        # CSI_PROW_DEPLOYMENT=kubernetes-x.yy must be mapped to kubernetes-latest
        # as fallback. Same for kubernetes-distributed-x.yy.
    fi

    file="$dir/${deployment}/deploy.sh"
    if ! [ -e "$file" ]; then
        # Replace the first xx.yy number with "latest", for example
        # kubernetes-1.21-test -> kubernetes-latest-test.
        # Ignore: See if you can use ${variable//search/replace} instead.
        # shellcheck disable=SC2001
        file="$dir/$(echo "$deployment" | sed -e 's/[0-9][0-9]*\.[0-9][0-9]*/latest/')/deploy.sh"
        if ! [ -e "$file" ]; then
            return 1
        fi
    fi
    echo "$file"
}

# This installs the CSI driver. It's called with a list of env variables
# that override the default images. CSI_PROW_DRIVER_CANARY overrides all
# image versions with that canary version.
install_csi_driver () {
    local images deploy_driver
    images="$*"

    if [ "${CSI_PROW_DEPLOYMENT}" = "none" ]; then
        return 1
    fi

    if ${CSI_PROW_BUILD_JOB}; then
        # Ignore: Double quote to prevent globbing and word splitting.
        # Ignore: To read lines rather than words, pipe/redirect to a 'while read' loop.
        # shellcheck disable=SC2086 disable=SC2013
        for i in $(grep '^\s*CMDS\s*=' Makefile | sed -e 's/\s*CMDS\s*=//'); do
            kind load docker-image --name csi-prow $i:csiprow || die "could not load the $i:latest image into the kind cluster"
        done
    fi

    if deploy_driver="$(find_deployment "$(pwd)/deploy")"; then
        :
    elif [ "${CSI_PROW_DRIVER_REPO}" = "none" ]; then
        return 1
    else
        git_checkout "${CSI_PROW_DRIVER_REPO}" "${CSI_PROW_WORK}/csi-driver" "${CSI_PROW_DRIVER_VERSION}" --depth=1 || die "checking out CSI driver repo failed"
        if deploy_driver="$(find_deployment "${CSI_PROW_WORK}/csi-driver/deploy")"; then
            :
        else
            die "deploy.sh not found in ${CSI_PROW_DRIVER_REPO} ${CSI_PROW_DRIVER_VERSION}. To disable E2E testing, set CSI_PROW_DRIVER_REPO=none"
        fi
    fi

    if [ "${CSI_PROW_DRIVER_CANARY}" != "stable" ]; then
      if [ "${CSI_PROW_DRIVER_CANARY}" == "canary" ]; then
        images="$images IMAGE_TAG=${CSI_PROW_DRIVER_CANARY} IMAGE_REGISTRY=${CSI_PROW_DRIVER_CANARY_REGISTRY}"
      else
        images="$images IMAGE_TAG=${CSI_PROW_DRIVER_CANARY}"
      fi
    fi
    # Ignore: Double quote to prevent globbing and word splitting.
    # It's intentional here for $images.
    # shellcheck disable=SC2086
    if ! run env "CSI_PROW_TEST_DRIVER=${CSI_PROW_WORK}/test-driver.yaml" $images "${deploy_driver}"; then
        # Collect information about failed deployment before failing.
        collect_cluster_info
        (start_loggers >/dev/null; wait)
        info "For container output see job artifacts."
        die "deploying the CSI driver with ${deploy_driver} failed"
    fi
}

# Installs all necessary snapshotter CRDs
install_snapshot_crds() {
  # Wait until volumesnapshot CRDs are in place.
  CRD_BASE_DIR="https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${CSI_SNAPSHOTTER_VERSION}/client/config/crd"
  if [[ ${REPO_DIR} == *"external-snapshotter"* ]]; then
      CRD_BASE_DIR="${REPO_DIR}/client/config/crd"
  fi
  echo "Installing snapshot CRDs from ${CRD_BASE_DIR}"
  kubectl apply -f "${CRD_BASE_DIR}/snapshot.storage.k8s.io_volumesnapshotclasses.yaml" --validate=false
  kubectl apply -f "${CRD_BASE_DIR}/snapshot.storage.k8s.io_volumesnapshots.yaml" --validate=false
  kubectl apply -f "${CRD_BASE_DIR}/snapshot.storage.k8s.io_volumesnapshotcontents.yaml" --validate=false
  cnt=0
  until kubectl get volumesnapshotclasses.snapshot.storage.k8s.io \
    && kubectl get volumesnapshots.snapshot.storage.k8s.io \
    && kubectl get volumesnapshotcontents.snapshot.storage.k8s.io; do
    if [ $cnt -gt 30 ]; then
        echo >&2 "ERROR: snapshot CRDs not ready after over 1 min"
        exit 1
    fi
    echo "$(date +%H:%M:%S)" "waiting for snapshot CRDs, attempt #$cnt"
	cnt=$((cnt + 1))
    sleep 2
  done
}

# Install snapshot controller and associated RBAC, retrying until the pod is running.
install_snapshot_controller() {
  CONTROLLER_DIR="https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/${CSI_SNAPSHOTTER_VERSION}"
  if [[ ${REPO_DIR} == *"external-snapshotter"* ]]; then
      CONTROLLER_DIR="${REPO_DIR}"
  fi
  SNAPSHOT_RBAC_YAML="${CONTROLLER_DIR}/deploy/kubernetes/snapshot-controller/rbac-snapshot-controller.yaml"
  echo "kubectl apply -f ${SNAPSHOT_RBAC_YAML}"
  # Ignore: Double quote to prevent globbing and word splitting.
  # shellcheck disable=SC2086
  kubectl apply -f ${SNAPSHOT_RBAC_YAML}

  cnt=0
  until kubectl get clusterrolebinding snapshot-controller-role; do
     if [ $cnt -gt 30 ]; then
        echo "Cluster role bindings:"
        kubectl describe clusterrolebinding
        echo >&2 "ERROR: snapshot controller RBAC not ready after over 5 min"
        exit 1
    fi
    echo "$(date +%H:%M:%S)" "waiting for snapshot RBAC setup complete, attempt #$cnt"
	cnt=$((cnt + 1))
    sleep 10
  done

  SNAPSHOT_CONTROLLER_YAML="${CONTROLLER_DIR}/deploy/kubernetes/snapshot-controller/setup-snapshot-controller.yaml"
  if [[ ${REPO_DIR} == *"external-snapshotter"* ]]; then
      # snapshot-controller image built from the PR will get a "csiprow" tag.
      # Load it into the "kind" cluster so that we can deploy it.
      NEW_TAG="csiprow"
      NEW_IMG="snapshot-controller:${NEW_TAG}"
      echo "kind load docker-image --name csi-prow ${NEW_IMG}"
      kind load docker-image --name csi-prow ${NEW_IMG} || die "could not load the snapshot-controller:csiprow image into the kind cluster"

      # deploy snapshot-controller
      echo "Deploying snapshot-controller from ${SNAPSHOT_CONTROLLER_YAML} with $NEW_IMG."
      # Replace image in SNAPSHOT_CONTROLLER_YAML with snapshot-controller:csiprow and deploy
      # NOTE: This logic is similar to the logic here:
      # https://github.com/kubernetes-csi/csi-driver-host-path/blob/v1.4.0/deploy/util/deploy-hostpath.sh#L155
      # Ignore: Double quote to prevent globbing and word splitting.
      # shellcheck disable=SC2086
      # Ignore: Use find instead of ls to better handle non-alphanumeric filenames.
      # shellcheck disable=SC2012
      for i in $(ls ${SNAPSHOT_CONTROLLER_YAML} | sort); do
          echo "   $i"
          # Ignore: Useless cat. Consider 'cmd < file | ..' or 'cmd file | ..' instead.
          # shellcheck disable=SC2002
          # Ignore: See if you can use ${variable//search/replace} instead.
          # shellcheck disable=SC2001
          modified="$(cat "$i" | while IFS= read -r line; do
              nocomments="$(echo "$line" | sed -e 's/ *#.*$//')"
              if echo "$nocomments" | grep -q '^[[:space:]]*image:[[:space:]]*'; then
                  # Split 'image: k8s.gcr.io/sig-storage/snapshot-controller:v3.0.0'
                  # into image (snapshot-controller:v3.0.0),
                  # name (snapshot-controller),
                  # tag (v3.0.0).
                  image=$(echo "$nocomments" | sed -e 's;.*image:[[:space:]]*;;')
                  name=$(echo "$image" | sed -e 's;.*/\([^:]*\).*;\1;')
                  tag=$(echo "$image" | sed -e 's;.*:;;')

                  # Now replace registry and/or tag
                  NEW_TAG="csiprow"
                  line="$(echo "$nocomments" | sed -e "s;$image;${name}:${NEW_TAG};")"
	          echo "        using $line" >&2
              fi
              echo "$line"
          done)"
          if ! echo "$modified" | kubectl apply -f -; then
              echo "modified version of $i:"
              echo "$modified"
              exit 1
          fi
      done
  elif [ "${CSI_PROW_DRIVER_CANARY}" = "canary" ]; then
      echo "Deploying snapshot-controller from ${SNAPSHOT_CONTROLLER_YAML} with canary images."
      yaml="$(kubectl apply --dry-run=client -o yaml -f "$SNAPSHOT_CONTROLLER_YAML")"
      # Ignore: See if you can use ${variable//search/replace} instead.
      # shellcheck disable=SC2001
      modified="$(echo "$yaml" | sed -e "s;image: .*/\([^/:]*\):.*;image: ${CSI_PROW_DRIVER_CANARY_REGISTRY}/\1:canary;")"
      diff <(echo "$yaml") <(echo "$modified")
      if ! echo "$modified" | kubectl apply -f -; then
          echo "modified version of $SNAPSHOT_CONTROLLER_YAML:"
          echo "$modified"
          exit 1
      fi
  else
      echo "kubectl apply -f $SNAPSHOT_CONTROLLER_YAML"
      kubectl apply -f "$SNAPSHOT_CONTROLLER_YAML"
  fi

  cnt=0
  expected_running_pods=$(kubectl apply --dry-run=client -o "jsonpath={.spec.replicas}" -f "$SNAPSHOT_CONTROLLER_YAML")
  expected_namespace=$(kubectl apply --dry-run=client -o "jsonpath={.metadata.namespace}" -f "$SNAPSHOT_CONTROLLER_YAML")
  while [ "$(kubectl get pods -n "$expected_namespace" -l app=snapshot-controller | grep 'Running' -c)" -lt "$expected_running_pods" ]; do
    if [ $cnt -gt 30 ]; then
        echo "snapshot-controller pod status:"
        kubectl describe pods -n "$expected_namespace" -l app=snapshot-controller
        echo >&2 "ERROR: snapshot controller not ready after over 5 min"
        exit 1
    fi
    echo "$(date +%H:%M:%S)" "waiting for snapshot controller deployment to complete, attempt #$cnt"
	cnt=$((cnt + 1))
    sleep 10
  done
}

# collect logs and cluster status (like the version of all components, Kubernetes version, test version)
collect_cluster_info () {
    cat <<EOF
=========================================================
Kubernetes:
$(kubectl version)

Driver installation in default namespace:
$(kubectl get all)

Images in cluster:
REPOSITORY TAG REVISION
$(
# Here we iterate over all images that are in use and print some information about them.
# The "revision" label is where our build process puts the version number and revision,
# which is always unique, in contrast to the tag (think "canary"...).
docker exec csi-prow-control-plane docker image ls --format='{{.Repository}} {{.Tag}} {{.ID}}' | grep -e csi -e hostpath | while read -r repo tag id; do
    echo "$repo" "$tag" "$(docker exec csi-prow-control-plane docker image inspect --format='{{ index .Config.Labels "revision"}}' "$id")"
done
)

=========================================================
EOF

}

# Gets logs of all containers in all namespaces. When passed -f, kubectl will
# keep running and capture new output. Prints the pid of all background processes.
# The caller must kill (when using -f) and/or wait for them.
#
# May be called multiple times and thus appends.
start_loggers () {
    kubectl get pods --all-namespaces -o go-template --template='{{range .items}}{{.metadata.namespace}} {{.metadata.name}} {{range .spec.containers}}{{.name}} {{end}}{{"\n"}}{{end}}' | while read -r namespace pod containers; do
        for container in $containers; do
            mkdir -p "${ARTIFACTS}/$namespace/$pod"
            kubectl logs -n "$namespace" "$@" "$pod" "$container" >>"${ARTIFACTS}/$namespace/$pod/$container.log" &
            echo "$!"
        done
    done
}

# Patches the image versions of test/e2e/testing-manifests/storage-csi/mock in the k/k
# source code, if needed.
patch_kubernetes () {
    local source="$1" target="$2"

    if [ "${CSI_PROW_DRIVER_CANARY}" = "canary" ]; then
        # We cannot replace k8s.gcr.io/sig-storage with gcr.io/k8s-staging-sig-storage because
        # e2e.test does not support it (see test/utils/image/manifest.go). Instead we
        # invoke the e2e.test binary with KUBE_TEST_REPO_LIST set to a file that
        # overrides that registry.
        find "$source/test/e2e/testing-manifests/storage-csi/mock" -name '*.yaml' -print0 | xargs -0 sed -i -e 's;k8s.gcr.io/sig-storage/\(.*\):v.*;k8s.gcr.io/sig-storage/\1:canary;'
        cat >"$target/e2e-repo-list" <<EOF
sigStorageRegistry: gcr.io/k8s-staging-sig-storage
EOF
        cat >&2 <<EOF

Using a modified version of k/k/test/e2e:
$(cd "$source" && git diff 2>&1)

EOF
    fi
}

# Makes the E2E test suite binary available as "${CSI_PROW_WORK}/e2e.test".
install_e2e () {
    if [ -e "${CSI_PROW_WORK}/e2e.test" ]; then
        return
    fi

    git_checkout "${CSI_PROW_E2E_REPO}" "${GOPATH}/src/${CSI_PROW_E2E_IMPORT_PATH}" "${CSI_PROW_E2E_VERSION}" --depth=1 &&
    if [ "${CSI_PROW_E2E_IMPORT_PATH}" = "k8s.io/kubernetes" ]; then
        patch_kubernetes "${GOPATH}/src/${CSI_PROW_E2E_IMPORT_PATH}" "${CSI_PROW_WORK}" &&
        go_version="${CSI_PROW_GO_VERSION_E2E:-$(go_version_for_kubernetes "${GOPATH}/src/${CSI_PROW_E2E_IMPORT_PATH}" "${CSI_PROW_E2E_VERSION}")}" &&
        run_with_go "$go_version" make WHAT=test/e2e/e2e.test "-C${GOPATH}/src/${CSI_PROW_E2E_IMPORT_PATH}" &&
        ln -s "${GOPATH}/src/${CSI_PROW_E2E_IMPORT_PATH}/_output/bin/e2e.test" "${CSI_PROW_WORK}"
    else
        run_with_go "${CSI_PROW_GO_VERSION_E2E}" go test -c -o "${CSI_PROW_WORK}/e2e.test" "${CSI_PROW_E2E_IMPORT_PATH}/test/e2e"
    fi
}

# Makes the csi-sanity test suite binary available as
# "${CSI_PROW_WORK}/csi-sanity".
install_sanity () (
    if [ -e "${CSI_PROW_WORK}/csi-sanity" ]; then
        return
    fi

    git_checkout "${CSI_PROW_SANITY_REPO}" "${GOPATH}/src/${CSI_PROW_SANITY_PACKAGE_PATH}" "${CSI_PROW_SANITY_VERSION}" --depth=1 || die "checking out csi-sanity failed"
    ( cd "${GOPATH}/src/${CSI_PROW_SANITY_PACKAGE_PATH}/cmd/csi-sanity" && run_with_go "${CSI_PROW_GO_VERSION_SANITY}" go build -o "${CSI_PROW_WORK}/csi-sanity" ) || die "building csi-sanity failed"
)

# Captures pod output while running some other command.
run_with_loggers () (
    loggers=$(start_loggers -f)
    trap 'kill $loggers' EXIT

    run "$@"
)

# Invokes the filter-junit.go tool.
run_filter_junit () {
    run_with_go "${CSI_PROW_GO_VERSION_BUILD}" go run "${RELEASE_TOOLS_ROOT}/filter-junit.go" "$@"
}

# Runs the E2E test suite in a sub-shell.
run_e2e () (
    name="$1"
    shift

    install_e2e || die "building e2e.test failed"
    install_ginkgo || die "installing ginkgo failed"

    # Rename, merge and filter JUnit files. Necessary in case that we run the E2E suite again
    # and to avoid the large number of "skipped" tests that we get from using
    # the full Kubernetes E2E testsuite while only running a few tests.
    move_junit () {
        if ls "${ARTIFACTS}"/junit_[0-9]*.xml 2>/dev/null >/dev/null; then
            run_filter_junit -t="External.Storage|CSI.mock.volume" -o "${ARTIFACTS}/junit_${name}.xml" "${ARTIFACTS}"/junit_[0-9]*.xml && rm -f "${ARTIFACTS}"/junit_[0-9]*.xml
        fi
    }
    trap move_junit EXIT

    cd "${GOPATH}/src/${CSI_PROW_E2E_IMPORT_PATH}" &&
    run_with_loggers env KUBECONFIG="$KUBECONFIG" KUBE_TEST_REPO_LIST="$(if [ -e "${CSI_PROW_WORK}/e2e-repo-list" ]; then echo "${CSI_PROW_WORK}/e2e-repo-list"; fi)" ginkgo -v "$@" "${CSI_PROW_WORK}/e2e.test" -- -report-dir "${ARTIFACTS}" -storage.testdriver="${CSI_PROW_WORK}/test-driver.yaml"
)

# Run csi-sanity against installed CSI driver.
run_sanity () (
    install_sanity || die "installing csi-sanity failed"

    if [[ "${CSI_PROW_SANITY_POD}" =~ " " ]]; then
        # Contains spaces, more complex than a simple pod name.
        # Evaluate as a shell command.
        pod=$(eval "${CSI_PROW_SANITY_POD}") || die "evaluation failed: CSI_PROW_SANITY_POD=${CSI_PROW_SANITY_POD}"
    else
        pod="${CSI_PROW_SANITY_POD}"
    fi

    cat >"${CSI_PROW_WORK}/mkdir_in_pod.sh" <<EOF
#!/bin/sh
kubectl exec "$pod" -c "${CSI_PROW_SANITY_CONTAINER}" -- mkdir "\$@" && echo "\$@"
EOF
    # Using "rm -rf" as fallback for "rmdir" is a workaround for:
    # Node Service
    #     should work
    # /nvme/gopath.tmp/src/github.com/kubernetes-csi/csi-test/pkg/sanity/node.go:624
    # STEP: reusing connection to CSI driver at dns:///172.17.0.2:30896
    # STEP: creating mount and staging directories
    # STEP: creating a single node writer volume
    # STEP: getting a node id
    # STEP: node staging volume
    # STEP: publishing the volume on a node
    # STEP: cleaning up calling nodeunpublish
    # STEP: cleaning up calling nodeunstage
    # STEP: cleaning up deleting the volume
    # cleanup: deleting sanity-node-full-35A55673-604D59E1 = 5211b280-4fad-11e9-8127-0242dfe2bdaf
    # cleanup: warning: NodeUnpublishVolume: rpc error: code = NotFound desc = volume id 5211b280-4fad-11e9-8127-0242dfe2bdaf does not exit in the volumes list
    # rmdir: '/tmp/mount': Directory not empty
    # command terminated with exit code 1
    #
    # Somehow the mount directory was not empty. All tests after that
    # failed in "mkdir".  This only occurred once, so its uncertain
    # why it happened.
    cat >"${CSI_PROW_WORK}/rmdir_in_pod.sh" <<EOF
#!/bin/sh
if ! kubectl exec "$pod" -c "${CSI_PROW_SANITY_CONTAINER}" -- rmdir "\$@"; then
    kubectl exec "$pod" -c "${CSI_PROW_SANITY_CONTAINER}" -- rm -rf "\$@"
    exit 1
fi
EOF

    cat >"${CSI_PROW_WORK}/checkdir_in_pod.sh" <<EOF
#!/bin/sh
CHECK_PATH=\$(cat <<SCRIPT
if [ -f "\$@" ]; then
    echo "file"
elif [ -d "\$@" ]; then
    echo "directory"
elif [ -e "\$@" ]; then
    echo "other"
else
    echo "not_found"
fi
SCRIPT
)
kubectl exec "$pod" -c "${CSI_PROW_SANITY_CONTAINER}" -- /bin/sh -c "\${CHECK_PATH}"
EOF

    chmod u+x "${CSI_PROW_WORK}"/*dir_in_pod.sh

    # This cannot run in parallel, because -csi.junitfile output
    # from different Ginkgo nodes would go to the same file. Also the
    # staging and target directories are the same.
    run_with_loggers "${CSI_PROW_WORK}/csi-sanity" \
                     -ginkgo.v \
                     -csi.junitfile "${ARTIFACTS}/junit_sanity.xml" \
                     -csi.endpoint "dns:///$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' csi-prow-control-plane):$(kubectl get "services/${CSI_PROW_SANITY_SERVICE}" -o "jsonpath={..nodePort}")" \
                     -csi.stagingdir "/tmp/staging" \
                     -csi.mountdir "/tmp/mount" \
                     -csi.createstagingpathcmd "${CSI_PROW_WORK}/mkdir_in_pod.sh" \
                     -csi.createmountpathcmd "${CSI_PROW_WORK}/mkdir_in_pod.sh" \
                     -csi.removestagingpathcmd "${CSI_PROW_WORK}/rmdir_in_pod.sh" \
                     -csi.removemountpathcmd "${CSI_PROW_WORK}/rmdir_in_pod.sh" \
                     -csi.checkpathcmd "${CSI_PROW_WORK}/checkdir_in_pod.sh" \
)

ascii_to_xml () {
    # We must escape special characters and remove escape sequences
    # (no good representation in the simple XML that we generate
    # here). filter_junit.go would choke on them during decoding, even
    # when disabling strict parsing.
    sed -e 's/&/&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g' -e 's/\x1B...//g'
}

# The "make test" output starts each test with "### <test-target>:"
# and then ends when the next test starts or with "make: ***
# [<test-target>] Error 1" when there was a failure. Here we read each
# line of that output, split it up into individual tests and generate
# a make-test.xml file in JUnit format.
make_test_to_junit () {
    local ret out testname testoutput
    ret=0
    # Plain make-test.xml was not delivered as text/xml by the web
    # server and ignored by spyglass. It seems that the name has to
    # match junit*.xml.
    out="${ARTIFACTS}/junit_make_test.xml"
    testname=
    echo "<testsuite>" >>"$out"

    while IFS= read -r line; do
        echo "$line" # pass through
        if echo "$line" | grep -q "^### [^ ]*:$"; then
            if [ "$testname" ]; then
                # previous test successful
                echo "    </system-out>" >>"$out"
                echo "  </testcase>" >>"$out"
            fi
            # Ignore: See if you can use ${variable//search/replace} instead.
            # shellcheck disable=SC2001
            #
            # start new test
            testname="$(echo "$line" | sed -e 's/^### \([^ ]*\):$/\1/')"
            testoutput=
            echo "  <testcase name=\"$testname\">" >>"$out"
            echo "    <system-out>" >>"$out"
        elif echo "$line" | grep -q '^make: .*Error [0-9]*$'; then
            if [ "$testname" ]; then
                # Ignore: Consider using { cmd1; cmd2; } >> file instead of individual redirects.
                # shellcheck disable=SC2129
                #
                # end test with failure
                echo "    </system-out>" >>"$out"
                # Include the same text as in <system-out> also in <failure>,
                # because then it is easier to view in spyglass (shown directly
                # instead of having to click through to stdout).
                echo "    <failure>" >>"$out"
                echo -n "$testoutput" | ascii_to_xml >>"$out"
                echo "    </failure>" >>"$out"
                echo "  </testcase>" >>"$out"
            fi
            # remember failure for exit code
            ret=1
            # not currently inside a test
            testname=
        else
            if [ "$testname" ]; then
                # Test output.
                echo "$line" | ascii_to_xml >>"$out"
                testoutput="$testoutput$line
"
            fi
        fi
    done
    # if still in a test, close it now
    if [ "$testname" ]; then
        echo "    </system-out>" >>"$out"
        echo "  </testcase>" >>"$out"
    fi
    echo "</testsuite>" >>"$out"

    # this makes the error more visible in spyglass
    if [ "$ret" -ne 0 ]; then
        echo "ERROR: 'make test' failed"
        return 1
    fi
}

# version_gt returns true if arg1 is greater than arg2.
#
# This function expects versions to be one of the following formats:
#   X.Y.Z, release-X.Y.Z, vX.Y.Z
#
#   where X,Y, and Z are any number.
#
# Partial versions (1.2, release-1.2) work as well.
# The follow substrings are stripped before version comparison:
#   - "v"
#   - "release-"
#   - "kubernetes-"
#
# Usage:
# version_gt release-1.3 v1.2.0  (returns true)
# version_gt v1.1.1 v1.2.0  (returns false)
# version_gt 1.1.1 v1.2.0  (returns false)
# version_gt 1.3.1 v1.2.0  (returns true)
# version_gt 1.1.1 release-1.2.0  (returns false)
# version_gt 1.2.0 1.2.2  (returns false)
function version_gt() {
    versions=$(for ver in "$@"; do ver=${ver#release-}; ver=${ver#kubernetes-}; echo "${ver#v}"; done)
    greaterVersion=${1#"release-"};
    greaterVersion=${greaterVersion#"kubernetes-"};
    greaterVersion=${greaterVersion#"v"};
    test "$(printf '%s' "$versions" | sort -V | head -n 1)" != "$greaterVersion"
}

main () {
    local images ret
    ret=0

    # Set up work directory.
    ensure_paths

    images=
    if ${CSI_PROW_BUILD_JOB}; then
        # A successful build is required for testing.
        run_with_go "${CSI_PROW_GO_VERSION_BUILD}" make all "GOFLAGS_VENDOR=${GOFLAGS_VENDOR}" "BUILD_PLATFORMS=${CSI_PROW_BUILD_PLATFORMS}" || die "'make all' failed"
        # We don't want test failures to prevent E2E testing below, because the failure
        # might have been minor or unavoidable, for example when experimenting with
        # changes in "release-tools" in a PR (that fails the "is release-tools unmodified"
        # test).
        if tests_enabled "unit"; then
            if [ -f Gopkg.toml ] && ! install_dep; then
                warn "installing 'dep' failed, cannot test vendoring"
                ret=1
            fi
            if ! run_with_go "${CSI_PROW_GO_VERSION_BUILD}" make -k test "GOFLAGS_VENDOR=${GOFLAGS_VENDOR}" 2>&1 | make_test_to_junit; then
                warn "'make test' failed, proceeding anyway"
                ret=1
            fi
        fi
        # Required for E2E testing.
        run_with_go "${CSI_PROW_GO_VERSION_BUILD}" make container "GOFLAGS_VENDOR=${GOFLAGS_VENDOR}" || die "'make container' failed"
    fi

    if tests_need_kind; then
        install_kind || die "installing kind failed"

        if ${CSI_PROW_BUILD_JOB}; then
            cmds="$(grep '^\s*CMDS\s*=' Makefile | sed -e 's/\s*CMDS\s*=//')"
            # Get the image that was just built (if any) from the
            # top-level Makefile CMDS variable and set the
            # deploy.sh env variables for it. We also need to
            # side-load those images into the cluster.
            for i in $cmds; do
                e=$(echo "$i" | tr '[:lower:]' '[:upper:]' | tr - _)
                images="$images ${e}_REGISTRY=none ${e}_TAG=csiprow"

                # We must avoid the tag "latest" because that implies
                # always pulling the image
                # (https://github.com/kubernetes-sigs/kind/issues/328).
                docker tag "$i:latest" "$i:csiprow" || die "tagging the locally built container image for $i failed"

                # For components with multiple cmds, the RBAC file should be in the following format:
                #   rbac-$cmd.yaml
                # If this file cannot be found, we can default to the standard location:
                #   deploy/kubernetes/rbac.yaml
                rbac_file_path=$(find . -type f -name "rbac-$i.yaml")
                if [ "$rbac_file_path" == "" ]; then
                    rbac_file_path="$(pwd)/deploy/kubernetes/rbac.yaml"
                fi

                if [ -e "$rbac_file_path" ]; then
                    # This is one of those components which has its own RBAC rules (like external-provisioner).
                    # We are testing a locally built image and also want to test with the the current,
                    # potentially modified RBAC rules.
                    e=$(echo "$i" | tr '[:lower:]' '[:upper:]' | tr - _)
                    images="$images ${e}_RBAC=$rbac_file_path"
                fi
            done
        fi

        # Run the external driver tests and optionally also mock tests.
        local focus="External.Storage"
        if "$CSI_PROW_E2E_MOCK"; then
            focus="($focus|CSI.mock.volume)"
        fi

        if tests_need_non_alpha_cluster; then
            start_cluster || die "starting the non-alpha cluster failed"

            # Install necessary snapshot CRDs and snapshot controller
            install_snapshot_crds
            install_snapshot_controller


            # Installing the driver might be disabled.
            if ${CSI_PROW_DRIVER_INSTALL} "$images"; then
                collect_cluster_info

                if sanity_enabled; then
                    if ! run_sanity; then
                        ret=1
                    fi
                fi

                if tests_enabled "parallel"; then
                    # Ignore: Double quote to prevent globbing and word splitting.
                    # shellcheck disable=SC2086
                    if ! run_e2e parallel ${CSI_PROW_GINKO_PARALLEL} \
                         -focus="$focus" \
                         -skip="$(regex_join "${CSI_PROW_E2E_SERIAL}" "${CSI_PROW_E2E_ALPHA}" "${CSI_PROW_E2E_SKIP}")"; then
                        warn "E2E parallel failed"
                        ret=1
                    fi

                    # Run tests that are feature tagged, but non-alpha
                    # Ignore: Double quote to prevent globbing and word splitting.
                    # shellcheck disable=SC2086
                    if ! run_e2e parallel-features ${CSI_PROW_GINKO_PARALLEL} \
                         -focus="$focus.*($(regex_join "${CSI_PROW_E2E_FOCUS}"))" \
                         -skip="$(regex_join "${CSI_PROW_E2E_SERIAL}")"; then
                        warn "E2E parallel features failed"
                        ret=1
                    fi
                fi

                if tests_enabled "serial"; then
                    if ! run_e2e serial \
                         -focus="$focus.*($(regex_join "${CSI_PROW_E2E_SERIAL}"))" \
                         -skip="$(regex_join "${CSI_PROW_E2E_ALPHA}" "${CSI_PROW_E2E_SKIP}")"; then
                        warn "E2E serial failed"
                        ret=1
                    fi
                fi
            fi
            delete_cluster_inside_prow_job non-alpha
        fi

        if tests_need_alpha_cluster && [ "${CSI_PROW_E2E_ALPHA_GATES}" ]; then
            # Need to (re)create the cluster.
            start_cluster "${CSI_PROW_E2E_ALPHA_GATES}" || die "starting alpha cluster failed"

            # Install necessary snapshot CRDs and snapshot controller
            install_snapshot_crds
            install_snapshot_controller

            # Installing the driver might be disabled.
            if ${CSI_PROW_DRIVER_INSTALL} "$images"; then
                collect_cluster_info

                if tests_enabled "parallel-alpha"; then
                    # Ignore: Double quote to prevent globbing and word splitting.
                    # shellcheck disable=SC2086
                    if ! run_e2e parallel-alpha ${CSI_PROW_GINKO_PARALLEL} \
                         -focus="$focus.*($(regex_join "${CSI_PROW_E2E_ALPHA}"))" \
                         -skip="$(regex_join "${CSI_PROW_E2E_SERIAL}" "${CSI_PROW_E2E_SKIP}")"; then
                        warn "E2E parallel alpha failed"
                        ret=1
                    fi
                fi

                if tests_enabled "serial-alpha"; then
                    if ! run_e2e serial-alpha \
                         -focus="$focus.*(($(regex_join "${CSI_PROW_E2E_SERIAL}")).*($(regex_join "${CSI_PROW_E2E_ALPHA}"))|($(regex_join "${CSI_PROW_E2E_ALPHA}")).*($(regex_join "${CSI_PROW_E2E_SERIAL}")))" \
                         -skip="$(regex_join "${CSI_PROW_E2E_SKIP}")"; then
                        warn "E2E serial alpha failed"
                        ret=1
                    fi
                fi
            fi
            delete_cluster_inside_prow_job alpha
        fi
    fi

    # Merge all junit files into one. This gets rid of duplicated "skipped" tests.
    if ls "${ARTIFACTS}"/junit_*.xml 2>/dev/null >&2; then
        run_filter_junit -o "${CSI_PROW_WORK}/junit_final.xml" "${ARTIFACTS}"/junit_*.xml && rm "${ARTIFACTS}"/junit_*.xml && mv "${CSI_PROW_WORK}/junit_final.xml" "${ARTIFACTS}"
    fi

    return "$ret"
}

# This function can be called by a repo's top-level cloudbuild.sh:
# it handles environment set up in the GCR cloud build and then
# invokes "make push-multiarch" to do the actual image building.
gcr_cloud_build () {
    # Register gcloud as a Docker credential helper.
    # Required for "docker buildx build --push".
    gcloud auth configure-docker

    # Might not be needed here, but call it just in case.
    ensure_paths

    if find . -name Dockerfile | grep -v ^./vendor | xargs --no-run-if-empty cat | grep -q ^RUN; then
        # Needed for "RUN" steps on non-linux/amd64 platforms.
        # See https://github.com/multiarch/qemu-user-static#getting-started
        (set -x; docker run --rm --privileged multiarch/qemu-user-static --reset -p yes)
    fi

    # Extract tag-n-hash value from GIT_TAG (form vYYYYMMDD-tag-n-hash) for REV value.
    REV=v$(echo "$GIT_TAG" | cut -f3- -d 'v')

    run_with_go "${CSI_PROW_GO_VERSION_BUILD}" make push-multiarch REV="${REV}" REGISTRY_NAME="${REGISTRY_NAME}" BUILD_PLATFORMS="${CSI_PROW_BUILD_PLATFORMS}"
}
