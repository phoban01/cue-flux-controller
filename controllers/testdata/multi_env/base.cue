package kube

env:        *"dev" | string     @tag(env,short=dev|stg|prd)
_name:      *"podinfo" | string @tag(name)
_namespace: *"default" | string @tag(namespace)

kubernetes: deployment: {
	apiVersion: "apps/v1"
	kind:       "Deployment"
	metadata: {
		name:      _name
		namespace: _namespace
		labels: app: _name
	}
	spec: {
		replicas: *1 | int
		selector: matchLabels: app: _name
		template: {
			metadata: labels: app: _name
			spec: containers: [
				{
					name:  _name
					image: _name
				},
				...,
			]
		}
	}
}

kubernetes: serviceaccount: {
	apiVersion: "v1"
	kind:       "ServiceAccount"
	metadata: {
		name:      _name
		namespace: _namespace
	}
}
