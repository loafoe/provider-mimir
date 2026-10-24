# Provider Mimir

`provider-mimir` is a native [Crossplane](https://crossplane.io/) provider for
[Grafana Mimir](https://grafana.com/oss/mimir/). It manages Mimir ruler and
alertmanager resources through the Mimir HTTP API and exposes
XRM-conformant managed resources.

This provider is implemented entirely in Go against the Mimir API. It does
**not** use [Upjet](https://github.com/crossplane/upjet) or Terraform, and no
code generation from a Terraform provider is involved.

## Supported Resources

Both cluster-scoped and namespaced variants of every managed resource are
available. Cluster-scoped resources live under `*.crossplane.io` API groups,
while namespaced resources live under the corresponding `*.m.crossplane.io`
groups.

| Resource | Kind | API Group (cluster) | API Group (namespaced) |
| --- | --- | --- | --- |
| Alertmanager configuration | `Config` | `alertmanager.mimir.crossplane.io` | `alertmanager.mimir.m.crossplane.io` |
| Ruler alerting rule group | `RuleGroupAlerting` | `ruler.mimir.crossplane.io` | `ruler.mimir.m.crossplane.io` |
| Ruler recording rule group | `RuleGroupRecording` | `ruler.mimir.crossplane.io` | `ruler.mimir.m.crossplane.io` |
| Ruler recording rule group (legacy) | `GroupRecording` | `ruler.mimir.crossplane.io` | — |
| Provider configuration | `ProviderConfig` | `mimir.crossplane.io` | `mimir.m.crossplane.io` |
| Provider configuration usage | `ProviderConfigUsage` | `mimir.crossplane.io` | `mimir.m.crossplane.io` |

## Getting Started

Install the provider package:

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-mimir
spec:
  package: ghcr.io/loafoe/provider-mimir:latest
```

See [`examples/install.yaml`](examples/install.yaml) for a concrete example.

### ProviderConfig

The `ProviderConfig` points at your Mimir installation and configures
authentication. Credentials are read from a Kubernetes `Secret` using either
HTTP basic auth or a bearer token.

```yaml
apiVersion: mimir.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  uri: https://mimir.example.com
  # Optional component overrides. Default to uri when omitted.
  rulerUri: https://mimir.example.com/prometheus
  alertmanagerUri: https://mimir.example.com
  # Optional tenant/organization ID for multi-tenancy.
  orgId: mytenant
  # Optional additional headers sent with every request.
  headers:
    X-Scope-OrgID: mytenant
  credentials:
    source: Secret
    authType: token
    tokenAuth:
      tokenSecretRef:
        name: example-creds
        namespace: crossplane-system
        key: token
```

For basic authentication use `authType: basic` with `basicAuth.usernameSecretRef`
and `basicAuth.passwordSecretRef`. An optional `tls` block supports `insecure`
and CA/client certificate references.

### Managed Resources

Cluster-scoped alertmanager configuration:

```yaml
apiVersion: alertmanager.mimir.crossplane.io/v1
kind: Config
metadata:
  name: example-alertmanager-config
spec:
  forProvider:
    receiver:
      - name: default
    route:
      - receiver: default
  providerConfigRef:
    name: default
```

Cluster-scoped ruler alerting rules:

```yaml
apiVersion: ruler.mimir.crossplane.io/v1
kind: RuleGroupAlerting
metadata:
  name: example-alerting-rules
spec:
  forProvider:
    name: example_alerts
    namespace: default
    interval: 30s
    rule:
      - alert: InstanceDown
        expr: up == 0
        for: 5m
        annotations:
          summary: "Instance {{ $labels.instance }} is down"
        labels:
          severity: critical
  providerConfigRef:
    name: default
```

Namespaced resources additionally require a `metadata.namespace` and reference a
`ClusterProviderConfig` (or a namespaced `ProviderConfig`):

```yaml
apiVersion: ruler.mimir.m.crossplane.io/v1
kind: RuleGroupRecording
metadata:
  name: example-recording-rules
  namespace: default
spec:
  forProvider:
    name: example_recording
    namespace: default
    interval: 30s
    rule:
      - expr: up
        record: job:up:count
  providerConfigRef:
    kind: ClusterProviderConfig
    name: default
```

More manifests are available under [`examples/cluster`](examples/cluster) and
[`examples/namespaced`](examples/namespaced).

## Developing

Run against a Kubernetes cluster:

```console
make run
```

Build, push, and install:

```console
make all
```

Build the provider binary:

```console
make build
```

Regenerate deepcopy methods, Crossplane method sets, and CRDs after changing the
API types:

```console
go generate ./apis/...
```

Run end-to-end tests against a local control plane:

```console
make e2e
```

## Report a Bug

For filing bugs, suggesting improvements, or requesting new features, please
open an [issue](https://github.com/loafoe/provider-mimir/issues).
