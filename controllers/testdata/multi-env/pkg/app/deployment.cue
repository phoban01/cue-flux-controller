package app

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

#Deployment: appsv1.#Deployment & {
	_config:         #Config
	_serviceAccount: string
	apiVersion:      "apps/v1"
	kind:            "Deployment"
	metadata:        _config.meta
	spec:            appsv1.#DeploymentSpec & {
		replicas: _config.replicas
		selector: matchLabels: app: _config.meta.name
		template: {
			metadata: labels: app: _config.meta.name
			spec: corev1.#PodSpec & {
				serviceAccountName: _serviceAccount
				containers: [
					{
						name: "podinfo"
						command: [
							"./podinfo",
							"--port=\(_config.port)",
						]
						image: "\(_config.image):\(_config.tag)"
						ports: [{
							containerPort: _config.port
						}]
					},
				]
			}
		}
	}
}
