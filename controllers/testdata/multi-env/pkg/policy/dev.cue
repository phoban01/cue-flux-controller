package policy

import (
	appsv1 "k8s.io/api/apps/v1"
)

#Dev: #Deployment: appsv1.#Deployment & {
	spec: replicas: <=4
}
