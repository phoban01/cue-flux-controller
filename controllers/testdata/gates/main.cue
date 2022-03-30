package main

import (
	corev1 "k8s.io/api/core/v1"
)

deployTag:  string @tag(gate)
deployGate: deployTag == "dummy"

resources: corev1.#ConfigMap & {
	apiVersion: "v1"
	kind:       "ConfigMap"
	metadata: {
		name:      string @tag(name)
		namespace: string @tag(namespace)
	}
	data: {
		foo: "bar"
	}
}

out: [ resources]
