package platform

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

#KRM: {
	apiVersion: string
	kind:       string
	metadata:   metav1.#ObjectMeta
}

#HasOwnerLabel: #KRM & {
	metadata: labels: owner: string
}
