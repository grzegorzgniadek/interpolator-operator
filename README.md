# interpolator-operator
[![Release Container](https://github.com/grzegorzgniadek/interpolator-operator/actions/workflows/release-container.yaml/badge.svg?branch=master)](https://github.com/grzegorzgniadek/interpolator-operator/releases)
[![Release Charts](https://github.com/grzegorzgniadek/interpolator-operator/actions/workflows/release-charts.yaml/badge.svg?branch=master)](https://github.com/grzegorzgniadek/interpolator-operator/releases)
[![Version](https://img.shields.io/github/v/tag/grzegorzgniadek/interpolator-operator?sort=semver&label=Version&color=darkgreen)](https://github.com/grzegorzgniadek/interpolator-operator/tags)
[![Apache 2.0 license](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/license/apache-2-0)

interpolator-operator is a secret data interpolation system for Kubernetes.

## Supported Kubernetes versions

interpolator-operator has been developed for and tested with Kubernetes 1.28.

## How it works

When Custom resource is created, controller takes secret keys and values and creates new secret as outputSecretName


## Architecture and components

- a `Deployment` to run interpolator's controller,

```bash
$ kubectl top pods
NAME                                               CPU(cores)   MEMORY(bytes)
interpolator-operator-controller-manager-669d64b6cc-md889   2m           21Mi
```

## Installation

1. Install interpolator-operator's Helm chart from [charts](https://grzegorzgniadek.github.io/interpolator-operator) repository:

```bash
helm upgrade --install \
     --create-namespace --namespace interpolator-operator-system \
     interpolator-operator interpolator-operator \
     --repo https://grzegorzgniadek.github.io/interpolator-operator/
```

## Installation with plain YAML files

1. You can use Helm to generate plain YAML files and then deploy these YAML files with `kubectl apply` or whatever you want:

```bash
helm template --namespace interpolator-operator-system \
     interpolator-operator interpolator-operator \
     --repo https://grzegorzgniadek.github.io/interpolator-operator/ \
     > /tmp/interpolator.yaml
kubectl create namespace interpolator-operator-system
kubectl apply -f /tmp/interpolator.yaml --namespace interpolator-operator-system
```

2. You can use bundled yaml files from dist directory
Build the installer for the image built and published in the registry:

```sh
make build-installer
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

1. Using the installer

```bash
kubectl apply -f dist/install.yaml
```

## Configuration and customization

You can see the full list of parameters (along with their meaning and default values) in the chart's [values.yaml](https://github.com/grzegorzgniadek/interpolator-operator/blob/master/charts/interpolator/values.yaml) file.


#### Customize resources

```bash
helm upgrade --install \
     --create-namespace --namespace interpolator-operator-system  \
     interpolator-operator interpolator-operator \
     --repo https://grzegorzgniadek.github.io/interpolator-operator/ \
     --set controllerManager.manager.resources.limits.cpu=200m
```

#### Turn on Prometheus Service Monitor(Metrics)

```bash
helm install \
     --create-namespace --namespace interpolator-operator-system  \
     interpolator-operator interpolator-operator \
     --repo https://grzegorzgniadek.github.io/interpolator-operator/ \
     --set prometheusCRDS.enabled=true \
     --set prometheusMonitor.enabled=true \
     --set prometheusMonitor.interval=15s
```

## Use sample resource
```bash
kubectl apply -f https://raw.githubusercontent.com/grzegorzgniadek/interpolator-operator/master/config/samples/dummy-resources.yaml
```
If we want to create ConfigMap as result interpolated resource
```bash
kubectl apply -f https://raw.githubusercontent.com/grzegorzgniadek/interpolator-operator/master/config/samples/inter_v1_interpolator1-configmap.yaml
```
If we want to create Secret as result interpolated resource
```bash
kubectl apply -f https://raw.githubusercontent.com/grzegorzgniadek/interpolator-operator/master/config/samples/inter_v1_interpolator2-secret.yaml
```