package platform

import (
	"path"
)

_path:   string @tag(dir,var="cwd")
_tenant: path.Base(_path)

_appconf: {
	replicas: 1
	name:     "server"
	image:    "apache"
	tag:      "latest"
}
