package charts

import (
	"encoding/base64"
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
	secrets := make(map[string]interface{})
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
		} else if resource.Kind == "Secret" {
			name := getStringFromMap(resource.Metadata, "name")
			if name != "" {
				secrets[name] = resource.Data
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
			app, err := extractAppInfo(resource, configMaps, secrets)
			if err == nil && app != nil {
				apps = append(apps, *app)
			}
		}
	}
	return apps, nil
}


func extractAppInfo(resource spec.Resource, configMaps map[string]interface{}, secrets map[string]interface{}) (*spec.App, error) {
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
		Mounts:  make(map[string]string),
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


	// Handle environment variables
	if envInterface, exists := container["env"]; exists {
		if envSlice, ok := envInterface.([]interface{}); ok {
			for _, env := range envSlice {
				if envMap, ok := env.(map[string]interface{}); ok {
					envKey := getStringFromMap(envMap, "name")
					if envKey == "" {
						continue // Skip if no name
					}
					
					// Handle direct value
					if value, exists := envMap["value"]; exists {
						app.Configs[envKey] = fmt.Sprintf("%v", value)
					}
					
					// Handle valueFrom
					if valueFrom, exists := envMap["valueFrom"]; exists {
						if valueFromMap, ok := valueFrom.(map[string]interface{}); ok {
							
							// Handle configMapKeyRef
							if configMapKeyRef, exists := valueFromMap["configMapKeyRef"]; exists {
								if keyRefMap, ok := configMapKeyRef.(map[string]interface{}); ok {
									configMapName := getStringFromMap(keyRefMap, "name")
									key := getStringFromMap(keyRefMap, "key")
									if config, exists := configMaps[configMapName]; exists {
								    if cfg, ok := config.(map[string]interface{}); ok{
											if value, exists := cfg[key]; exists {
												app.Configs[envKey] = fmt.Sprintf("%v", value)
											}
										}else{
											return nil, fmt.Errorf("invalid configmap state")
										}
									}
								}
							}
							
							// Handle fieldRef
							if fieldRef, exists := valueFromMap["fieldRef"]; exists {
								if fieldRefMap, ok := fieldRef.(map[string]interface{}); ok {
									fieldPath := getStringFromMap(fieldRefMap, "fieldPath")
									if fieldPath != "" {
										value := getValueFromFieldPath(resource, fieldPath)
										if value != "" {
											app.Configs[envKey] = value
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

	if volumeMountsInterface, exists := container["volumeMounts"]; exists {
		if volumeMounts, ok := volumeMountsInterface.([]interface{}); ok {
			if volumesInterface, exists := templateSpec["volumes"]; exists {
				if volumes, ok := volumesInterface.([]interface{}); ok {
					for _, volumeMount := range volumeMounts {
						if mountMap, ok := volumeMount.(map[string]interface{}); ok {
							mountName := getStringFromMap(mountMap, "name")
							mountPath := getStringFromMap(mountMap, "mountPath")
							subPath := getStringFromMap(mountMap, "subPath")
	
							for _, volume := range volumes {
								if volumeMap, ok := volume.(map[string]interface{}); ok {
									volumeName := getStringFromMap(volumeMap, "name")
									if volumeName == mountName {
										// Handle ConfigMap volumes
										if configMap, exists := volumeMap["configMap"]; exists {
											if configMapMap, ok := configMap.(map[string]interface{}); ok {
												configMapName := getStringFromMap(configMapMap, "name")
												if config, exists := configMaps[configMapName]; exists {
													if cfgData, ok := config.(map[string]interface{}); ok {
														// If subPath is specified, mount specific file at mountPath
														if subPath != "" {
															if value, exists := cfgData[subPath]; exists {
																app.Mounts[mountPath] = fmt.Sprintf("%v", value)
															}
														} else if itemsInterface, hasItems := configMapMap["items"]; hasItems {
															// Check if specific items are defined
															if items, ok := itemsInterface.([]interface{}); ok {
																for _, item := range items {
																	if itemMap, ok := item.(map[string]interface{}); ok {
																		key := getStringFromMap(itemMap, "key")
																		path := getStringFromMap(itemMap, "path")
																		if value, exists := cfgData[key]; exists {
																			fullPath := mountPath + "/" + path
																			app.Mounts[fullPath] = fmt.Sprintf("%v", value)
																		}
																	}
																}
															}
														} else {
															// Mount all keys from ConfigMap
															for key, value := range cfgData {
																fullPath := mountPath + "/" + key
																app.Mounts[fullPath] = fmt.Sprintf("%v", value)
															}
														}
													}
												}
											}
										}
										
										// Handle Secret volumes
										if secret, exists := volumeMap["secret"]; exists {
											if secretMap, ok := secret.(map[string]interface{}); ok {
												secretName := getStringFromMap(secretMap, "secretName")
												if secretData, exists := secrets[secretName]; exists {
													if secData, ok := secretData.(map[string]interface{}); ok {
														// If subPath is specified, mount specific file at mountPath
														if subPath != "" {
															if encodedValue, exists := secData[subPath]; exists {
																if decodedBytes, err := base64.StdEncoding.DecodeString(fmt.Sprintf("%v", encodedValue)); err == nil {
																	app.Mounts[mountPath] = string(decodedBytes)
																} else {
																	fmt.Printf("warning: failed to decode base64 for secret %s key %s: %v\n", secretName, subPath, err)
																}
															}
														} else if itemsInterface, hasItems := secretMap["items"]; hasItems {
															// Check if specific items are defined
															if items, ok := itemsInterface.([]interface{}); ok {
																for _, item := range items {
																	if itemMap, ok := item.(map[string]interface{}); ok {
																		key := getStringFromMap(itemMap, "key")
																		path := getStringFromMap(itemMap, "path")
																		if encodedValue, exists := secData[key]; exists {
																			// Decode base64
																			if decodedBytes, err := base64.StdEncoding.DecodeString(fmt.Sprintf("%v", encodedValue)); err == nil {
																				fullPath := mountPath + "/" + path
																				app.Mounts[fullPath] = string(decodedBytes)
																			} else {
																				fmt.Printf("warning: failed to decode base64 for secret %s key %s: %v\n", secretName, key, err)
																			}
																		}
																	}
																}
															}
														} else {
															// Mount all keys from Secret
															for key, encodedValue := range secData {
																if decodedBytes, err := base64.StdEncoding.DecodeString(fmt.Sprintf("%v", encodedValue)); err == nil {
																	fullPath := mountPath + "/" + key
																	app.Mounts[fullPath] = string(decodedBytes)
																} else {
																	fmt.Printf("warning: failed to decode base64 for secret %s key %s: %v\n", secretName, key, err)
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
						}
					}
				}
			}
		}
	}

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

func getValueFromFieldPath(resource spec.Resource, fieldPath string) string {
	// Split the field path by dots to navigate nested fields
	parts := strings.Split(fieldPath, ".")
	
	// Start with the resource as the root
	var current interface{} = map[string]interface{}{
		"metadata": resource.Metadata,
		"spec":     resource.Spec,
	}
	
	// Navigate through the path
	for _, part := range parts {
		if currentMap, ok := current.(map[string]interface{}); ok {
			if value, exists := currentMap[part]; exists {
				current = value
			} else {
				return ""
			}
		} else {
			return ""
		}
	}
	
	// Convert final value to string
	if str, ok := current.(string); ok {
		return str
	}
	
	return fmt.Sprintf("%v", current)
}
