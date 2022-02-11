@if(mon)
package kube

import "list"

_kinds: ["Deployment", "ServiceAccount"]

for k, x in kubernetes {
	if list.Contains(_kinds, x.kind) && x.metadata.namespace != "default" {
		kubernetes: "\(k)": {
			metadata: labels: {"cuelang.test.io/inject-sidecar": "true"}
		}
	}
}
