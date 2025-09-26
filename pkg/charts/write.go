package charts

import (
	"fmt"
	"os"

	"github.com/ashupednekar/compose/pkg"
	"github.com/ashupednekar/compose/pkg/spec"
	"go.yaml.in/yaml/v3"
)


func WriteCompose(apps []spec.App, name string) error {
	for _, app := range apps{
		dockerCompose := spec.DockerCompose{
			Services: make(map[string]spec.DockerComposeService),
		}
		service := spec.DockerComposeService{
			Image: app.Image,
			Command: app.Command,
			Restart: "unless-stopped",
			Networks: []string{}, // TODO: convert k8s netpol to this
			Environment: app.Configs,
		  NetworkMode: "host",
		}
		dockerCompose.Services[app.Name] = service
    data, err := yaml.Marshal(&dockerCompose)
	  if err != nil{
	  	return fmt.Errorf("error marshaling docker-compose to yaml: %v\n", err)
	  }
		var composeDir string
		if name == app.Name{
			composeDir = fmt.Sprintf("%s/%s", pkg.Settings.ManifestDir, app.Name)
		}else{
			composeDir = fmt.Sprintf("%s/%s/%s", pkg.Settings.ManifestDir, name, app.Name)
		}
	  if err := os.MkdirAll(composeDir, 0755); err != nil{
	  	return fmt.Errorf("error creating manifest subdirectory")
	  }
	  if err := os.WriteFile(
	  	fmt.Sprintf("%s/docker-compose.yaml", composeDir), data, 0644,
	  ); err != nil{
	  	return fmt.Errorf("error writing docker-compose yaml %v\n", err)
	  }
	  fmt.Printf("docker-compose.yaml written to %s\n", composeDir)
	}
	return nil
}
