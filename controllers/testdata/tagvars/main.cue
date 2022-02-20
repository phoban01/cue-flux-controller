package main

_os:        string @tag(os,var="os")
_namespace: string @tag(namespace)
_name:      _os + "-identity"

out: (#Test & {
	_config: {
		name:      _name
		namespace: _namespace
	}
}).out

#Test: {
	_config: {
		name:      string
		namespace: string
	}
	out: {
		apiVersion: "v1"
		kind:       "ServiceAccount"
		metadata: {
			name:      _config.name
			namespace: _config.namespace
		}
	}
}
