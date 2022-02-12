@if(dev)
package platform

import (
	"deploy.test/pkg/policy"
)

_meta: labels: "kubernetes.io/environment": "dev"

app: deploy: policy.#Dev.#Deployment
