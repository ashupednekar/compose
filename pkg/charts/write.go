package charts

import (
	"fmt"

	"github.com/ashupednekar/compose/pkg/spec"
)




func WriteCompose(apps []spec.App, path string){
	for _, app := range apps{
		service := spec.DockerComposeService{
			Image: app.Image,
			Restart: "unless-stopped",
			Networks: []string{}, // TODO: convert k8s netpol to this
			Environment: app.Configs,
		}
		fmt.Println(service)
	}
}
