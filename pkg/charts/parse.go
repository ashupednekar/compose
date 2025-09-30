package charts

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ashupednekar/compose/pkg/spec"
	"go.yaml.in/yaml/v3"
)

func (utils *ChartUtils) Parse(chart string, valuesPath string, setValues []string, useHostNetwork bool) ([]spec.App, error) {
	rel, err := utils.Template(chart, valuesPath, setValues)
	if err != nil {
		fmt.Printf("error templating chart: %v\n", err)
	}

	resources := strings.Split(rel.Manifest, "---")

	configMaps := make(map[string]interface{})
	secrets := make(map[string]interface{})
	services := make(map[string]spec.ServiceInfo)
	usedPorts := make(map[int]string) // port -> service name mapping for conflict detection
	var apps []spec.App

	// First pass: collect ConfigMaps, Secrets, and Services
	for _, content := range resources {
		content = strings.TrimSpace(content)
		if content == "" {
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
		} else if resource.Kind == "Service" {
			serviceInfo, err := extractServiceInfo(resource, useHostNetwork, usedPorts)
			if err != nil {
				return nil, fmt.Errorf("error processing service: %v", err)
			}
			if serviceInfo != nil {
				name := getStringFromMap(resource.Metadata, "name")
				services[name] = *serviceInfo
			}
		}
	}

	if useHostNetwork {
		configMaps = replaceServiceNamesWithLocalhost(configMaps, services)
	}

	// Second pass: process Deployments and StatefulSets
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
			// Extract all containers (main + sidecars) from the pod
			podApps, err := extractPodApps(resource, configMaps, secrets, services, useHostNetwork)
			if err != nil {
				fmt.Printf("error extracting pod apps: %v\n", err)
				continue
			}
			
			if len(podApps) > 0 {
				// Handle sidecar networking
				if len(podApps) > 1 && !useHostNetwork {
					// Multiple containers in pod - setup shared network namespace
					mainApp := &podApps[0]
					mainApp.NetworkMode = ""
					
					for i := 1; i < len(podApps); i++ {
						sidecar := &podApps[i]
						sidecar.NetworkMode = fmt.Sprintf("service:%s", mainApp.Name)
						sidecar.Ports = []string{} // Sidecars don't expose ports directly
					}
				} else if useHostNetwork {
					// Host networking for all containers
					for i := range podApps {
						podApps[i].NetworkMode = "host"
						podApps[i].Ports = []string{} // No port mapping needed with host network
					}
				}
				
				apps = append(apps, podApps...)
			}
		}
	}
	
	return apps, nil
}

func extractServiceInfo(resource spec.Resource, useHostNetwork bool, usedPorts map[int]string) (*spec.ServiceInfo, error) {
	name := getStringFromMap(resource.Metadata, "name")
	if name == "" {
		return nil, fmt.Errorf("service missing metadata.name")
	}
	ports, ok := resource.Spec["ports"]
	if !ok {
		return nil, fmt.Errorf("service missing ports")
	}
	portsSlice, ok := ports.([]interface{})
	if !ok {
		return nil, fmt.Errorf("service ports not in expected format")
	}
	serviceInfo := &spec.ServiceInfo{
		Name:     name,
		Ports:    []spec.PortInfo{},
		Selector: make(map[string]string),
	}
	if selector, exists := resource.Spec["selector"]; exists {
		if selectorMap, ok := selector.(map[string]interface{}); ok {
			for k, v := range selectorMap {
				if strVal, ok := v.(string); ok {
					serviceInfo.Selector[k] = strVal
				}
			}
		}
	}
	for _, portInterface := range portsSlice {
		if portMap, ok := portInterface.(map[string]interface{}); ok {
			portInfo := spec.PortInfo{}
			
			if port, exists := portMap["port"]; exists {
				if portInt, ok := port.(int); ok {
					portInfo.Port = portInt
				} else if portStr, ok := port.(string); ok {
					if p, err := strconv.Atoi(portStr); err == nil {
						portInfo.Port = p
					}
				}
			}
			
			if targetPort, exists := portMap["targetPort"]; exists {
				if targetPortInt, ok := targetPort.(int); ok {
					portInfo.Port = targetPortInt
				} else if targetPortStr, ok := targetPort.(string); ok {
					if tp, err := strconv.Atoi(targetPortStr); err == nil {
						portInfo.Port = tp
					}
				}
			} else {
				// If targetPort is not specified, it defaults to port
				portInfo.Port = portInfo.Port
			}
			
			if protocol, exists := portMap["protocol"]; exists {
				if protocolStr, ok := protocol.(string); ok {
					portInfo.Protocol = protocolStr
				}
			} else {
				portInfo.Protocol = "TCP"
			}

			if useHostNetwork {
				if existingService, exists := usedPorts[portInfo.Port]; exists {
					return nil, fmt.Errorf("port conflict: port %d is already used by service %s, cannot be used by service %s", 
						portInfo.Port, existingService, name)
				}
				usedPorts[portInfo.Port] = name
			}
			serviceInfo.Ports = append(serviceInfo.Ports, portInfo)
		}
	}

	return serviceInfo, nil
}

func replaceServiceNamesWithLocalhost(configMaps map[string]interface{}, services map[string]spec.ServiceInfo) map[string]interface{} {
	updatedConfigMaps := make(map[string]interface{})
	
	for configMapName, configMapData := range configMaps {
		if cfgMap, ok := configMapData.(map[string]interface{}); ok {
			updatedCfgMap := make(map[string]interface{})
			
			for key, value := range cfgMap {
				if strValue, ok := value.(string); ok {
					updatedValue := strValue
					// Replace each service name with localhost
					for serviceName := range services {
						pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(serviceName) + `\b`)
						updatedValue = pattern.ReplaceAllString(updatedValue, "localhost")
					}
					updatedCfgMap[key] = updatedValue
				} else {
					updatedCfgMap[key] = value
				}
			}
			updatedConfigMaps[configMapName] = updatedCfgMap
		} else {
			updatedConfigMaps[configMapName] = configMapData
		}
	}
	
	return updatedConfigMaps
}

// Extract all containers from a pod (main + sidecars)
func extractPodApps(resource spec.Resource, configMaps map[string]interface{}, secrets map[string]interface{}, services map[string]spec.ServiceInfo, useHostNetwork bool) ([]spec.App, error) {
	podName := getStringFromMap(resource.Metadata, "name")
	if podName == "" {
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

	// Get pod labels for service matching
	labels := make(map[string]string)
	if metadata, exists := specs["metadata"]; exists {
		if metadataMap, ok := metadata.(map[string]interface{}); ok {
			if labelsInterface, exists := metadataMap["labels"]; exists {
				if labelsMap, ok := labelsInterface.(map[string]interface{}); ok {
					for k, v := range labelsMap {
						if strVal, ok := v.(string); ok {
							labels[k] = strVal
						}
					}
				}
			}
		}
	}

	var apps []spec.App
	
	// Process each container
	for i, containerInterface := range containers {
		container, ok := containerInterface.(map[string]interface{})
		if !ok {
			continue
		}

		containerName := getStringFromMap(container, "name")
		if containerName == "" {
			containerName = fmt.Sprintf("%s-%d", podName, i)
		}

		app := spec.App{
			Name:    containerName,
			Type:    resource.Kind,
			Image:   getStringFromMap(container, "image"),
			Configs: make(map[string]string),
			Mounts:  make(map[string]string),
			Ports:   []string{},
		}

		// Only add ports to the first container (main container)
		if i == 0 {
			for _, serviceInfo := range services {
				if matchesSelector(labels, serviceInfo.Selector) {
					for _, portInfo := range serviceInfo.Ports {
						if useHostNetwork {
							app.Ports = append(app.Ports, fmt.Sprintf("%d:%d", portInfo.Port, portInfo.Port))
						} else {
							app.Ports = append(app.Ports, fmt.Sprintf("%d:%d", portInfo.Port, portInfo.Port))
						}
					}
					break
				}
			}
		}

		// Extract command and args
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

		// Extract lifecycle hooks
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

		// Extract envFrom (ConfigMap references)
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
							continue
						}
						
						if value, exists := envMap["value"]; exists {
							app.Configs[envKey] = fmt.Sprintf("%v", value)
						}
						
						if valueFrom, exists := envMap["valueFrom"]; exists {
							if valueFromMap, ok := valueFrom.(map[string]interface{}); ok {
								if configMapKeyRef, exists := valueFromMap["configMapKeyRef"]; exists {
									if keyRefMap, ok := configMapKeyRef.(map[string]interface{}); ok {
										configMapName := getStringFromMap(keyRefMap, "name")
										key := getStringFromMap(keyRefMap, "key")
										if config, exists := configMaps[configMapName]; exists {
											if cfg, ok := config.(map[string]interface{}); ok {
												if value, exists := cfg[key]; exists {
													app.Configs[envKey] = fmt.Sprintf("%v", value)
												}
											} else {
												return nil, fmt.Errorf("invalid configmap state")
											}
										}
									}
								}
								
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

		// Handle volume mounts
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

		apps = append(apps, app)
	}
	
	return apps, nil
}

func matchesSelector(labels map[string]string, selector map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	
	for key, value := range selector {
		if labelValue, exists := labels[key]; !exists || labelValue != value {
			return false
		}
	}
	
	return true
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
	parts := strings.Split(fieldPath, ".")
	
	var current interface{} = map[string]interface{}{
		"metadata": resource.Metadata,
		"spec":     resource.Spec,
	}
	
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
	
	if str, ok := current.(string); ok {
		return str
	}
	
	return fmt.Sprintf("%v", current)
}
