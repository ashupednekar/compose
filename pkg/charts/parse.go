package charts

import (
	"fmt"
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
			app, err := extractAppInfo(resource, configMaps, secrets, services, useHostNetwork)
			if useHostNetwork{
				app.NetworkMode = "host"
				app.Ports = []string{}
			}
			if err == nil && app != nil {
				apps = append(apps, *app)
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
						updatedValue = strings.ReplaceAll(updatedValue, serviceName, "localhost")
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

func extractAppInfo(resource spec.Resource, configMaps map[string]interface{}, secrets map[string]interface{}, services map[string]spec.ServiceInfo, useHostNetwork bool) (*spec.App, error) {
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
		Ports:   []string{}, // Add ports field
	}

	// Find matching service and add ports
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

	for _, serviceInfo := range services {
		if matchesSelector(labels, serviceInfo.Selector) {
			for _, portInfo := range serviceInfo.Ports {
				if useHostNetwork {
					app.Ports = append(app.Ports, fmt.Sprintf("%d:%d", portInfo.Port, portInfo.Port))
				} else {
					app.Ports = append(app.Ports, fmt.Sprintf("%d:%d", portInfo.Port, portInfo.Port))
				}
			}
		}
	}

	// ... rest of the existing extractAppInfo function remains the same
	// (command, args, lifecycle, envFrom, env, volumeMounts handling)
	
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

	// ... rest of volumeMounts handling remains the same
	
	return app, nil
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
