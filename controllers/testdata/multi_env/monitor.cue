@if(mon)
package kube

kubernetes: monitor: {
	apiVersion: "monitoring.coreos.com/v1"
	kind:       "ServiceMonitor"
	metadata: {
		name:      _name
		namespace: _namespace
	}
}
