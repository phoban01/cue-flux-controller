package policy

import (
	"deploy.test/pkg/components"
)

#Dev: #Deployment: components.#Deployment & {
	spec: replicas: <=4
}
