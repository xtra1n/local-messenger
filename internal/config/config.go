package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Log      LogConfig
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./etc")
	v.AddConfigPath("/etc/local-messenger")

	v.SetEnvPrefix("MESSENGER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", "8080")
	v.SetDefault("database.path", "./data/messenger.db")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %v", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Server.Port == "" {
		return fmt.Errorf("server.port is required")
	}
	if cfg.Database.Path == "" {
		return fmt.Errorf("database.path is required")
	}

	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[cfg.Log.Level] {
		return fmt.Errorf("invalid log.level: %s (must be debug, info, warn, error)", cfg.Log.Level)
	}

	validFormats := map[string]bool{
		"text": true,
		"json": true,
	}
	if !validFormats[cfg.Log.Format] {
		return fmt.Errorf("invalid log.format: %s (must be text or json)", cfg.Log.Format)
	}

	return nil
}
