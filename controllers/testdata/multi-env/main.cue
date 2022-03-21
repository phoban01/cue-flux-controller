package platform

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

_env: *"dev" | string @tag(env,short=dev|stg|prd)

out: [ for x in resources {x}]

// resources will hold the resources we want to deploy, ensuring they are all KRMs
resources: [ID=_]: #KRM

// We'll define KRM to ensure all of our resources comply with Kubernetes Resource Model
#KRM: {
	metav1.#TypeMeta
	metadata:          metav1.#ObjectMeta
	["spec" | "data"]: runtime.#Object
}
