package app

import (
	corev1 "k8s.io/api/core/v1"
)

#Service: corev1.#Service & {
	_config:    #Config
	apiVersion: "v1"
	kind:       "Service"
	metadata:   _config.meta
	spec:       corev1.#ServiceSpec & {
		type:     "ClusterIP"
		selector: _config.meta.labels
		ports: [{
			name:       "http"
			port:       _config.port
			targetPort: _config.port
			protocol:   "TCP"
		}]
	}
}
