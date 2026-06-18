package config

import (
	"os"

	// "golang.org/x/tools/go/cfg"
	"gopkg.in/yaml.v3"
)

type Config struct {
	TerminalConfig TermCfg `yaml:"termcfg"`
	BackgroundPath string `yaml:"bgpath"`
}

type TermCfg struct {
	Terminal string `yaml:"terminal"`
	TermHotKey uint32 `yaml:"termhotkey"`
}

func LoadConfig(filepath string) (*Config,error) {
	var cfg Config
	data,err := os.ReadFile(filepath)
	if err != nil {
		return nil,err
	}

	err = yaml.Unmarshal(data,&cfg)
	if err != nil {
		return nil,err
	}

	return &cfg,nil
}