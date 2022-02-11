package kube

kubernetes: deployment: spec: replicas: 2

k8s: [ for x in kubernetes {x}]
