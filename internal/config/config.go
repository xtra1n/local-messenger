package config

import "os"

type Config struct {
	HTTPPort string
	LogLevel string
	DBPath   string
}

func Load() *Config {
	cfg := &Config{
		HTTPPort: "8080",
		LogLevel: "debug",
		DBPath:   "./data/messenger.db",
	}

	if port := os.Getenv("HTTP_PORT"); port != "" {
		cfg.HTTPPort = port
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.LogLevel = level
	}

	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		cfg.DBPath = dbPath
	}

	return cfg
}
