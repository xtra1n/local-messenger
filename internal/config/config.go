package config

import "os"

type Config struct {
	HTTPPort string
	LogLevel string
}

func Load() *Config {
	cfg := &Config{
		HTTPPort: "8080",
		LogLevel: "debug",
	}

	if port := os.Getenv("HTTP_PORT"); port != "" {
		cfg.HTTPPort = port
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.LogLevel = level
	}

	return cfg
}
