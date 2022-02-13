package platform

import (
	"deploy.test/pkg/components"
	"encoding/yaml"
	// "path"
)

// # TODO: extend example to allow configuring multiple applications
// # Question: how to validate policy in that scenario?

_env: *"dev" | string @tag(env,short=dev|stg|prd)
// _path:   string                     @tag(dir,var="cwd")
// _tenant: path.Base(_path)
_tenant: string @tag(namespace)

_meta: components.#Metadata & {
	name:      string @tag(name)
	namespace: _tenant
	labels: {
		app: *_meta.name | string @tag(name)
	}
}

// this should be its own component
// policy could be cross cutting... infra team defines policy for kubernetes components
// app teams define policu for app configs

_appconf: {
	name:     string
	replicas: int
	image:    string
	tag:      string
}

app: {
	ns:     components.#Namespace & {metadata: name: _meta.namespace}
	sa:     components.#ServiceAccount & {metadata:  _meta}
	cm:     components.#ConfigMap & {metadata:       _meta}
	deploy: components.#Deployment & {metadata:      _meta}
	// svc: components.#Service & {metadata:           _meta}
}

app: deploy: {
	spec: replicas: _appconf.replicas
	spec: template: spec: containers: [
		components.#Container & {
			name:  _appconf.name
			image: "\(_appconf.image):\(_appconf.tag)"
		},
	]
}

app: cm: data: _appconf.config

out: [ for k, v in app if (k == "cm" && v["data"] != _|_) || k != "cm" {v}]

oyaml: yaml.MarshalStream(out)
