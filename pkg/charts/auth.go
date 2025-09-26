package charts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ashupednekar/compose/pkg/spec"
	"helm.sh/helm/v3/pkg/registry"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)


type ChartUtils struct{
	Client *registry.Client
}

func NewChartUtils() (*ChartUtils, error){
	registryClient, err := registry.NewClient()
	c := ChartUtils{Client: registryClient}
	if err != nil{
		return nil, fmt.Errorf("failed to create registry client")
	}
	return &c, nil
}

func getHelmConfigDir() string {
	if helmConfig := os.Getenv("HELM_CONFIG_HOME"); helmConfig != "" {
		return helmConfig
	}
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "helm")
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "helm")
}

func (utils *ChartUtils) Authenticate(authInfo *spec.AuthInfo) error {
	if savedAuthInfo, err := utils.readSavedAuthInfo(); err == nil {
		log.Printf("Using saved authentication configuration\n")
		authInfo = savedAuthInfo
	} else {
		if err := utils.saveAuthInfo(authInfo); err != nil {
			log.Printf("Warning: failed to save auth config: %v\n", err)
		}
	}
	helmConfigDir := getHelmConfigDir()
	credentialsFile := filepath.Join(helmConfigDir, "registry", "config.json")
	if err := os.MkdirAll(filepath.Dir(credentialsFile), 0755); err != nil {
		return fmt.Errorf("failed to create helm config directory: %w", err)
	}
	store, err := credentials.NewFileStore(credentialsFile)
	if err != nil {
		return fmt.Errorf("failed to create credential store: %w", err)
	}
	registryURL := strings.TrimPrefix(authInfo.Registry, "https://")
	registryURL = strings.TrimPrefix(registryURL, "http://")
	cred := auth.Credential{
		Username: authInfo.Username,
		Password: authInfo.Password,
	}
	ctx := context.Background()
	if err := store.Put(ctx, registryURL, cred); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}
	if err := utils.Client.Login(
		registryURL,
		registry.LoginOptBasicAuth(authInfo.Username, authInfo.Password),
	); err != nil {
		return fmt.Errorf("authentication test failed: %w", err)
	}
	return nil
}

func (utils *ChartUtils) saveAuthInfo(authInfo *spec.AuthInfo) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".config", "compose")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	configFile := filepath.Join(configDir, "config.json")
	data, err := json.MarshalIndent(authInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth info: %w", err)
	}
	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

func (utils *ChartUtils) readSavedAuthInfo() (*spec.AuthInfo, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	configFile := filepath.Join(homeDir, ".config", "compose", "config.json")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist")
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var authInfo spec.AuthInfo
	if err := json.Unmarshal(data, &authInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal auth info: %w", err)
	}
	return &authInfo, nil
}
