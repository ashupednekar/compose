package charts

import (
	"fmt"
	"os"

	"github.com/ashupednekar/compose/pkg"
	"github.com/ashupednekar/compose/pkg/spec"
	"go.yaml.in/yaml/v3"
)


func WriteCompose(apps []spec.App, name string) error {
	dockerCompose := spec.DockerCompose{
		Services: make(map[string]spec.DockerComposeService),
	}
	for _, app := range apps{
		service := spec.DockerComposeService{
			Image: app.Image,
			Command: app.Command,
			Restart: "unless-stopped",
			Networks: []string{}, // TODO: convert k8s netpol to this
			Environment: app.Configs,
		  NetworkMode: "host",
		}
		dockerCompose.Services[app.Name] = service
	}
	data, err := yaml.Marshal(&dockerCompose)
	if err != nil{
		return fmt.Errorf("error marshaling docker-compose to yaml: %v\n", err)
	}
	composeDir := fmt.Sprintf("%s/%s", pkg.Settings.ManifestDir, name)
	if err := os.MkdirAll(composeDir, 0755); err != nil{
		return fmt.Errorf("error creating manifest subdirectory")
	}
	if err := os.WriteFile(
		fmt.Sprintf("%s/docker-compose.yaml", composeDir), data, 0644,
	); err != nil{
		return fmt.Errorf("error writing docker-compose yaml %v\n", err)
	}
	fmt.Printf("docker-compose.yaml written to %s", composeDir)
	return nil
}
