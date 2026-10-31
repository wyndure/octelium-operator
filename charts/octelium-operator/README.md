# Octelium operator

This chart installs the Octelium operator and its namespaced custom resource definitions. The controller watches resources across namespaces and uses each resource's `octeliumRef` to select its resource-server endpoint and credentials. This operator uses Octelium v0.47.0 APIs.

## Install

Build and publish an operator image from this repository, then install with its repository and tag:

```sh
helm upgrade --install octelium-operator ./charts/octelium-operator \
  --namespace octelium-operator --create-namespace \
  --set image.repository=ghcr.io/wyndure/octelium-operator \
  --set image.tag=latest
```

The default repository is `ghcr.io/wyndure/octelium-operator`. The default image tag is the operator's `Chart.appVersion`, independently of the upstream Octelium version. The chart does not publish the image. Set `image.digest` to use an immutable digest instead of a tag.

SPIFFE is enabled by default, matching the operator's Kustomize deployment. Install SPIRE and the `csi.spiffe.io` CSI driver, and configure workload registration for the operator namespace, ServiceAccount and Pod labels. The Workload API socket is mounted at `/run/spire/sockets/spire-agent.sock`. When your SPIRE registration selects namespaces using `octelium.wyndure.com/spiffe=enabled`, label the namespace:

```sh
kubectl label namespace octelium-operator octelium.wyndure.com/spiffe=enabled --overwrite
```

For installations where all connections explicitly use `credentials.type: Insecure` (plaintext, unauthenticated gRPC), pass `--set spiffe.enabled=false`. This matches a resource server running without SPIFFE. Disabling the CSI mount does not change a connection's credential type or trigger automatic plaintext fallback. Connection credentials and resource-server endpoints are configured in `Octelium` CRs, rather than Helm values. See the repository README for connection examples and namespace grants.

Only one operator installation should manage a cluster's CRs. Leader election coordinates replicas of that installation in its release namespace; separate installations in different namespaces would both reconcile the same resources.

The GitHub Actions workflow publishes this chart to `oci://ghcr.io/wyndure/charts/octelium-operator` after the checks pass. Use `--version` to select a release or development prerelease. Container images are published separately at `ghcr.io/wyndure/octelium-operator`.

## Configuration

| Value | Default | Purpose |
| --- | --- | --- |
| `replicaCount` | `1` | Manager replicas; multiple replicas require leader election. |
| `image.repository`, `image.tag`, `image.digest` | See above | Operator container image. |
| `imagePullSecrets` | `[]` | Kubernetes image pull Secret references. |
| `serviceAccount.create`, `serviceAccount.name` | `true`, generated | Create an account or use a named existing account. |
| `serviceAccount.annotations` | `{}` | ServiceAccount annotations. |
| `rbac.create` | `true` | Install controller and leader-election permissions. |
| `leaderElection.enabled` | `true` | Elect an active controller using a namespace-local Lease. |
| `syncInterval` | `30s` | Interval for detecting changes in Octelium. |
| `spiffe.enabled`, `spiffe.csiDriver` | `true`, `csi.spiffe.io` | Mount the SPIFFE Workload API socket through CSI. |
| `metrics.enabled`, `metrics.port` | `false`, `8080` | Expose the manager's HTTP metrics through a ClusterIP Service. |
| `podSecurityContext`, `securityContext` | Non-root UID/GID 65532; read-only root filesystem | Pod and manager container security settings. |
| `resources` | 10m CPU / 64Mi memory requested; 256Mi memory limit | Manager resources. |
| `livenessProbe`, `readinessProbe` | HTTP `/healthz`, `/readyz` on port 8081 | Health probes; `{}` disables a probe. |
| `podLabels`, `podAnnotations` | `{}` | Additional Pod metadata; required selector and Octelium component labels take precedence. |
| `extraArgs`, `extraEnv`, `extraEnvFrom` | `[]` | Additional manager arguments and Kubernetes environment configuration. |
| `extraInitContainers`, `extraVolumes`, `extraVolumeMounts` | `[]` | Additional Kubernetes init containers, Pod volumes and manager mounts. |
| `nodeSelector`, `tolerations`, `affinity`, `priorityClassName` | Empty | Scheduling settings. |
| `nameOverride`, `fullnameOverride` | Empty | Override resource naming. |
| `terminationGracePeriodSeconds` | `10` | Pod termination grace period. |

The controller ClusterRole comes from the generated `config/rbac/role.yaml`, including ConfigMap publication permissions for `metadataConfigMap`. Managed resources remain namespaced; the controller needs cluster-wide reads and reconciliation permissions to manage resources in multiple namespaces. Leader election adds only Lease access and Event creation/patching in the release namespace. If `rbac.create=false`, provision equivalent permissions yourself. If `serviceAccount.create=false`, supply `serviceAccount.name`.

Metrics use the operator's existing unauthenticated HTTP endpoint. Enable them only where your cluster's network access policy permits the intended scrapers.

## CRD lifecycle

Helm installs the twelve CRDs from `crds/` on first installation. Helm does not upgrade or delete these CRDs automatically. Before upgrading the chart, apply the CRDs from the new chart version:

```sh
kubectl apply --server-side -f charts/octelium-operator/crds/
helm upgrade octelium-operator ./charts/octelium-operator \
  --namespace octelium-operator --reuse-values
```

Use `--skip-crds` when a separate administrator manages CRDs. Uninstalling the release leaves the CRDs and their instances in place. Retire managed resources and allow finalizers to finish while the operator is running before decommissioning it.

For chart development, `make helm-sync` regenerates CRDs/RBAC and refreshes the chart copies. `make helm-lint` validates the chart and `make helm-package` writes an archive into `dist/`. Edit API types and RBAC markers to change generated content.

## License

See [LICENSE](LICENSE) for GNU AGPL v3 and [LICENSE-APACHE](LICENSE-APACHE) for retained Apache-2.0 notices.
