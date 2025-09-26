package charts

import (
	"fmt"
	"strings"

	"github.com/ashupednekar/compose/pkg/spec"
	"go.yaml.in/yaml/v3"
)

func (utils *ChartUtils) Parse(chart string, valuesPath string, setValues []string) ([]spec.App, error){
	rel, err := utils.Template(chart, valuesPath, setValues)
	if err != nil{
		fmt.Printf("error templating chart: %v\n", err)
	}

	resources := strings.Split(rel.Manifest, "---")

	configMaps := make(map[string]interface{})
	var apps []spec.App

	for _, content := range resources{
		content = strings.TrimSpace(content)
		if content == ""{
			continue
		}
		var resource spec.Resource
		if err := yaml.Unmarshal([]byte(content), &resource); err != nil {
			fmt.Printf("warning: error unmarshalling resource - %s", err)
			continue
		}
		if resource.Kind == "ConfigMap" {
			name := getStringFromMap(resource.Metadata, "name")
			if name != "" {
				configMaps[name] = resource.Data
			}
		}
	}
	for _, content := range resources {
		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}
		var resource spec.Resource
		if err := yaml.Unmarshal([]byte(content), &resource); err != nil {
			continue
		}
		if resource.Kind == "Deployment" || resource.Kind == "StatefulSet" {
			app, err := extractAppInfo(resource, configMaps)
			if err == nil && app != nil {
				apps = append(apps, *app)
			}
		}
	}
	return apps, nil
}


func extractAppInfo(resource spec.Resource, configMaps map[string]interface{}) (*spec.App, error) {
	name := getStringFromMap(resource.Metadata, "name")
	if name == "" {
		return nil, fmt.Errorf("missing metadata.name")
	}

	specs, ok := resource.Spec["template"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing spec.template")
	}

	templateSpec, ok := specs["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing spec.template.spec")
	}

	containersInterface, ok := templateSpec["containers"]
	if !ok {
		return nil, fmt.Errorf("missing containers")
	}

	containers, ok := containersInterface.([]interface{})
	if !ok || len(containers) == 0 {
		return nil, fmt.Errorf("no containers found")
	}

	container, ok := containers[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid container format")
	}

	app := &spec.App{
		Name:    name,
		Type:    resource.Kind,
		Image:   getStringFromMap(container, "image"),
		Configs: make(map[string]string),
	}

	if cmdInterface, exists := container["command"]; exists {
		if cmdSlice, ok := cmdInterface.([]interface{}); ok {
			for _, cmd := range cmdSlice {
				if cmdStr, ok := cmd.(string); ok {
					app.Command = append(app.Command, cmdStr)
				}
			}
		}
	}
	if argsInterface, exists := container["args"]; exists {
		if argsSlice, ok := argsInterface.([]interface{}); ok {
			for _, arg := range argsSlice {
				if argStr, ok := arg.(string); ok {
					app.Command = append(app.Command, argStr)
				}
			}
		}
	}

	if lifecycleInterface, exists := container["lifecycle"]; exists {
		if lifecycle, ok := lifecycleInterface.(map[string]interface{}); ok {
			if postStartInterface, exists := lifecycle["postStart"]; exists {
				if postStart, ok := postStartInterface.(map[string]interface{}); ok {
					hook := &spec.PostStartHook{}
					if execInterface, exists := postStart["exec"]; exists {
						if exec, ok := execInterface.(map[string]interface{}); ok {
							hook.Type = "exec"
							if cmdInterface, exists := exec["command"]; exists {
								if cmdSlice, ok := cmdInterface.([]interface{}); ok {
									for _, cmd := range cmdSlice {
										if cmdStr, ok := cmd.(string); ok {
											hook.Command = append(hook.Command, cmdStr)
										}
									}
								}
							}
						}
					} else if httpGetInterface, exists := postStart["httpGet"]; exists {
						if httpGet, ok := httpGetInterface.(map[string]interface{}); ok {
							hook.Type = "httpGet"
							path := getStringFromMap(httpGet, "path")
							port := getStringFromMap(httpGet, "port")
							hook.HTTPGet = fmt.Sprintf("%s:%s", path, port)
						}
					}
					if hook.Type != "" {
						app.PostStart = hook
					}
				}
			}
		}
	}

	if envFromInterface, exists := container["envFrom"]; exists {
		if envFromSlice, ok := envFromInterface.([]interface{}); ok {
			for _, envFrom := range envFromSlice {
				if envFromMap, ok := envFrom.(map[string]interface{}); ok {
					if configMapRef, exists := envFromMap["configMapRef"]; exists {
						if configMapRefMap, ok := configMapRef.(map[string]interface{}); ok {
							configMapName := getStringFromMap(configMapRefMap, "name")
							if config, exists := configMaps[configMapName]; exists {
                cfgMap, ok := config.(map[string]interface{})
                if !ok {
                  return nil, fmt.Errorf("invalid configmap state")
                }
                for k, v := range cfgMap {
                    app.Configs[k] = fmt.Sprintf("%v", v) 
                }
							}
						}
					}
				}
			}
		}
	}


	// TODO: pending, valueFrom.fieldRef.fieldPath
	if envInterface, exists := container["env"]; exists {
		if envSlice, ok := envInterface.([]interface{}); ok {
			for _, env := range envSlice {
				if envMap, ok := env.(map[string]interface{}); ok {
					if value, exists := envMap["value"]; exists{
						key := getStringFromMap(envMap, "name")
						app.Configs[key] = value.(string)
					}
					if valueFrom, exists := envMap["valueFrom"]; exists {
						if valueFromMap, ok := valueFrom.(map[string]interface{}); ok {
							if configMapKeyRef, exists := valueFromMap["configMapKeyRef"]; exists {
								if keyRefMap, ok := configMapKeyRef.(map[string]interface{}); ok {
									configMapName := getStringFromMap(keyRefMap, "name")
									key := getStringFromMap(keyRefMap, "key")
									if config, exists := configMaps[configMapName]; exists {
								    if cfg, ok := config.(map[string]interface{}); ok{
											if value, exists := cfg[key]; exists {
												app.Configs[key] = value.(string)
											}
										}else{
											return nil, fmt.Errorf("invalid configmap state")
										}
										
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// if volumeMountsInterface, exists := container["volumeMounts"]; exists {
	// 	if volumeMounts, ok := volumeMountsInterface.([]interface{}); ok {
	// 		if volumesInterface, exists := templateSpec["volumes"]; exists {
	// 			if volumes, ok := volumesInterface.([]interface{}); ok {
	// 				for _, volumeMount := range volumeMounts {
	// 					if mountMap, ok := volumeMount.(map[string]interface{}); ok {
	// 						mountName := getStringFromMap(mountMap, "name")
	//
	// 						for _, volume := range volumes {
	// 							if volumeMap, ok := volume.(map[string]interface{}); ok {
	// 								volumeName := getStringFromMap(volumeMap, "name")
	// 								if volumeName == mountName {
	// 									if configMap, exists := volumeMap["configMap"]; exists {
	// 										if configMapMap, ok := configMap.(map[string]interface{}); ok {
	// 											configMapName := getStringFromMap(configMapMap, "name")
	// 											if config, exists := configMaps[configMapName]; exists {
	// 												if cm, ok := config.(string); ok{
	// 												  app.Configs[configMapName] = cm
	// 												}else{
	// 													fmt.Printf("warning: non string configmap, ignoring: %s\n", cm)
	// 												}
	// 											}
	// 										}
	// 									}
	// 								}
	// 							}
	// 						}
	// 					}
	// 				}
	// 			}
	// 		}
	// 	}
	// }

	return app, nil
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if value, exists := m[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}


