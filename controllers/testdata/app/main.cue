package main

import (
	podinfo "app.example/podinfo"
)

resources: (podinfo.#App & {
	input: {
		meta: {
			name:      string @tag(name)
			namespace: string @tag(namespace)
		}
		image: "ghcr.io/stefanprodan/podinfo"
		tag:   "6.0.3"
	}
}).out

out: [ for x in resources {x}]
