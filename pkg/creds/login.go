package creds

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ashupednekar/compose/pkg/charts"
	"github.com/ashupednekar/compose/pkg/spec"
)


func AuthenticateWithRegistry(method string, engine string, force bool) (*charts.ChartUtils, error) {
	c, err := charts.NewChartUtils()
	if err != nil{
		return nil, fmt.Errorf("error initiating chart utils: %s", err)
	}
	switch method {
		case "dockerconfig":
			log.Printf("using docker config with engine: %s\n", engine)
			if err := authenticateWithDockerConfig(c, engine, force); err != nil {
				log.Printf("error authenticating with docker config: %s\n", err)
				return nil, err
			}
		case "tokenrefresher":
			log.Println("using token refresher")
		default:
			return nil, fmt.Errorf("unknown authentication method: %s\n", method)
		}
		log.Println("login completed")
		return c, nil
}


func getDockerConfigPath(engine string) (string, error) {
	var configDir string
	
	switch engine {
	case "docker":
		configDir = filepath.Join(os.Getenv("HOME"), ".docker")
	case "podman":
		uid := os.Getuid()
    runtimeDir := fmt.Sprintf("/run/user/%d/containers", uid)
    authPath := filepath.Join(runtimeDir, "auth.json")
    if _, err := os.Stat(authPath); err == nil {
        return authPath, nil
    }
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			configDir = filepath.Join(xdgConfig, "containers")
		} else {
			configDir = filepath.Join(os.Getenv("HOME"), ".config", "containers")
		}
	default:
		return "", fmt.Errorf("unsupported engine: %s", engine)
	}
	configPath := filepath.Join(configDir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", fmt.Errorf("config file not found at %s", configPath)
	}
	return configPath, nil
}

func readDockerConfig(configPath string) (*spec.DockerConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var config spec.DockerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	return &config, nil
}

func extractAuthInfo(auth spec.DockerAuth, registry string) (*spec.AuthInfo, error) {
	//TODO: support keychains, later
	var username, password string
	if auth.Auth != "" {
		decoded, err := base64.StdEncoding.DecodeString(auth.Auth)
		if err != nil {
			return nil, fmt.Errorf("failed to decode auth string: %w", err)
		}
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid auth format")
		}
		username, password = parts[0], parts[1]
	} else if auth.Username != "" && auth.Password != "" {
		username, password = auth.Username, auth.Password
	} else {
		return nil, fmt.Errorf("no valid authentication found for registry")
	}
	return &spec.AuthInfo{
		Username: username,
		Password: password,
		Registry: registry,
	}, nil
}


func authenticateWithDockerConfig(c *charts.ChartUtils, engine string, force bool) error {
	configPath, err := getDockerConfigPath(engine)
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}
	log.Printf("Reading config from: %s\n", configPath)
	config, err := readDockerConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to read docker config: %w", err)
	}
	if len(config.Auths) == 0 {
		return fmt.Errorf("no authentication entries found in config file")
	}
	successCount := 0
	for registry, auth := range config.Auths {
		log.Printf("Processing registry: %s\n", registry)
		authInfo, err := extractAuthInfo(auth, registry)
		if err != nil {
			log.Printf("Warning: failed to extract auth info for %s: %s\n", registry, err)
			continue
		}	
		if err := c.Authenticate(authInfo); err != nil {
			log.Printf("Warning: failed to authenticate with %s: %s\n", registry, err)
			continue
		}
		successCount++
	}
	if successCount == 0 {
		return fmt.Errorf("failed to authenticate with any registry")
	}
	return nil
}
