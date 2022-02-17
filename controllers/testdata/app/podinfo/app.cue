package podinfo

#Config: {
	meta: {
		name:      string
		namespace: *"default" | string
		labels: app: *meta.name | string
	}
	image:    string
	tag:      string
	port:     *9898 | int
	replicas: *1 | int
}

#App: {
	input: #Config
	out: {
		deploy: #Deployment & {_config: input}
	}
}
