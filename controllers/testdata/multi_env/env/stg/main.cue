package kube

env:        "stg"
_name:      *"podinfo-stg" | string @tag(name)
_namespace: *"default" | string     @tag(namespace)

kubernetes: [
	#ServiceAccount,
	#Deployment & {
		spec: replicas: 5
	},
]
