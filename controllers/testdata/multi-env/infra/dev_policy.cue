@if(dev)
package platform

import (
	"github.com/phoban01/cuedemo/examples/multi-env/pkg/policy"
)

resources: deploy: policy.#Dev.#Deployment
resources: [_]: metadata: labels: "kubernetes.io/environment": "dev"
