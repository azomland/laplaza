package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type PlazaConfig struct {
	Title          string `toml:"title"`
	Domain         string `toml:"domain"`
	Port           int    `toml:"port"`
	MaxUsersPerBench int  `toml:"max_users_per_bench"`
	AllowAnonymous bool   `toml:"allow_anonymous"`
	History        bool   `toml:"history"`
	Ads            bool   `toml:"ads"`
	DataDir        string `toml:"data_dir"`
}

func Default() PlazaConfig {
	return PlazaConfig{
		Title:            "Mi Plaza",
		Domain:           "localhost",
		Port:             8080,
		MaxUsersPerBench: 33,
		AllowAnonymous:   true,
		History:          false,
		Ads:              false,
		DataDir:          "./data",
	}
}

func Load(path string) (PlazaConfig, error) {
	cfg := Default()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	_, err := toml.DecodeFile(path, &cfg)
	return cfg, err
}
