package pkg

import (
	"fmt"
	"os"

	"go-simpler.org/env"
)

type ComposeConf struct{
	ManifestDir string `env:"MANIFEST_DIR"`
}

var (
	Settings *ComposeConf
)

func LoadSettings() (*ComposeConf, error){
	settings := ComposeConf{}	
	err := env.Load(&settings, nil)
	if settings.ManifestDir == ""{
    settings.ManifestDir = "~/.config/compose/manifests"
	}
	if err := os.MkdirAll(settings.ManifestDir, 0755); err != nil{
		return nil, fmt.Errorf("error to create manifest directory: %s", err)
	}
	//TODO: add lazy execution with once..
	if err != nil{
		return &settings, fmt.Errorf("improperly configured: %v", err)
	}
	Settings = &settings
	return Settings, nil
}
