package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	DbConnStr string `yaml:"db_conn_str"`
	JwtSecret string `yaml:"jwt_secret"`
}

func New(path string) (*Config, error) {
	var cfg Config

	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling config data: %w", err)
	}

	return &cfg, nil
}
