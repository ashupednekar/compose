package pkg

import (
	"fmt"
	"go-simpler.org/env"
)

type ComposeConf struct{
	ManifestDir string `env:"MANIFEST_DIR,required"`

}

var (
	Settings *ComposeConf
)

func LoadSettings() (*ComposeConf, error){
	settings := ComposeConf{}
	err := env.Load(&settings, nil)
	//TODO: add lazy execution with once..
	if err != nil{
		return &settings, fmt.Errorf("improperly configured: %v", err)
	}
	Settings = &settings
	return Settings, nil
}
