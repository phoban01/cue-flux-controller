package platform

import (
	corev1 "k8s.io/api/core/v1"
	"github.com/phoban01/cuedemo/examples/multi-env/pkg/app"
)

resources: ns: corev1.#Namespace & {
	apiVersion: "v1"
	kind:       "Namespace"
	metadata: {
		name: _env
	}
}

resources: (app.#App & {
	input: {
		meta: name:      "podinfo"
		meta: namespace: _env
		image: "ghcr.io/stefanprodan/podinfo"
		tag:   "6.0.3"
	}
}).out
