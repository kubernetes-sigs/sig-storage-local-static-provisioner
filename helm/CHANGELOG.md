## v2.0.0

### Breaking changes

- Helm Chart values flattened when possible (#393, @mauriciopoppe)
  - Flattened top level `common` dictionary, all the keys are now at the top level.
  - Flattened top level `daemonset` dictionary, all the keys are now at the top level.
- `rbac.pspEnabled` removed
- `useAlphaAPI` removed

### Features

- Add enableWindows helm chart value to control the deployment of Windows manifests (#388, @jennwah)
- Helm chart v1.0.0 uses registry.k8s.io/sig-storage/local-volume-provisioner:v2.5.0
  Add field .Values.daemonset.nodeSelectorWindows to the helm chart. (#353, @mauriciopoppe)
