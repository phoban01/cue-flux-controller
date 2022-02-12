package platform

import (
	"deploy.test/pkg/components"
)

_meta: labels: "kubernetes.io/cluster-name": "cluster01"
_meta: annotations: "ingress/domain":        "private"

app: deploy: components.#Deployment & {
	spec: template: spec: serviceAccount: app.sa.metadata.name
}
