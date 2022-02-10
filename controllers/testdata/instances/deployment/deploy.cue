package kube

_name:      string                @tag(name)
_namespace: *"cue-build" | string @tag(namespace)

#deployment: {
	apiVersion: "apps/v1"
	kind:       "Deployment"
	metadata: {
		name:      _name
		namespace: _namespace
		labels: app: _name
	}
	spec: {
		replicas: 1
		selector: matchLabels: app: _name
		template: {
			metadata: labels: app: _name
			spec: containers: [{
				name:  _name
				image: _name
			}]
		}
	}
}

#serviceAccount: {
	apiVersion: "v1"
	kind:       "ServiceAccount"
	metadata: {
		name:      _name
		namespace: _namespace
	}
}

kubernetes: [#deployment, #serviceAccount]
