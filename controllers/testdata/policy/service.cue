package kube

env: *"dev" | string @tag(env,short=dev|stg|prd)

_name:      *"podinfo-dev" | string @tag(name)
_namespace: *"default" | string     @tag(namespace)
_port:      *8080 | int             @tag(port,type=int)

apiVersion: "v1"
kind:       "Service"
metadata: {
	name:      _name
	namespace: _namespace
}
spec: {
	selector: {
		app: "podinfo"
	}
	ports: [
		{
			port: _port
		},
	]
}
