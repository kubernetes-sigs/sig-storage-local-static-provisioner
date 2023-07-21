## v2.0.0

**Breaking changes:**

- Flattened some keys of the helm chart
  - Flattened top level `common` dictionary, all the keys are now at the top level.
  - Flattened top level `daemonset` dictionary, all the keys are now at the top level.
- `rbac.pspEnabled` removed
- `useAlphaAPI` removed
