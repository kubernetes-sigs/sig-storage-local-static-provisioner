## v2.0.0

### Breaking changes

- Helm Chart values flattened when possible ([#393](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/393), [@mauriciopoppe](https://github.com/mauriciopoppe))
  - Flattened top level `common` dictionary, all the keys are now at the top level.
  - Flattened top level `daemonset` dictionary, all the keys are now at the top level.
- `rbac.pspEnabled` removed
- `useAlphaAPI` removed

### Features

- Add enableWindows helm chart value to control the deployment of Windows manifests ([#388](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/388), [@jennwah](https://github.com/jennwah))
- Add support for additional volumes ([#401](https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/pull/401), [@stevehipwell](https://github.com/stevehipwell))
