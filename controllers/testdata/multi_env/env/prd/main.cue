package kube

env:        "prd"
_name:      *"podinfo-prd" | string @tag(name)
_namespace: *"default" | string     @tag(namespace)

kubernetes: [
	#ServiceAccount,
	#Deployment & {
		spec: replicas: 10
	},
]
