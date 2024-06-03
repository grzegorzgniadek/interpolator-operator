# interpolator



![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.1.0](https://img.shields.io/badge/AppVersion-0.1.0-informational?style=flat-square) 

A Helm chart for Kubernetes

This helm chart is maintained and released by the fluxcd-community on a best effort basis.









## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| controllerManager.manager.args[0] | string | `"--leader-elect"` |  |
| controllerManager.manager.args[1] | string | `"--health-probe-bind-address=:8081"` |  |
| controllerManager.manager.containerSecurityContext.allowPrivilegeEscalation | bool | `false` |  |
| controllerManager.manager.containerSecurityContext.capabilities.drop[0] | string | `"ALL"` |  |
| controllerManager.manager.image.repository | string | `"registry.gitlab.ggniadekit.com/devops/container-images/controller"` |  |
| controllerManager.manager.image.tag | string | `"latest"` |  |
| controllerManager.manager.imagePullPolicy | string | `"Always"` |  |
| controllerManager.manager.imagePullSecrets[0].name | string | `"gitlab-registry"` |  |
| controllerManager.manager.resources.limits.cpu | string | `"200m"` |  |
| controllerManager.manager.resources.limits.memory | string | `"128Mi"` |  |
| controllerManager.manager.resources.requests.cpu | string | `"10m"` |  |
| controllerManager.manager.resources.requests.memory | string | `"64Mi"` |  |
| controllerManager.replicas | int | `1` |  |
| controllerManager.serviceAccount.annotations | object | `{}` |  |
| kubernetesClusterDomain | string | `"cluster.local"` |  |