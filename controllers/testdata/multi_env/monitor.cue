@if(mon)
package platform

app: monitor: {
	apiVersion: "monitoring.coreos.com/v1"
	kind:       "ServiceMonitor"
	metadata:   _meta
}
