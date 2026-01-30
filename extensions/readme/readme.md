## Overview

`provider-mimir` is a [Crossplane](https://crossplane.io/) provider for [Grafana Mimir](https://grafana.com/oss/mimir/). It allows platform teams to manage Mimir resources (like Alertmanager configurations, Rule Groups, etc.) alongside infrastructure and application deployments using Kubernetes-style APIs.

## Installation

Install the provider by applying the following `Provider` manifest to your Crossplane cluster:

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-mimir
spec:
  package: xpkg.upbound.io/loafoe/provider-mimir:v0.2.2
```

## Configuration

1. **Create a Secret** containing the necessary credentials (e.g., API tokens or Basic Auth) to communicate with the Mimir API.
2. **Apply a `ProviderConfig`** to configure the connection:

```yaml
apiVersion: mimir.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: mimir-creds
      key: credentials
  # Address of the Mimir API
  address: http://mimir.monitoring.svc.cluster.local:8080
```

## License

This project is licensed under the Apache 2.0 License.
