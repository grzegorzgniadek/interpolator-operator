# interpolator



![Version: 0.9.0](https://img.shields.io/badge/Version-0.9.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.9.0](https://img.shields.io/badge/AppVersion-0.9.0-informational?style=flat-square) 

feat: use texttemplate and sprig







## Requirements

| Repository | Name | Version |
|------------|------|---------|
| https://prometheus-community.github.io/helm-charts | prometheus-operator-crds | 14.0.* |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| controllerManager.manager.args[0] | string | `"--metrics-bind-address=:8080"` |  |
| controllerManager.manager.args[1] | string | `"--leader-elect"` |  |
| controllerManager.manager.args[2] | string | `"--health-probe-bind-address=:8081"` |  |
| controllerManager.manager.containerSecurityContext.allowPrivilegeEscalation | bool | `false` |  |
| controllerManager.manager.containerSecurityContext.capabilities.drop[0] | string | `"ALL"` |  |
| controllerManager.manager.image.repository | string | `"ghcr.io/grzegorzgniadek/interpolator-operator"` |  |
| controllerManager.manager.image.tag | string | `"0.9.0"` |  |
| controllerManager.manager.imagePullPolicy | string | `"Always"` |  |
| controllerManager.manager.resources.limits.cpu | string | `"200m"` |  |
| controllerManager.manager.resources.limits.memory | string | `"128Mi"` |  |
| controllerManager.manager.resources.requests.cpu | string | `"10m"` |  |
| controllerManager.manager.resources.requests.memory | string | `"64Mi"` |  |
| controllerManager.replicas | int | `1` |  |
| controllerManager.serviceAccount.annotations | object | `{}` |  |
| kubernetesClusterDomain | string | `"cluster.local"` |  |
| metricsService.enabled | bool | `true` |  |
| metricsService.ports[0].name | string | `"metrics"` |  |
| metricsService.ports[0].port | int | `8080` |  |
| metricsService.ports[0].protocol | string | `"TCP"` |  |
| metricsService.ports[0].targetPort | int | `8080` |  |
| metricsService.type | string | `"ClusterIP"` |  |
| prometheusCRDS.enabled | bool | `false` |  |
| prometheusMonitor.enabled | bool | `false` |  |
| prometheusMonitor.interval | string | `"15s"` |  |