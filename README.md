# interpolator-operator

[![Releases](https://github.com/grzegorzgniadek/interpolator-operator/actions/workflows/release.yml/badge.svg?branch=release)](https://github.com/grzegorzgniadek/interpolator-operator/releases)
[![Go report card](https://goreportcard.com/badge/github.com/grzegorzgniadek/interpolator-operator)](https://goreportcard.com/report/github.com/grzegorzgniadek/interpolator-operator)
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
interpolator-controller-manager-669d64b6cc-md889   2m           21Mi
```

## Installation

1. Install interpolator-operator's Helm chart from our [charts](https://grzegorzgniadek.github.io/interpolator-operator) repository:

```bash
helm upgrade --install \
     --create-namespace --namespace interpolator-system \
     interpolator interpolator \
     --repo https://grzegorzgniadek.github.io/interpolator-operator/
```

## Installation with plain YAML files

You can use Helm to generate plain YAML files and then deploy these YAML files with `kubectl apply` or whatever you want:

```bash
helm template --namespace interpolator-system \
     interpolator interpolator \
     --repo https://grzegorzgniadek.github.io/interpolator-operator/ \
     > /tmp/interpolator.yaml
kubectl create namespace interpolator-system
kubectl apply -f /tmp/interpolator.yaml --namespace interpolator-system
```

## Configuration and customization

You can see the full list of parameters (along with their meaning and default values) in the chart's [values.yaml](https://github.com/grzegorzgniadek/interpolator-operator/blob/master/charts/interpolator/values.yaml) file.


```bash
helm upgrade --install \
     --create-namespace --namespace interpolator-system  \
     interpolator interpolator \
     --repo https://grzegorzgniadek.github.io/interpolator-operator/ \
     --set controllerManager.manager.resources.limits.cpu=200m
```
