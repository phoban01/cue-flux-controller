package policy

import (
	"deploy.test/pkg/components"
)

#Staging: #Deployment: components.#Deployment & {
	spec: replicas: <=10
}
