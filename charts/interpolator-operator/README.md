# interpolator-operator



![Version: 0.13.0](https://img.shields.io/badge/Version-0.13.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.13.0](https://img.shields.io/badge/AppVersion-0.13.0-informational?style=flat-square) 

A Helm chart to distribute interpolator-operator









## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| certManager.enable | bool | `false` |  |
| crd.enable | bool | `true` |  |
| crd.keep | bool | `true` |  |
| manager.affinity | object | `{}` |  |
| manager.args[0] | string | `"--leader-elect"` |  |
| manager.enabled | bool | `true` |  |
| manager.image.pullPolicy | string | `"IfNotPresent"` |  |
| manager.image.repository | string | `"ghcr.io/grzegorzgniadek/interpolator-operator"` |  |
| manager.image.tag | string | `"0.13.0"` |  |
| manager.nodeSelector | object | `{}` |  |
| manager.podSecurityContext.runAsNonRoot | bool | `true` |  |
| manager.podSecurityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| manager.replicas | int | `1` |  |
| manager.resources.limits.cpu | string | `"500m"` |  |
| manager.resources.limits.memory | string | `"128Mi"` |  |
| manager.resources.requests.cpu | string | `"10m"` |  |
| manager.resources.requests.memory | string | `"64Mi"` |  |
| manager.securityContext.allowPrivilegeEscalation | bool | `false` |  |
| manager.securityContext.capabilities.drop[0] | string | `"ALL"` |  |
| manager.securityContext.readOnlyRootFilesystem | bool | `true` |  |
| manager.terminationGracePeriodSeconds | int | `10` |  |
| manager.tolerations | list | `[]` |  |
| metrics.enable | bool | `true` |  |
| metrics.port | int | `8443` |  |
| metrics.secure | bool | `true` |  |
| prometheus.enable | bool | `false` |  |
| rbac.helpers.enable | bool | `false` |  |
| rbac.namespaced | bool | `false` |  |
| serviceAccount.enable | bool | `true` |  |