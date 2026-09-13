//go:build container

package config

import (
	"fmt"

	"github.com/caarlos0/env/v10"
)

func loadEnv() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse env vars: %w", err)
	}
	return &cfg, nil
}
