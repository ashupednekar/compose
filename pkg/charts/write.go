package charts

import (
	"fmt"
	"os"
	"strings"

	"github.com/ashupednekar/compose/pkg"
	"github.com/ashupednekar/compose/pkg/spec"
	"go.yaml.in/yaml/v3"
)


func WriteCompose(apps []spec.App, name string) error {
	useRootDir := len(apps) == 1
	for _, app := range apps{
		fmt.Printf("Name: %v\n", app.Name)
		fmt.Printf("Image: %v\n", app.Image)
		fmt.Printf("Command: %v\n", app.Command)
		fmt.Printf("Envs: %v\n", app.Configs)
		fmt.Printf("Mounts: %v\n", app.Mounts)
		fmt.Printf("PostStart: %v\n===", app.PostStart)	
		dockerCompose := spec.DockerCompose{
			Services: make(map[string]spec.DockerComposeService),
		}
		var composeDir string
		if useRootDir && name == app.Name{
			composeDir = fmt.Sprintf("%s/%s", pkg.Settings.ManifestDir, app.Name)
		}else{
			composeDir = fmt.Sprintf("%s/%s/%s", pkg.Settings.ManifestDir, name, app.Name)
		}
		if err := os.MkdirAll(composeDir, 0755); err != nil{
	  	return fmt.Errorf("error creating manifest subdirectory")
	  }
		service := spec.DockerComposeService{
			Image: app.Image,
			Command: app.Command,
			Restart: "unless-stopped",
			Networks: []string{}, // TODO: convert k8s netpol to this
			Volumes: []string{},
			Environment: app.Configs,
		  NetworkMode: "host",
		}
		for mount, content := range app.Mounts{
			parts := strings.Split(mount, "/") 
			mountFileName := parts[len(parts)-1]
			//TODO: :Z/:z for podman permissions
			if err := os.WriteFile(
				fmt.Sprintf("%s/%s", composeDir, mountFileName), []byte(content), 0644,
			); err != nil{
				return fmt.Errorf("error writing mapped file: %v\n", err)
			}
			volumeMount := fmt.Sprintf("%s:%s", mountFileName, mount) 
			service.Volumes = append(service.Volumes, volumeMount)
		}
		dockerCompose.Services[app.Name] = service
    data, err := yaml.Marshal(&dockerCompose)
	  if err != nil{
	  	return fmt.Errorf("error marshaling docker-compose to yaml: %v\n", err)
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
