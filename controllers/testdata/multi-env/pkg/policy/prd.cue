package policy

import (
	appsv1 "k8s.io/api/apps/v1"
)

#Prd: #Deployment: appsv1.#Deployment & {
	spec: replicas: >1
	spec: template: spec: containers: [{
		image: !~"latest$"
	}]
}
