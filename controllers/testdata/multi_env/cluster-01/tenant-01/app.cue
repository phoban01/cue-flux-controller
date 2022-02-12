package platform

import "encoding/yaml"

_appconf: {
	replicas: 4
	name:     "proxy"
	image:    "nginx"
	tag:      "latest"
	config:
		PROXY: yaml.Marshal(_config)
}

_config: {
	whitelist_urls: ["http://example.org"]
}
