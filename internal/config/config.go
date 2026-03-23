package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
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
	Path         string `mapstructure:"path"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type LogConfig struct {
	Level    string `mapstructure:"level"`
	Format   string `mapstructure:"format"`
	FilePath string `mapstructure:"file_path"`
}

func Load() (*Config, error) {
	// 1. Загружаем .env файл (если есть)
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			return nil, fmt.Errorf("failed to load .env: %w", err)
		}
	}

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
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
	v.SetDefault("log.file_path", "")

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
	var errs []string

	if cfg.Server.Port == "" {
		errs = append(errs, "server.port is required")
	}

	if cfg.Server.Host == "" {
		errs = append(errs, "server.host is required")
	}

	if cfg.Database.Path == "" {
		errs = append(errs, "database.path is required")
	}

	if cfg.Database.MaxOpenConns < 1 {
		errs = append(errs, "database.max_open_conns must be >= 1")
	}

	if cfg.Database.MaxIdleConns < 1 {
		errs = append(errs, "database.max_idle_conns must be >= 1")
	}

	if cfg.Database.MaxIdleConns > cfg.Database.MaxOpenConns {
		errs = append(errs, "database.max_idle_conns must be <= database.max_open_conns")
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
		errs = append(errs, fmt.Sprintf("invalid log.format: %s (must be text or json)", cfg.Log.Format))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

func (c *Config) String() string {
	return fmt.Sprintf(
		"Config{Server: %s:%s, DB: %s, Log: %s/%s}",
		c.Server.Host,
		c.Server.Port,
		c.Database.Path,
		c.Log.Level,
		c.Log.Format,
	)
}
