package app

#Config: {
	meta: {
		name:      string
		namespace: *"default" | string
		labels: app: *meta.name | string
		annotations: {...}
	}
	image:    string
	tag:      string
	port:     *9898 | int
	replicas: *1 | int
}

#App: {
	input: #Config
	out: {
		sa:      #ServiceAccount & {_config: input}
		deploy:  #Deployment & {_config:     input, _serviceAccount: sa.metadata.name}
		service: #Service & {_config:        input}
	}
}
