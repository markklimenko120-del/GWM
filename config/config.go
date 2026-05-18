package config

import (
	"fmt"
	"os"
	"strings"
	"github.com/spf13/viper"
)

type Config struct {
	TerminalConfig TermCfg `mapstructure:"termcfg"`
	BackgroundPath string `mapstructure:"bgpath"`
}

type TermCfg struct {
	Terminal string `mapstructure:"terminal"`
	TermHotKey [2]uint32 `mapstructure:"termhotkey`
}

func Load() (*Config,error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	
	viper.AutomaticEnv()

	viper.SetEnvKeyReplacer(strings.NewReplacer(".","_"))

	viper.SetDefault("bgpath","./backgrounds/bg1.jpg")
	viper.SetDefault("termcfg.terminal","kitty")
	viper.SetDefault("termcfg.termhotkey",[2]uint32{0xffe9,0xffe1})

	if err := viper.ReadInConfig(); err != nil {
		if _,ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil,fmt.Errorf("Ошибка чтения конфига! %v",err)
		}
		fmt.Print("Конфиг не найден,будут использоваться значения по умолчанию")
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg);err != nil {
		return nil,fmt.Errorf("Не удалось распарсить конфиг %v",err)
	}

	return &cfg,nil
}

func MustLoad() *Config {
	cfg,err := Load()
	if err != nil {
		fmt.Printf("Ошибка загрузки конфига! %v",err)
		os.Exit(1)
	}

	return cfg
}