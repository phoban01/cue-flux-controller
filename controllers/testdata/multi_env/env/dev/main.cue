package kube

env:        *"dev" | string         @tag(env,short=dev|stg|prd)
_name:      *"podinfo-dev" | string @tag(name)
_namespace: *"default" | string     @tag(namespace)

kubernetes: deployment: spec: replicas: 2

k8s: [ for x in kubernetes {x}]
