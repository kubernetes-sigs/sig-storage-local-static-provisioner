# Release notes for v2.10.0

[Documentation](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/tree/master/docs)

## Changes by Kind

### Feature

- feat(helm): expose additional args through Helm values for the provisioner DaemonSet ([#631](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/631), [@miltalex](https://github.com/miltalex))
- feat: support node-local CSI volumes in node-cleanup via `--csi-drivers` ([#632](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/632), [@karantyagi-reltio](https://github.com/karantyagi-reltio))

### Documentation

- docs: update repo version references to 2.9.0 after the v2.9.0 cut ([#603](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/603), [@andyzhangx](https://github.com/andyzhangx))

### Bug or Regression

- fix: remove the startup not-ready taint only after successful discovery completes ([#630](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/630), [@andyzhangx](https://github.com/andyzhangx))
- fix(release): correct the `release-<major>.<minor>` branch regex in `hack/release.sh` so release-branch postsubmits detect canary versions correctly ([#599](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/599), [@andyzhangx](https://github.com/andyzhangx))
- fix: CVE-2026-56854 ([#617](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/617), [@andyzhangx](https://github.com/andyzhangx))
- fix: CVE-2026-56855 ([#620](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/620), [@andyzhangx](https://github.com/andyzhangx))
- fix: CVE-2026-84304 ([#619](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/619), [@andyzhangx](https://github.com/andyzhangx))

### Other (Cleanup or Flake)

- ci: add trivy scanning for the `local-volume-node-cleanup` image ([#605](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/605), [@andyzhangx](https://github.com/andyzhangx))
- chore: bump Go to 1.26.6 ([#614](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/614), [@andyzhangx](https://github.com/andyzhangx))
- chore: bump `gcb-docker-gcloud` to `v20260729-0b834bafc6` ([#613](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/613), [@andyzhangx](https://github.com/andyzhangx))
- helm: bump chart to 2.9.0 ([#602](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/602), [@andyzhangx](https://github.com/andyzhangx))

### GitHub Actions / Dependencies

- bump github/codeql-action init+analyze from 4.37.3 to 4.37.4 ([#604](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/604), [@andyzhangx](https://github.com/andyzhangx))
- bump github/codeql-action init+analyze from 4.37.4 to 4.37.5 ([#606](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/606), [@andyzhangx](https://github.com/andyzhangx))
- chore(deps): bump github/codeql-action/init from `85689a1` to `d1ba80a` ([#607](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/607), [@dependabot](https://github.com/apps/dependabot))
- chore(deps): bump github/codeql-action/analyze from `85689a1` to `d1ba80a` ([#608](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/608), [@dependabot](https://github.com/apps/dependabot))
- bump github/codeql-action init+analyze from 4.37.5 to 4.37.6 ([#612](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/612), [@andyzhangx](https://github.com/andyzhangx))
- bump github/codeql-action init+analyze from 4.37.6 to 4.37.7 ([#615](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/615), [@andyzhangx](https://github.com/andyzhangx))
- bump github/codeql-action init+analyze from 4.37.7 to 4.37.8 ([#616](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/616), [@andyzhangx](https://github.com/andyzhangx))
- bump github/codeql-action init+analyze from 4.37.8 to 4.37.9 ([#618](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/618), [@andyzhangx](https://github.com/andyzhangx))
- chore(deps): bump helm/kind-action from 1.14.0 to 1.15.0 ([#621](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/621), [@dependabot](https://github.com/apps/dependabot))
- bump github/codeql-action init+analyze from 4.37.9 to 4.38.0 ([#622](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/622), [@andyzhangx](https://github.com/andyzhangx))
- bump github/codeql-action init+analyze from 4.38.0 to 4.38.1 ([#626](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/626), [@andyzhangx](https://github.com/andyzhangx))
- bump github/codeql-action init+analyze from 4.38.1 to 4.38.2 ([#627](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/627), [@andyzhangx](https://github.com/andyzhangx))

## Dependencies

Notable Go module upgrades since v2.9.0:

- Go: 1.25 → 1.26
- golang.org/x/sys: v0.46.0 → v0.47.0
- cel.dev/expr: v0.25.1 → v0.25.2
- go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp: v0.53.0 → v0.61.0
- go.opentelemetry.io/otel: v1.43.0 → v1.44.0
- go.opentelemetry.io/otel/{metric,sdk,trace}: v1.43.0 → v1.44.0
- golang.org/x/crypto: v0.53.0 → v0.56.0
- golang.org/x/net: v0.56.0 → v0.58.0
- golang.org/x/oauth2: v0.34.0 → v0.36.0
- golang.org/x/term: v0.44.0 → v0.45.0
- golang.org/x/text: v0.40.0 → v0.41.0
- golang.org/x/tools: v0.47.0 → v0.48.0
- gonum.org/v1/gonum: v0.16.0 → v0.17.0
- google.golang.org/genproto/googleapis/api: `ff82c1b0f217` → `3dc84a4a5aaa`
- google.golang.org/genproto/googleapis/rpc: `0b37fe3546d5` → `18b4a7587f8a`
- google.golang.org/grpc: v1.79.3 → v1.83.2
- google.golang.org/protobuf: v1.36.11 → v1.36.12
