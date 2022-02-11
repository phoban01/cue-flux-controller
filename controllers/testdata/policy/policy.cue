package kube

env: string @tag(env)

_name:      string
_namespace: string
_port:      int

apiVersion: string
kind:       string
metadata: {
	name:      string
	namespace: string
}
spec: {
	selector: {
		app: string
	}
	ports: [
		{
			port:       >=8000 & <=9000
			targetPort: *port | int
		},
	]
}
