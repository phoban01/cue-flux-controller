package components

import (
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
)

#Metadata: {
	name:      string
	namespace: string
	labels: [string]:      string
	annotations: [string]: string
}

#KRM: {
	kind:       string
	apiVersion: string
	metadata:   #Metadata
}

#Container: {
	name:  string
	image: string
}

#Deployment: appsv1.#Deployment & {
	let _name = metadata.name

	apiVersion: "apps/v1"
	kind:       "Deployment"
	metadata:   #Metadata
	spec: {
		replicas: *1 | int
		selector: matchLabels: app: _name
		template: {
			metadata: labels: app: _name
			spec: containers: [ #Container]
		}
	}
}

#Namespace: corev1.#Namespace & {
	apiVersion: "v1"
	kind:       "Namespace"
}

#ServiceAccount: corev1.#ServiceAccount & {
	apiVersion: "v1"
	kind:       "ServiceAccount"
}

#ConfigMap: corev1.#ConfigMap & {
	apiVersion: "v1"
	kind:       "ConfigMap"
}
