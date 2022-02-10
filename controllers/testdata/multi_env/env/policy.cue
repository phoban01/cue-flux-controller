package kube

if env == "dev" {
	kubernetes: deployment: spec: replicas: <=4
}
