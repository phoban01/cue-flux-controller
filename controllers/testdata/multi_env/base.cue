package kube

env: string @tag(env)

_name:      string
_namespace: string

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
				if env != "prd" {
					{
						name:  "sidecar"
						image: "proxy:latest"
					}
				},
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
