# caddy-ingress-controller

A helm chart for the Caddy Kubernetes ingress controller

## TL;DR:

```bash
helm install my-release caddy-ingress-controller\
  --repo https://caddyserver.github.io/ingress/ \
  --namespace=caddy-system
```

## Introduction

This chart bootstraps a caddy-ingress-deployment deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Helm 3+
- Kubernetes 1.19+

## Installing the Chart

```bash
helm repo add caddyserver https://caddyserver.github.io/ingress/
helm install my-release caddyserver/caddy-ingress-controller --namespace=caddy-system
```

## Uninstalling the Chart

To uninstall `my-release`:

```console
$ helm uninstall my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

> **Tip**: List all releases using `helm list` or start clean with `helm uninstall my-release`

## Additional Configuration

## Troubleshooting

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"caddy/ingress"` |  |
| image.tag | string | `"latest"` |  |
| imagePullSecrets | list | `[]` |  |
| ingressController.config.acmeCA | string | `""` |  |
| ingressController.config.acmeEABKeyId | string | `""` |  |
| ingressController.config.acmeEABMacKey | string | `""` |  |
| ingressController.config.debug | bool | `false` |  |
| ingressController.config.email | string | `""` |  |
| ingressController.config.metrics | bool | `true` |  |
| ingressController.config.onDemandTLS | bool | `false` |  |
| ingressController.config.proxyProtocol | bool | `false` |  |
| ingressController.rbac.create | bool | `true` |  |
| ingressController.verbose | bool | `false` |  |
| ingressController.leaseId | string | `""` |  |
| ingressController.watchNamespace | string | `""` |  |
| minikube | bool | `false` |  |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podDisruptionBudget.maxUnavailable | string | `nil` |  |
| podDisruptionBudget.minAvailable | int | `1` |  |
| podSecurityContext | object | `{}` |  |
| replicaCount | int | `2` |  |
| resources | object | `{}` |  |
| securityContext.allowPrivilegeEscalation | bool | `true` |  |
| securityContext.capabilities.add[0] | string | `"NET_BIND_SERVICE"` |  |
| securityContext.capabilities.drop[0] | string | `"ALL"` |  |
| securityContext.runAsGroup | int | `0` |  |
| securityContext.runAsUser | int | `0` |  |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.name | string | `"caddy-ingress-controller"` |  |
| tolerations | list | `[]` |  |