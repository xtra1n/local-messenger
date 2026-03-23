package config

import (
	"os"
	"testing"
)

func TestLoad_Deafults(t *testing.T) {
	os.Clearenv()
	os.Setenv("MESSENGER_SERVER_PORT", "9090")
	defer os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Port != "9090" {
		t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "9090")
	}
	if cfg.Database.Path != "./data/messenger.db" {
		t.Errorf("Database.Path = %q, want %q", cfg.Database.Path, "./data/messenger.db")
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "info")
	}
}

func TestLoad_Validation_InvalidLevel(t *testing.T) {
	os.Clearenv()
	os.Setenv("MESSENGER_LOG_LEVEL", "invalid")
	defer os.Clearenv()

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid log level")
	}
}

func TestLoad_Validation_InvalidFormat(t *testing.T) {
	os.Clearenv()
	os.Setenv("MESSENGER_LOG_FORMAT", "xml")
	defer os.Clearenv()

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid log format")
	}
}

func TestLoad_Validation_EmptyPort(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	cfg := &Config{
		Server:   ServerConfig{Port: ""},
		Database: DatabaseConfig{Path: "./test.db"},
		Log:      LogConfig{Level: "info", Format: "text"},
	}

	err := validate(cfg)
	if err == nil {
		t.Fatal("validate() expected error for empty port")
	}
}
