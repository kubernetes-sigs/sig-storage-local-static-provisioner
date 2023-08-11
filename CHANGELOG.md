# HEAD

# [v2.6.0](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/releases/tag/v2.6.0)

### Feature

- Add metrics to local PV node cleanup controller. ([#399](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/399), [@justinblalock87](https://github.com/justinblalock87))
- Feat: add enableWindows helm chart value to control the deployment of Windows manifests ([#388](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/388), [@jennwah](https://github.com/jennwah))
- Helm Chart values flattened when possible, please check the CHANGELOG.md file inside the helm/ directory ([#393](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/393), [@mauriciopoppe](https://github.com/mauriciopoppe))
- Optional controller to automatically clean up stale PV/PVC objects when a Node is deleted ([#385](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/385), [@justinblalock87](https://github.com/justinblalock87))
- Support for watching ConfigMap changes and restarting main sync loop including informer and job controller (if specified) ([#265](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/265), [@yibozhuang](https://github.com/yibozhuang))

### Documentation

- Helm chart v1.0.0 uses registry.k8s.io/sig-storage/local-volume-provisioner:v2.5.0
  Add field .Values.daemonset.nodeSelectorWindows to the helm chart. ([#353](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/353), [@mauriciopoppe](https://github.com/mauriciopoppe))

### Bug or Regression

- Fix: CVE-2022-1996
  fix: CVE-2022-29526 ([#335](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/335), [@umagnus](https://github.com/umagnus))
- Fix: CVE-2022-27664 ([#339](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/339), [@andyzhangx](https://github.com/andyzhangx))
- Fix: CVE-2022-32149 ([#342](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/342), [@andyzhangx](https://github.com/andyzhangx))
- Fix: CVE-2022-41723 ([#367](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/367), [@andyzhangx](https://github.com/andyzhangx))
- Fix: CVE-2023-2431 ([#383](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/383), [@andyzhangx](https://github.com/andyzhangx))
- Fix: set admin user in windows image build ([#338](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/338), [@andyzhangx](https://github.com/andyzhangx))

### Other (Cleanup or Flake)

- Chore: replace unmaintained `github.com/ghodss/yaml` dependency with `sigs.k8s.io/yaml` ([#387](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/387), [@Juneezee](https://github.com/Juneezee))
- Images are no longer published on quay.io. Use registry.k8s.io for image access. ([#394](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/394), [@msau42](https://github.com/msau42))

### Uncategorized

- Cleanup: remove Windows 20H2 image build since 20H2 is not maintained and supported any more from more than 1 year ago ([#382](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/382), [@andyzhangx](https://github.com/andyzhangx))

## Dependencies

### Added
- github.com/blang/semver/v4: [v4.0.0](https://github.com/blang/semver/v4/tree/v4.0.0)
- github.com/cenkalti/backoff/v4: [v4.1.3](https://github.com/cenkalti/backoff/v4/tree/v4.1.3)
- github.com/emicklei/go-restful/v3: [v3.9.0](https://github.com/emicklei/go-restful/v3/tree/v3.9.0)
- github.com/go-logr/stdr: [v1.2.2](https://github.com/go-logr/stdr/tree/v1.2.2)
- github.com/go-task/slim-sprig: [348f09d](https://github.com/go-task/slim-sprig/tree/348f09d)
- github.com/golang-jwt/jwt/v4: [v4.4.2](https://github.com/golang-jwt/jwt/v4/tree/v4.4.2)
- github.com/golang/snappy: [v0.0.3](https://github.com/golang/snappy/tree/v0.0.3)
- github.com/grpc-ecosystem/grpc-gateway/v2: [v2.7.0](https://github.com/grpc-ecosystem/grpc-gateway/v2/tree/v2.7.0)
- github.com/onsi/ginkgo/v2: [v2.9.1](https://github.com/onsi/ginkgo/v2/tree/v2.9.1)
- go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful: v0.35.0
- go.opentelemetry.io/otel/exporters/otlp/internal/retry: v1.10.0
- go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc: v1.10.0
- go.opentelemetry.io/otel/exporters/otlp/otlptrace: v1.10.0
- google.golang.org/grpc/cmd/protoc-gen-go-grpc: v1.1.0
- k8s.io/dynamic-resource-allocation: v0.27.3
- k8s.io/kms: v0.27.3
- sigs.k8s.io/kustomize/kustomize/v5: v5.0.1

### Changed
- bitbucket.org/bertimus9/systemstat: 0eeff89 → v0.5.0
- cloud.google.com/go: v0.81.0 → v0.97.0
- dmitri.shuralyov.com/gpu/mtl: 28db891 → 666a987
- github.com/Azure/go-autorest/autorest/adal: [v0.9.13 → v0.9.20](https://github.com/Azure/go-autorest/autorest/adal/compare/v0.9.13...v0.9.20)
- github.com/Azure/go-autorest/autorest/mocks: [v0.4.1 → v0.4.2](https://github.com/Azure/go-autorest/autorest/mocks/compare/v0.4.1...v0.4.2)
- github.com/Azure/go-autorest/autorest: [v0.11.18 → v0.11.27](https://github.com/Azure/go-autorest/autorest/compare/v0.11.18...v0.11.27)
- github.com/GoogleCloudPlatform/k8s-cloud-provider: [ea6160c → f118173](https://github.com/GoogleCloudPlatform/k8s-cloud-provider/compare/ea6160c...f118173)
- github.com/MakeNowJust/heredoc: [bb23615 → v1.0.0](https://github.com/MakeNowJust/heredoc/compare/bb23615...v1.0.0)
- github.com/Microsoft/hcsshim: [v0.8.22 → v0.8.25](https://github.com/Microsoft/hcsshim/compare/v0.8.22...v0.8.25)
- github.com/antlr/antlr4/runtime/Go/antlr: [b48c857 → v1.4.10](https://github.com/antlr/antlr4/runtime/Go/antlr/compare/b48c857...v1.4.10)
- github.com/chai2010/gettext-go: [c6fed77 → v1.0.2](https://github.com/chai2010/gettext-go/compare/c6fed77...v1.0.2)
- github.com/checkpoint-restore/go-criu/v5: [v5.0.0 → v5.3.0](https://github.com/checkpoint-restore/go-criu/v5/compare/v5.0.0...v5.3.0)
- github.com/cilium/ebpf: [v0.6.2 → v0.7.0](https://github.com/cilium/ebpf/compare/v0.6.2...v0.7.0)
- github.com/cncf/udpa/go: [5459f2c → 04548b0](https://github.com/cncf/udpa/go/compare/5459f2c...04548b0)
- github.com/cncf/xds/go: [fbca930 → cb28da3](https://github.com/cncf/xds/go/compare/fbca930...cb28da3)
- github.com/container-storage-interface/spec: [v1.5.0 → v1.7.0](https://github.com/container-storage-interface/spec/compare/v1.5.0...v1.7.0)
- github.com/containerd/console: [v1.0.2 → v1.0.3](https://github.com/containerd/console/compare/v1.0.2...v1.0.3)
- github.com/containerd/ttrpc: [v1.0.2 → v1.1.0](https://github.com/containerd/ttrpc/compare/v1.0.2...v1.1.0)
- github.com/coredns/corefile-migration: [v1.0.14 → v1.0.20](https://github.com/coredns/corefile-migration/compare/v1.0.14...v1.0.20)
- github.com/coreos/go-systemd/v22: [v22.3.2 → v22.4.0](https://github.com/coreos/go-systemd/v22/compare/v22.3.2...v22.4.0)
- github.com/cpuguy83/go-md2man/v2: [v2.0.1 → v2.0.2](https://github.com/cpuguy83/go-md2man/v2/compare/v2.0.1...v2.0.2)
- github.com/creack/pty: [v1.1.11 → v1.1.9](https://github.com/creack/pty/compare/v1.1.11...v1.1.9)
- github.com/cyphar/filepath-securejoin: [v0.2.2 → v0.2.3](https://github.com/cyphar/filepath-securejoin/compare/v0.2.2...v0.2.3)
- github.com/daviddengcn/go-colortext: [511bcaf → v1.0.0](https://github.com/daviddengcn/go-colortext/compare/511bcaf...v1.0.0)
- github.com/dnaeon/go-vcr: [v1.0.1 → v1.2.0](https://github.com/dnaeon/go-vcr/compare/v1.0.1...v1.2.0)
- github.com/docker/distribution: [v2.7.1+incompatible → v2.8.1+incompatible](https://github.com/docker/distribution/compare/v2.7.1...v2.8.1)
- github.com/docker/go-units: [v0.4.0 → v0.5.0](https://github.com/docker/go-units/compare/v0.4.0...v0.5.0)
- github.com/envoyproxy/go-control-plane: [63b5d3c → 49ff273](https://github.com/envoyproxy/go-control-plane/compare/63b5d3c...49ff273)
- github.com/felixge/httpsnoop: [v1.0.1 → v1.0.3](https://github.com/felixge/httpsnoop/compare/v1.0.1...v1.0.3)
- github.com/fsnotify/fsnotify: [v1.4.9 → v1.6.0](https://github.com/fsnotify/fsnotify/compare/v1.4.9...v1.6.0)
- github.com/go-errors/errors: [v1.0.1 → v1.4.2](https://github.com/go-errors/errors/compare/v1.0.1...v1.4.2)
- github.com/go-kit/log: [v0.1.0 → v0.2.0](https://github.com/go-kit/log/compare/v0.1.0...v0.2.0)
- github.com/go-logfmt/logfmt: [v0.5.0 → v0.5.1](https://github.com/go-logfmt/logfmt/compare/v0.5.0...v0.5.1)
- github.com/go-logr/logr: [v1.2.0 → v1.2.3](https://github.com/go-logr/logr/compare/v1.2.0...v1.2.3)
- github.com/go-logr/zapr: [v1.2.0 → v1.2.3](https://github.com/go-logr/zapr/compare/v1.2.0...v1.2.3)
- github.com/go-openapi/jsonpointer: [v0.19.5 → v0.19.6](https://github.com/go-openapi/jsonpointer/compare/v0.19.5...v0.19.6)
- github.com/go-openapi/jsonreference: [v0.19.5 → v0.20.1](https://github.com/go-openapi/jsonreference/compare/v0.19.5...v0.20.1)
- github.com/go-openapi/swag: [v0.19.14 → v0.22.3](https://github.com/go-openapi/swag/compare/v0.19.14...v0.22.3)
- github.com/godbus/dbus/v5: [v5.0.4 → v5.0.6](https://github.com/godbus/dbus/v5/compare/v5.0.4...v5.0.6)
- github.com/golang/mock: [v1.5.0 → v1.6.0](https://github.com/golang/mock/compare/v1.5.0...v1.6.0)
- github.com/golang/protobuf: [v1.5.2 → v1.5.3](https://github.com/golang/protobuf/compare/v1.5.2...v1.5.3)
- github.com/google/cadvisor: [v0.43.0 → v0.47.1](https://github.com/google/cadvisor/compare/v0.43.0...v0.47.1)
- github.com/google/cel-go: [v0.10.1 → v0.12.6](https://github.com/google/cel-go/compare/v0.10.1...v0.12.6)
- github.com/google/go-cmp: [v0.5.5 → v0.5.9](https://github.com/google/go-cmp/compare/v0.5.5...v0.5.9)
- github.com/google/martian/v3: [v3.1.0 → v3.2.1](https://github.com/google/martian/v3/compare/v3.1.0...v3.2.1)
- github.com/google/pprof: [cbba55b → 4bb14d4](https://github.com/google/pprof/compare/cbba55b...4bb14d4)
- github.com/google/uuid: [v1.1.2 → v1.3.0](https://github.com/google/uuid/compare/v1.1.2...v1.3.0)
- github.com/googleapis/gax-go/v2: [v2.0.5 → v2.1.1](https://github.com/googleapis/gax-go/v2/compare/v2.0.5...v2.1.1)
- github.com/imdario/mergo: [v0.3.5 → v0.3.6](https://github.com/imdario/mergo/compare/v0.3.5...v0.3.6)
- github.com/inconshreveable/mousetrap: [v1.0.0 → v1.0.1](https://github.com/inconshreveable/mousetrap/compare/v1.0.0...v1.0.1)
- github.com/karrick/godirwalk: [v1.16.1 → v1.17.0](https://github.com/karrick/godirwalk/compare/v1.16.1...v1.17.0)
- github.com/kr/pretty: [v0.2.1 → v0.3.0](https://github.com/kr/pretty/compare/v0.2.1...v0.3.0)
- github.com/mailru/easyjson: [v0.7.6 → v0.7.7](https://github.com/mailru/easyjson/compare/v0.7.6...v0.7.7)
- github.com/matttproud/golang_protobuf_extensions: [c182aff → v1.0.2](https://github.com/matttproud/golang_protobuf_extensions/compare/c182aff...v1.0.2)
- github.com/moby/ipvs: [v1.0.1 → v1.1.0](https://github.com/moby/ipvs/compare/v1.0.1...v1.1.0)
- github.com/moby/sys/mountinfo: [v0.4.1 → v0.6.2](https://github.com/moby/sys/mountinfo/compare/v0.4.1...v0.6.2)
- github.com/moby/term: [3f7ff69 → 1aeaba8](https://github.com/moby/term/compare/3f7ff69...1aeaba8)
- github.com/nxadm/tail: [v1.4.4 → v1.4.8](https://github.com/nxadm/tail/compare/v1.4.4...v1.4.8)
- github.com/onsi/ginkgo: [v1.14.0 → v1.16.4](https://github.com/onsi/ginkgo/compare/v1.14.0...v1.16.4)
- github.com/onsi/gomega: [v1.10.1 → v1.27.4](https://github.com/onsi/gomega/compare/v1.10.1...v1.27.4)
- github.com/opencontainers/runc: [v1.0.3 → v1.1.6](https://github.com/opencontainers/runc/compare/v1.0.3...v1.1.6)
- github.com/opencontainers/runtime-spec: [1c3f411 → 494a5a6](https://github.com/opencontainers/runtime-spec/compare/1c3f411...494a5a6)
- github.com/opencontainers/selinux: [v1.8.2 → v1.10.0](https://github.com/opencontainers/selinux/compare/v1.8.2...v1.10.0)
- github.com/pquerna/cachecontrol: [0dec1b3 → v0.1.0](https://github.com/pquerna/cachecontrol/compare/0dec1b3...v0.1.0)
- github.com/prometheus/client_golang: [v1.12.1 → v1.14.0](https://github.com/prometheus/client_golang/compare/v1.12.1...v1.14.0)
- github.com/prometheus/client_model: [v0.2.0 → v0.3.0](https://github.com/prometheus/client_model/compare/v0.2.0...v0.3.0)
- github.com/prometheus/common: [v0.32.1 → v0.37.0](https://github.com/prometheus/common/compare/v0.32.1...v0.37.0)
- github.com/prometheus/procfs: [v0.7.3 → v0.8.0](https://github.com/prometheus/procfs/compare/v0.7.3...v0.8.0)
- github.com/rogpeppe/go-internal: [v1.3.0 → v1.10.0](https://github.com/rogpeppe/go-internal/compare/v1.3.0...v1.10.0)
- github.com/seccomp/libseccomp-golang: [v0.9.1 → f33da4d](https://github.com/seccomp/libseccomp-golang/compare/v0.9.1...f33da4d)
- github.com/sirupsen/logrus: [v1.8.1 → v1.9.0](https://github.com/sirupsen/logrus/compare/v1.8.1...v1.9.0)
- github.com/spf13/afero: [v1.6.0 → v1.2.2](https://github.com/spf13/afero/compare/v1.6.0...v1.2.2)
- github.com/spf13/cobra: [v1.4.0 → v1.6.0](https://github.com/spf13/cobra/compare/v1.4.0...v1.6.0)
- github.com/stretchr/objx: [v0.2.0 → v0.5.0](https://github.com/stretchr/objx/compare/v0.2.0...v0.5.0)
- github.com/stretchr/testify: [v1.7.0 → v1.8.1](https://github.com/stretchr/testify/compare/v1.7.0...v1.8.1)
- github.com/tmc/grpc-websocket-proxy: [e5319fd → 673ab2c](https://github.com/tmc/grpc-websocket-proxy/compare/e5319fd...673ab2c)
- github.com/vishvananda/netns: [db3c7e5 → v0.0.2](https://github.com/vishvananda/netns/compare/db3c7e5...v0.0.2)
- github.com/vmware/govmomi: [v0.20.3 → v0.30.0](https://github.com/vmware/govmomi/compare/v0.20.3...v0.30.0)
- github.com/xlab/treeprint: [a009c39 → v1.1.0](https://github.com/xlab/treeprint/compare/a009c39...v1.1.0)
- github.com/yuin/goldmark: [v1.4.1 → v1.4.13](https://github.com/yuin/goldmark/compare/v1.4.1...v1.4.13)
- go.etcd.io/etcd/api/v3: v3.5.0 → v3.5.7
- go.etcd.io/etcd/client/pkg/v3: v3.5.0 → v3.5.7
- go.etcd.io/etcd/client/v2: v2.305.0 → v2.305.7
- go.etcd.io/etcd/client/v3: v3.5.0 → v3.5.7
- go.etcd.io/etcd/pkg/v3: v3.5.0 → v3.5.7
- go.etcd.io/etcd/raft/v3: v3.5.0 → v3.5.7
- go.etcd.io/etcd/server/v3: v3.5.0 → v3.5.7
- go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc: v0.20.0 → v0.35.0
- go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp: v0.20.0 → v0.35.1
- go.opentelemetry.io/otel/metric: v0.20.0 → v0.31.0
- go.opentelemetry.io/otel/sdk: v0.20.0 → v1.10.0
- go.opentelemetry.io/otel/trace: v0.20.0 → v1.10.0
- go.opentelemetry.io/otel: v0.20.0 → v1.10.0
- go.opentelemetry.io/proto/otlp: v0.7.0 → v0.19.0
- go.uber.org/goleak: v1.1.10 → v1.2.1
- golang.org/x/crypto: 4661260 → v0.1.0
- golang.org/x/exp: 85be41e → 6cc2880
- golang.org/x/mobile: e6ae53a → d2bd2a2
- golang.org/x/mod: 9b9b3d8 → v0.9.0
- golang.org/x/net: cd36cc0 → v0.8.0
- golang.org/x/oauth2: d3ed0bb → ee48083
- golang.org/x/sync: 036812b → v0.1.0
- golang.org/x/sys: 3681064 → v0.6.0
- golang.org/x/term: 03fcf44 → v0.6.0
- golang.org/x/text: v0.3.7 → v0.8.0
- golang.org/x/tools: 897bd77 → v0.7.0
- golang.org/x/xerrors: 5ec99f8 → 04be3eb
- google.golang.org/api: v0.46.0 → v0.60.0
- google.golang.org/genproto: 42d7afd → c8bf987
- google.golang.org/grpc: v1.40.0 → v1.51.0
- google.golang.org/protobuf: v1.27.1 → v1.28.1
- gopkg.in/check.v1: 8fa4692 → 10cb982
- gopkg.in/square/go-jose.v2: v2.2.2 → v2.6.0
- gopkg.in/yaml.v3: 496545a → v3.0.1
- k8s.io/api: v0.24.0-alpha.4 → v0.27.3
- k8s.io/apiextensions-apiserver: v0.24.0-alpha.4 → v0.27.3
- k8s.io/apimachinery: v0.24.0-alpha.4 → v0.27.3
- k8s.io/apiserver: v0.24.0-alpha.4 → v0.27.3
- k8s.io/cli-runtime: v0.24.0-alpha.4 → v0.27.3
- k8s.io/client-go: v0.24.0-alpha.4 → v0.27.3
- k8s.io/cloud-provider: v0.24.0-alpha.4 → v0.27.3
- k8s.io/cluster-bootstrap: v0.24.0-alpha.4 → v0.27.3
- k8s.io/code-generator: v0.24.0-alpha.4 → v0.27.3
- k8s.io/component-base: v0.24.0-alpha.4 → v0.27.3
- k8s.io/component-helpers: v0.24.0-alpha.4 → v0.27.3
- k8s.io/controller-manager: v0.24.0-alpha.4 → v0.27.3
- k8s.io/cri-api: v0.24.0-alpha.4 → v0.27.3
- k8s.io/csi-translation-lib: v0.24.0-alpha.4 → v0.27.3
- k8s.io/gengo: c02415c → c0856e2
- k8s.io/klog/v2: v2.40.1 → v2.90.1
- k8s.io/kube-aggregator: v0.24.0-alpha.4 → v0.27.3
- k8s.io/kube-controller-manager: v0.24.0-alpha.4 → v0.27.3
- k8s.io/kube-openapi: ddc6692 → 8b0f38b
- k8s.io/kube-proxy: v0.24.0-alpha.4 → v0.27.3
- k8s.io/kube-scheduler: v0.24.0-alpha.4 → v0.27.3
- k8s.io/kubectl: v0.24.0-alpha.4 → v0.27.3
- k8s.io/kubelet: v0.24.0-alpha.4 → v0.27.3
- k8s.io/kubernetes: v1.24.0-alpha.4 → v1.27.3
- k8s.io/legacy-cloud-providers: v0.24.0-alpha.4 → v0.27.3
- k8s.io/metrics: v0.24.0-alpha.4 → v0.27.3
- k8s.io/mount-utils: v0.24.0-alpha.4 → v0.27.3
- k8s.io/pod-security-admission: v0.24.0-alpha.4 → v0.27.3
- k8s.io/sample-apiserver: v0.24.0-alpha.4 → v0.27.3
- k8s.io/system-validators: v1.6.0 → v1.8.0
- k8s.io/utils: 3a6ce19 → a36077c
- sigs.k8s.io/apiserver-network-proxy/konnectivity-client: v0.0.30 → v0.1.2
- sigs.k8s.io/json: 9f7c6b3 → bc3834c
- sigs.k8s.io/kustomize/api: v0.10.1 → v0.13.2
- sigs.k8s.io/kustomize/kyaml: v0.13.0 → v0.14.1
- sigs.k8s.io/structured-merge-diff/v4: v4.2.1 → v4.2.3
- sigs.k8s.io/yaml: v1.2.0 → v1.3.0

### Removed
- bazil.org/fuse: 371fbbd
- cloud.google.com/go/firestore: v1.1.0
- github.com/PuerkitoBio/purell: [v1.1.1](https://github.com/PuerkitoBio/purell/tree/v1.1.1)
- github.com/PuerkitoBio/urlesc: [de5bf2a](https://github.com/PuerkitoBio/urlesc/tree/de5bf2a)
- github.com/ajstarks/svgo: [644b8db](https://github.com/ajstarks/svgo/tree/644b8db)
- github.com/armon/consul-api: [eb2c6b5](https://github.com/armon/consul-api/tree/eb2c6b5)
- github.com/armon/go-metrics: [f0300d1](https://github.com/armon/go-metrics/tree/f0300d1)
- github.com/armon/go-radix: [7fddfc3](https://github.com/armon/go-radix/tree/7fddfc3)
- github.com/auth0/go-jwt-middleware: [v1.0.1](https://github.com/auth0/go-jwt-middleware/tree/v1.0.1)
- github.com/aws/aws-sdk-go: [v1.38.49](https://github.com/aws/aws-sdk-go/tree/v1.38.49)
- github.com/bgentry/speakeasy: [v0.1.0](https://github.com/bgentry/speakeasy/tree/v0.1.0)
- github.com/bits-and-blooms/bitset: [v1.2.0](https://github.com/bits-and-blooms/bitset/tree/v1.2.0)
- github.com/bketelsen/crypt: [5cbc8cc](https://github.com/bketelsen/crypt/tree/5cbc8cc)
- github.com/blang/semver: [v3.5.1+incompatible](https://github.com/blang/semver/tree/v3.5.1)
- github.com/boltdb/bolt: [v1.3.1](https://github.com/boltdb/bolt/tree/v1.3.1)
- github.com/certifi/gocertifi: [2c3bb06](https://github.com/certifi/gocertifi/tree/2c3bb06)
- github.com/clusterhq/flocker-go: [2b8b725](https://github.com/clusterhq/flocker-go/tree/2b8b725)
- github.com/cockroachdb/datadriven: [bf6692d](https://github.com/cockroachdb/datadriven/tree/bf6692d)
- github.com/cockroachdb/errors: [v1.2.4](https://github.com/cockroachdb/errors/tree/v1.2.4)
- github.com/cockroachdb/logtags: [eb05cc2](https://github.com/cockroachdb/logtags/tree/eb05cc2)
- github.com/containerd/containerd: [v1.4.11](https://github.com/containerd/containerd/tree/v1.4.11)
- github.com/containerd/continuity: [v0.1.0](https://github.com/containerd/continuity/tree/v0.1.0)
- github.com/containerd/fifo: [v1.0.0](https://github.com/containerd/fifo/tree/v1.0.0)
- github.com/containerd/go-runc: [v1.0.0](https://github.com/containerd/go-runc/tree/v1.0.0)
- github.com/containerd/typeurl: [v1.0.2](https://github.com/containerd/typeurl/tree/v1.0.2)
- github.com/coreos/bbolt: [v1.3.2](https://github.com/coreos/bbolt/tree/v1.3.2)
- github.com/coreos/etcd: [v3.3.13+incompatible](https://github.com/coreos/etcd/tree/v3.3.13)
- github.com/coreos/go-systemd: [95778df](https://github.com/coreos/go-systemd/tree/95778df)
- github.com/coreos/pkg: [399ea9e](https://github.com/coreos/pkg/tree/399ea9e)
- github.com/dgrijalva/jwt-go: [v3.2.0+incompatible](https://github.com/dgrijalva/jwt-go/tree/v3.2.0)
- github.com/dgryski/go-sip13: [e10d5fe](https://github.com/dgryski/go-sip13/tree/e10d5fe)
- github.com/docker/docker: [v20.10.7+incompatible](https://github.com/docker/docker/tree/v20.10.7)
- github.com/docker/go-connections: [v0.4.0](https://github.com/docker/go-connections/tree/v0.4.0)
- github.com/elazarl/goproxy: [947c36d](https://github.com/elazarl/goproxy/tree/947c36d)
- github.com/emicklei/go-restful: [v2.9.5+incompatible](https://github.com/emicklei/go-restful/tree/v2.9.5)
- github.com/fatih/color: [v1.7.0](https://github.com/fatih/color/tree/v1.7.0)
- github.com/flynn/go-shlex: [3f9db97](https://github.com/flynn/go-shlex/tree/3f9db97)
- github.com/fogleman/gg: [0403632](https://github.com/fogleman/gg/tree/0403632)
- github.com/form3tech-oss/jwt-go: [v3.2.3+incompatible](https://github.com/form3tech-oss/jwt-go/tree/v3.2.3)
- github.com/frankban/quicktest: [v1.11.3](https://github.com/frankban/quicktest/tree/v1.11.3)
- github.com/getkin/kin-openapi: [v0.76.0](https://github.com/getkin/kin-openapi/tree/v0.76.0)
- github.com/getsentry/raven-go: [v0.2.0](https://github.com/getsentry/raven-go/tree/v0.2.0)
- github.com/go-ozzo/ozzo-validation: [v3.5.0+incompatible](https://github.com/go-ozzo/ozzo-validation/tree/v3.5.0)
- github.com/golang/freetype: [e2365df](https://github.com/golang/freetype/tree/e2365df)
- github.com/golangplus/testing: [af21d9c](https://github.com/golangplus/testing/tree/af21d9c)
- github.com/google/cel-spec: [v0.6.0](https://github.com/google/cel-spec/tree/v0.6.0)
- github.com/googleapis/gnostic: [v0.5.1](https://github.com/googleapis/gnostic/tree/v0.5.1)
- github.com/gophercloud/gophercloud: [v0.1.0](https://github.com/gophercloud/gophercloud/tree/v0.1.0)
- github.com/gopherjs/gopherjs: [fce0ec3](https://github.com/gopherjs/gopherjs/tree/fce0ec3)
- github.com/gorilla/mux: [v1.8.0](https://github.com/gorilla/mux/tree/v1.8.0)
- github.com/hashicorp/consul/api: [v1.1.0](https://github.com/hashicorp/consul/api/tree/v1.1.0)
- github.com/hashicorp/consul/sdk: [v0.1.1](https://github.com/hashicorp/consul/sdk/tree/v0.1.1)
- github.com/hashicorp/errwrap: [v1.0.0](https://github.com/hashicorp/errwrap/tree/v1.0.0)
- github.com/hashicorp/go-cleanhttp: [v0.5.1](https://github.com/hashicorp/go-cleanhttp/tree/v0.5.1)
- github.com/hashicorp/go-immutable-radix: [v1.0.0](https://github.com/hashicorp/go-immutable-radix/tree/v1.0.0)
- github.com/hashicorp/go-msgpack: [v0.5.3](https://github.com/hashicorp/go-msgpack/tree/v0.5.3)
- github.com/hashicorp/go-multierror: [v1.0.0](https://github.com/hashicorp/go-multierror/tree/v1.0.0)
- github.com/hashicorp/go-rootcerts: [v1.0.0](https://github.com/hashicorp/go-rootcerts/tree/v1.0.0)
- github.com/hashicorp/go-sockaddr: [v1.0.0](https://github.com/hashicorp/go-sockaddr/tree/v1.0.0)
- github.com/hashicorp/go-syslog: [v1.0.0](https://github.com/hashicorp/go-syslog/tree/v1.0.0)
- github.com/hashicorp/go-uuid: [v1.0.1](https://github.com/hashicorp/go-uuid/tree/v1.0.1)
- github.com/hashicorp/go.net: [v0.0.1](https://github.com/hashicorp/go.net/tree/v0.0.1)
- github.com/hashicorp/hcl: [v1.0.0](https://github.com/hashicorp/hcl/tree/v1.0.0)
- github.com/hashicorp/logutils: [v1.0.0](https://github.com/hashicorp/logutils/tree/v1.0.0)
- github.com/hashicorp/mdns: [v1.0.0](https://github.com/hashicorp/mdns/tree/v1.0.0)
- github.com/hashicorp/memberlist: [v0.1.3](https://github.com/hashicorp/memberlist/tree/v0.1.3)
- github.com/hashicorp/serf: [v0.8.2](https://github.com/hashicorp/serf/tree/v0.8.2)
- github.com/heketi/heketi: [v10.3.0+incompatible](https://github.com/heketi/heketi/tree/v10.3.0)
- github.com/heketi/tests: [f3775cb](https://github.com/heketi/tests/tree/f3775cb)
- github.com/jmespath/go-jmespath/internal/testify: [v1.5.1](https://github.com/jmespath/go-jmespath/internal/testify/tree/v1.5.1)
- github.com/jmespath/go-jmespath: [v0.4.0](https://github.com/jmespath/go-jmespath/tree/v0.4.0)
- github.com/jtolds/gls: [v4.20.0+incompatible](https://github.com/jtolds/gls/tree/v4.20.0)
- github.com/jung-kurt/gofpdf: [24315ac](https://github.com/jung-kurt/gofpdf/tree/24315ac)
- github.com/kr/fs: [v0.1.0](https://github.com/kr/fs/tree/v0.1.0)
- github.com/lpabon/godbc: [v0.1.1](https://github.com/lpabon/godbc/tree/v0.1.1)
- github.com/magiconair/properties: [v1.8.1](https://github.com/magiconair/properties/tree/v1.8.1)
- github.com/mattn/go-colorable: [v0.0.9](https://github.com/mattn/go-colorable/tree/v0.0.9)
- github.com/mattn/go-isatty: [v0.0.3](https://github.com/mattn/go-isatty/tree/v0.0.3)
- github.com/mattn/go-runewidth: [v0.0.7](https://github.com/mattn/go-runewidth/tree/v0.0.7)
- github.com/mindprince/gonvml: [9ebdce4](https://github.com/mindprince/gonvml/tree/9ebdce4)
- github.com/mitchellh/cli: [v1.0.0](https://github.com/mitchellh/cli/tree/v1.0.0)
- github.com/mitchellh/go-homedir: [v1.1.0](https://github.com/mitchellh/go-homedir/tree/v1.1.0)
- github.com/mitchellh/go-testing-interface: [v1.0.0](https://github.com/mitchellh/go-testing-interface/tree/v1.0.0)
- github.com/mitchellh/gox: [v0.4.0](https://github.com/mitchellh/gox/tree/v0.4.0)
- github.com/mitchellh/iochan: [v1.0.0](https://github.com/mitchellh/iochan/tree/v1.0.0)
- github.com/morikuni/aec: [v1.0.0](https://github.com/morikuni/aec/tree/v1.0.0)
- github.com/mvdan/xurls: [v1.1.0](https://github.com/mvdan/xurls/tree/v1.1.0)
- github.com/niemeyer/pretty: [a10e7ca](https://github.com/niemeyer/pretty/tree/a10e7ca)
- github.com/oklog/ulid: [v1.3.1](https://github.com/oklog/ulid/tree/v1.3.1)
- github.com/olekukonko/tablewriter: [v0.0.4](https://github.com/olekukonko/tablewriter/tree/v0.0.4)
- github.com/opencontainers/image-spec: [v1.0.1](https://github.com/opencontainers/image-spec/tree/v1.0.1)
- github.com/opentracing/opentracing-go: [v1.1.0](https://github.com/opentracing/opentracing-go/tree/v1.1.0)
- github.com/pascaldekloe/goe: [57f6aae](https://github.com/pascaldekloe/goe/tree/57f6aae)
- github.com/pelletier/go-toml: [v1.2.0](https://github.com/pelletier/go-toml/tree/v1.2.0)
- github.com/pkg/sftp: [v1.10.1](https://github.com/pkg/sftp/tree/v1.10.1)
- github.com/posener/complete: [v1.1.1](https://github.com/posener/complete/tree/v1.1.1)
- github.com/prometheus/tsdb: [v0.7.1](https://github.com/prometheus/tsdb/tree/v0.7.1)
- github.com/quobyte/api: [v0.1.8](https://github.com/quobyte/api/tree/v0.1.8)
- github.com/remyoudompheng/bigfft: [52369c6](https://github.com/remyoudompheng/bigfft/tree/52369c6)
- github.com/russross/blackfriday: [v1.5.2](https://github.com/russross/blackfriday/tree/v1.5.2)
- github.com/ryanuber/columnize: [9b3edd6](https://github.com/ryanuber/columnize/tree/9b3edd6)
- github.com/sean-/seed: [e2103e2](https://github.com/sean-/seed/tree/e2103e2)
- github.com/sergi/go-diff: [v1.1.0](https://github.com/sergi/go-diff/tree/v1.1.0)
- github.com/shurcooL/sanitized_anchor_name: [v1.0.0](https://github.com/shurcooL/sanitized_anchor_name/tree/v1.0.0)
- github.com/smartystreets/assertions: [v1.1.0](https://github.com/smartystreets/assertions/tree/v1.1.0)
- github.com/smartystreets/goconvey: [v1.6.4](https://github.com/smartystreets/goconvey/tree/v1.6.4)
- github.com/spf13/cast: [v1.3.0](https://github.com/spf13/cast/tree/v1.3.0)
- github.com/spf13/jwalterweatherman: [v1.0.0](https://github.com/spf13/jwalterweatherman/tree/v1.0.0)
- github.com/spf13/viper: [v1.7.0](https://github.com/spf13/viper/tree/v1.7.0)
- github.com/storageos/go-api: [v2.2.0+incompatible](https://github.com/storageos/go-api/tree/v2.2.0)
- github.com/subosito/gotenv: [v1.2.0](https://github.com/subosito/gotenv/tree/v1.2.0)
- github.com/ugorji/go: [v1.1.4](https://github.com/ugorji/go/tree/v1.1.4)
- github.com/urfave/cli: [v1.22.2](https://github.com/urfave/cli/tree/v1.22.2)
- github.com/urfave/negroni: [v1.0.0](https://github.com/urfave/negroni/tree/v1.0.0)
- github.com/xordataexchange/crypt: [b2862e3](https://github.com/xordataexchange/crypt/tree/b2862e3)
- go.opentelemetry.io/contrib: v0.20.0
- go.opentelemetry.io/otel/exporters/otlp: v0.20.0
- go.opentelemetry.io/otel/oteltest: v0.20.0
- go.opentelemetry.io/otel/sdk/export/metric: v0.20.0
- go.opentelemetry.io/otel/sdk/metric: v0.20.0
- gonum.org/v1/gonum: v0.6.2
- gonum.org/v1/netlib: 7672324
- gonum.org/v1/plot: e2840ee
- gopkg.in/ini.v1: v1.51.0
- gopkg.in/resty.v1: v1.12.0
- gotest.tools/v3: v3.0.3
- modernc.org/cc: v1.0.0
- modernc.org/golex: v1.0.0
- modernc.org/mathutil: v1.0.0
- modernc.org/strutil: v1.0.0
- modernc.org/xc: v1.0.0
- rsc.io/pdf: v0.1.1
- sigs.k8s.io/kustomize/cmd/config: v0.10.2
- sigs.k8s.io/kustomize/kustomize/v4: v4.4.1

# [v2.5.0](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/releases/tag/v2.5.0)

Image updates:

* change log level to V5 when discovering path not match pattern by @hellogdc in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/235
* Update Klog to V2 by @Kartik494 in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/234
* Duplicate volume guard by @davidmccormick in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/246
* add local volume discovery period flag by @dabaooline in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/261
* Multi linux arch and multi windows distro builds by @mauriciopoppe in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/273
* Update discovery and deletion code to work in Windows nodes through CSI Proxy by @mauriciopoppe in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/276
* feat: add lstc2022 windows image build by @andyzhangx in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/314
* fix: golang.org/x/crypto for CVE-2022-27191 by @andyzhangx in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/321
* feat: support namePattern as a list by @andyzhangx in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/320

Deployment updates:

* Helm chart init container support by @alice-sawatzky in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/251
* Add Windows daemonset to the helm template by @mauriciopoppe in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/275
* Release chart as a package and publish it on gh-pages branch by @skylenet in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/280
* ClusterRole system:persistent-volume-provisioner replaced with a custom ClusterRole with the same contents minus permissions to access PVCs by @mauriciopoppe in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/292
* Update PV to beta yaml file  by @Kartik494 in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/326

Doc updates:

* Add LKE Option to Bring up Local Disks by @rsyracuse in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/230
* Add 0 dependency EKS example by @arianvp in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/252
* Fix gce bootstrap script guide by @theidexisted in https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/290

**Full Changelog**: https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/compare/v2.4.0...v2.5.0

# [v2.4.0](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/releases/tag/v2.4.0)

Image updates:

- add `namePattern` field to filter volumes
  ([#187](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/187))

- blkdiscard.sh no longer zeros disks. This script was passing the -z option to
  blkdiscard which meant it was not performing discards. This has been fixed.
  If you desire zeroing, rather than block discarding, please switch to
  dd_zero.sh.
  ([#200](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/200))

- handle DeletedFinalStateUnknown object when receiving PV delete event
  ([222](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/222))

- We start to push multi-arch images to Kubernetes main image-serving system,
  our repository is hosted at k8s.gcr.io/sig-storage/local-volume-provisioner.
  Our legacy repository quay.io/external_storage/local-volume-provisioner is
  deprecated but still maintained. Note that only amd64 images will be pushed
  to this repository.
  ([206](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/206))

Helm updates:

- **Action required**: As the helm-chart structure changed the already running
  pod will be recreated during upgrade. Documentation can be found under
  [helm/README.md](./helm/README.md). Compare your existing values with the new
  chart parameter before upgrade.
  ([#179](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/179))

- Added daemonset.podAnnotations and daemonset.podLabels to Helm chart values.
  ([#213](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/213))

- Add opt-out for `/dev` volume in the chart
  ([#219](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/219))

- Accept `labelsForPV` elements in the chart
  ([220](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/220))

- Allow unprivileged provisioner in chart
  ([221](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/221))

# [v2.3.4](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/releases/tag/v2.3.4)

Image updates:

- A readiness check is added to expose discovery state
  Refer to [docs](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/blob/v2.3.4/docs/provisioner.md#readiness) for more information.
  ([#135](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/135))
- A new metric `local_volume_provisioner_persistentvolume_capacity_bytes` is
  added to report the capacity in bytes of the local volumes discovered
  ([#160](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/160))
- Fix an issue that may cause released PVs not to be recycled
  ([#174](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/174))

# [v2.3.3](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/releases/tag/v2.3.3)

Image updates:
- Allow user to configure additional PV labels
  ([#118](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/118))
- Add an option to create PVs owned by their respective Nodes
  ([#123](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/123))

Deployment updates:
- Fix invalid pod security policy in helm chart
  ([#93](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/93))
- Able to set storage class default in Kubernetes
  ([#125](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/125))

# [v2.3.2](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/releases/tag/v2.3.2)

Image updates:
- Fix an issue in block devices cleanup by Kubernetes Job
  ([#60](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/60))

Deployment updates:
- Support pod security policy
  ([#73](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/73))
- Support pod priority class
  ([#53](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/53))
- Minor bugs fixed

# [v2.3.1](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/releases/tag/v2.3.1)

Abandoned and not released.

# [v2.3.0](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/releases/tag/v2.3.0)

Image updates:
* Support mount options from StorageClass
  ([#1005](https://github.com/kubernetes-incubator/external-storage/pull/1005))
* Support fs volumes on block
  ([#980](https://github.com/kubernetes-incubator/external-storage/pull/980)).
  **Breaking change:** The change breaks backwards compatibility for block volumes: Users must explicitly set volumeMode to "Block" in config if a StorageClass is expected to be used for block volumes.
* Update base image to k8s.gcr.io/debian-base-amd64:0.4.0
  ([#1040](https://github.com/kubernetes-incubator/external-storage/pull/1040))

Deployment updates:
* Add option for nodeSelector in DaemonSet template
  ([#1022](https://github.com/kubernetes-incubator/external-storage/pull/1022))
* Add option to create namespace and use apps/v1 DaemonSet
  ([#1039](https://github.com/kubernetes-incubator/external-storage/pull/1039))
* Fixes provisioner jobs role name in helm template
  ([#1073](https://github.com/kubernetes-incubator/external-storage/pull/1073))

# [v2.2.0](https://github.com/kubernetes-incubator/external-storage/releases/tag/local-volume-provisioner-v2.2.0)
Image updates:
* Add Prometheus metrics
  ([#721](https://github.com/kubernetes-incubator/external-storage/pull/721))
* Support Retain reclaim policy
  ([#776](https://github.com/kubernetes-incubator/external-storage/pull/776))
* Add option for resync period and add a default of 5 minutes
  ([#800](https://github.com/kubernetes-incubator/external-storage/pull/800))
* Add option for cleaning filesystem PVs in a Job
  ([#863](https://github.com/kubernetes-incubator/external-storage/pull/863))
* Add option for using only Node.Name as the provisioner name, instead of Node.UID ([#947](https://github.com/kubernetes-incubator/external-storage/pull/947))

Deployment updates:
* Refactor helm generation
  ([#789](https://github.com/kubernetes-incubator/external-storage/pull/789))
* Add option for tolerations
  ([#792](https://github.com/kubernetes-incubator/external-storage/pull/792))
* Add /dev volume mount for raw block support
  ([#799)](https://github.com/kubernetes-incubator/external-storage/pull/799)
* Add option for resource requests and limits
  ([#831](https://github.com/kubernetes-incubator/external-storage/pull/831))

# [v2.1.0](https://github.com/kubernetes-incubator/external-storage/releases/tag/local-volume-provisioner-v2.1.0)
The following changes require Kubernetes 1.10 or higher.
* Add block volumeMode discovery and cleanup.
* **Important:** Beta PV.NodeAffinity field is used by default. If running against an older K8s version,
  the `useAlphaAPI` flag must be set in the configMap.

# [v2.0.0](https://github.com/kubernetes-incubator/external-storage/releases/tag/local-volume-provisioner-v2.0.0)
**Important:** This version is incompatible and has breaking changes with v1!
* Remove default config, a configmap is now required.
* Configmap data is changed from json to yaml syntax.
* All local volumes must be mount points.  For directory-based volumes, a
  bind-mount must be done in order for the provisioner to discover them. This
  requires the K8s [mount propagation feature](https://kubernetes.io/docs/concepts/storage/volumes/#mount-propagation)
  to be enabled.
* Detected capacity is rounded down to the nearest GB.
* New option to specify which node labels to add to the PV.

# [v1.0.1](https://github.com/kubernetes-incubator/external-storage/releases/tag/local-volume-provisioner-bootstrap-v1.0.1)
* Change fs capacity detection to use K8s volume util method.
* Add event on PV if cleanup or deletion fails.

# [v1.0.0](https://github.com/kubernetes-incubator/external-storage/releases/tag/local-volume-provisioner-bootstrap-v1.0.0)
* Run a provisioner on each node via DaemonSet.
* Discovers file-based volumes under configurable discovery directories and creates a local PV for each.
* When PV created by the provisioner is released, delete file contents and delete PV, to be discovered again.
* Use PV informer to populate volume cache.
