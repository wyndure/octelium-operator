# Octelium operator

The Octelium operator manages Octelium resources through Kubernetes custom resources. It connects to a resource server using SPIFFE mTLS or explicitly unauthenticated plaintext gRPC and reconciles desired configuration through the upstream resource-management code used by `octeliumctl`.

This version uses Octelium v0.47.0. Its Go modules are pinned to that release commit in `go.mod`.

## Install

Build and publish the operator image, then install the Helm chart:

```sh
docker build -t ghcr.io/wyndure/octelium-operator:latest .
docker push ghcr.io/wyndure/octelium-operator:latest

helm upgrade --install octelium-operator ./charts/octelium-operator \
  --namespace octelium-operator --create-namespace \
  --set image.repository=ghcr.io/wyndure/octelium-operator \
  --set image.tag=latest \
  --set spiffe.enabled=false
```

The example runs without SPIFFE using an explicitly configured plaintext connection. For SPIFFE authentication, enable `spiffe.enabled`, install the SPIFFE CSI driver and configure SPIRE workload registration for the operator. The [chart README](charts/octelium-operator/README.md) describes its settings, RBAC and CRD upgrade procedure.

The operator manages resources across Kubernetes namespaces. Run one installation per cluster; increase `replicaCount` with leader election enabled for availability.

A Kustomize installation is also available:

```sh
make build-installer IMG=ghcr.io/wyndure/octelium-operator:latest

kubectl apply --server-side -f dist/install.yaml
```

The Kustomize deployment mounts the SPIFFE CSI socket by default. Customize it with an overlay for your environment.

## Connect to Octelium

For a resource server running without SPIFFE, create the connection namespace:

```sh
kubectl create namespace octelium-system
```

Then use an explicit `Insecure` connection:

```yaml
apiVersion: octelium.wyndure.com/v1
kind: Octelium
metadata:
  name: default
  namespace: octelium-system
spec:
  endpoint: octelium-rscserver-backend.octelium.svc.cluster.local:8080
  credentials:
    type: Insecure
```

For rotating SPIFFE credentials, use:

```yaml
spec:
  endpoint: resource-server.example.com:8080
  credentials:
    type: SPIFFE
    trustDomain: example.org
    socketPath: /run/spire/sockets/spire-agent.sock
```

Inspect connection readiness with `kubectl get octeliums -n octelium-system`.

## Manage resources

The supported API is `core.octelium.wyndure.com/v1`, with kinds `OcteliumClusterConfig`, `OcteliumConfig`, `OcteliumPolicy`, `OcteliumIdentityProvider`, `OcteliumNamespace`, `OcteliumGroup`, `OcteliumUser`, `OcteliumService`, `OcteliumCredential`, `OcteliumSecret` and `OcteliumRegion`.

Each resource references an `Octelium` connection:

```yaml
apiVersion: core.octelium.wyndure.com/v1
kind: OcteliumNamespace
metadata:
  name: engineering
  namespace: octelium-system
  labels:
    team: engineering
octeliumRef:
  apiVersion: octelium.wyndure.com/v1
  kind: Octelium
  name: default
  namespace: octelium-system
metadataConfigMap: engineering-metadata
spec:
  attrs:
    environment: production
octelium:
  metadata:
    displayName: Engineering
```

`octeliumRef.apiVersion`, `kind` and `namespace` default to the connection API, `Octelium` and the resource's namespace. The `spec` uses the upstream protobuf specification directly. Kubernetes `metadata.name` selects the upstream resource; labels and annotations synchronize in both directions. A three-way merge preserves unrelated external metadata changes, with Kubernetes taking precedence when both sides change the same field. Octelium-specific metadata belongs under `octelium.metadata`.

The operator records readiness in `status.conditions`, the external identity in `status.externalUID` and runtime state in `status.upstreamStatus`. It periodically checks for external changes; connection and credential changes also trigger reconciliation. Upstream resource identities are shared within an Octelium instance, so use one CR to manage each target.

`OcteliumClusterConfig` updates the existing default ClusterConfig. Provision that initial configuration with the Octelium initializer before reconciling it. `octelium.initialStatus` supplies create-time runtime state for resources such as Region.

Deletion uses the observed external UID to select the upstream resource. Set `octelium.deletionPolicy: Retain` to preserve it when deleting the CR. ClusterConfig and system resources are retained. Allow finalizers to complete before removing the operator.

## Allow cross-namespace references

Connections accept references from their own namespace by default. Connection owners can grant additional namespaces and restrict resource kinds:

```yaml
spec:
  allowedResources:
    namespaces:
      from: Selector
      selector:
        matchLabels:
          octelium.wyndure.com/instance: production
    kinds:
      - group: core.octelium.wyndure.com
        kind: OcteliumNamespace
```

`namespaces.from` accepts `Same`, `Selector` or `All`. An omitted `kinds` list permits all supported kinds. These grants are checked on each reconciliation, including deletion. Existing resources remain bound to their connection and upstream identities. Grant Kubernetes access to connection CRs, managed CRs and their credentials to trusted administrators.

## Export metadata to a ConfigMap

The optional top-level `metadataConfigMap` names an output ConfigMap in the managed CR's namespace. After reconciliation, each upstream metadata field becomes a data key, including `name`, `uid`, `resourceVersion`, timestamps, labels and tags. Strings are stored directly; other values use JSON encoding.

The operator creates and owns the ConfigMap, restores changed metadata values, recreates deleted output and preserves unrelated data keys. Existing ConfigMaps must already be controlled by the same CR. Use a distinct name for each resource. Changing or removing the field stops updates to the previous ConfigMap, which remains owned until deleted or garbage-collected with its CR.

## Supply Secret data

Keep sensitive values in a native Kubernetes Secret in the managed CR's namespace:

```yaml
apiVersion: core.octelium.wyndure.com/v1
kind: OcteliumSecret
metadata:
  name: application-key
  namespace: octelium-system
octeliumRef:
  name: default
spec: {}
octelium:
  dataFromSecret:
    name: application-key-source
    key: value
```

`octelium.data` accepts upstream Config/Secret data payloads directly. Choose either `data` or `dataFromSecret` for Secret resources. Synchronization errors appear as a failed Ready condition; operator logs contain diagnostic details.

## Build and validate

Use Go 1.27.3 and GNU Make. The Makefile installs pinned generation and envtest tools into `.tools/`.

```sh
make build
make lint
make test
make test-race       # Requires a C compiler.
make helm-lint
make helm-package
```

Tests include an isolated Kubernetes API server and etcd through envtest. `make manifests generate` regenerates CRDs, RBAC and deep-copy code; `make helm-sync` refreshes the Helm chart copies. The CRD schemas derive from upstream protobuf descriptors. Runtime reconciliation delegates to upstream diff/apply and management APIs.

The manager accepts `--kubeconfig`, `--leader-elect`, `--sync-interval` (default `30s`), `--health-probe-bind-address` (default `:8081`), `--metrics-bind-address` (default `0`, disabled) and the controller-runtime Zap logging flags. Run `bin/manager --help` for details.

## License

The distributed operator uses GNU AGPL v3; see [LICENSE-AGPL-3.0](LICENSE-AGPL-3.0). Existing Apache-2.0 notices are retained, with the corresponding license in [LICENSE-APACHE](LICENSE-APACHE). Upstream dependencies retain their respective licenses.

## Legal

This project is a community endeavour and is not associated with Octelium Labs, LLC.

Octelium and Octelium logo are trademarks of Octelium Labs, LLC.

WireGuard is a registered trademark of Jason A. Donenfeld.
