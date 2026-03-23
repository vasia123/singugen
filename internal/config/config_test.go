package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// No config file, no env vars — should return defaults.
	t.Setenv("SINGUGEN_CONFIG", "/nonexistent/path.yaml")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Supervisor.MaxRestarts != 5 {
		t.Errorf("MaxRestarts = %d, want 5", cfg.Supervisor.MaxRestarts)
	}
	if cfg.Supervisor.RestartWindow != 2*time.Minute {
		t.Errorf("RestartWindow = %v, want 2m", cfg.Supervisor.RestartWindow)
	}
	if cfg.Supervisor.HealthCheckInterval != 10*time.Second {
		t.Errorf("HealthCheckInterval = %v, want 10s", cfg.Supervisor.HealthCheckInterval)
	}
	if cfg.Supervisor.ChildBinary != "/usr/local/bin/singugen-agent" {
		t.Errorf("ChildBinary = %q, want /usr/local/bin/singugen-agent", cfg.Supervisor.ChildBinary)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want info", cfg.Log.Level)
	}
	if cfg.Log.Format != "text" {
		t.Errorf("Log.Format = %q, want text", cfg.Log.Format)
	}
	if cfg.Agent.DataPath != "data" {
		t.Errorf("Agent.DataPath = %q, want data", cfg.Agent.DataPath)
	}
}

func TestLoad_YAMLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	yaml := `
telegram:
  token: "test-token-123"
  allow_from: [111, 222]

supervisor:
  max_restarts: 10
  restart_window: 5m
  child_binary: "/custom/agent"

log:
  level: "debug"
  format: "json"
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SINGUGEN_CONFIG", path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Telegram.Token != "test-token-123" {
		t.Errorf("Telegram.Token = %q, want test-token-123", cfg.Telegram.Token)
	}
	if len(cfg.Telegram.AllowFrom) != 2 || cfg.Telegram.AllowFrom[0] != 111 {
		t.Errorf("Telegram.AllowFrom = %v, want [111 222]", cfg.Telegram.AllowFrom)
	}
	if cfg.Supervisor.MaxRestarts != 10 {
		t.Errorf("MaxRestarts = %d, want 10", cfg.Supervisor.MaxRestarts)
	}
	if cfg.Supervisor.RestartWindow != 5*time.Minute {
		t.Errorf("RestartWindow = %v, want 5m", cfg.Supervisor.RestartWindow)
	}
	if cfg.Supervisor.ChildBinary != "/custom/agent" {
		t.Errorf("ChildBinary = %q, want /custom/agent", cfg.Supervisor.ChildBinary)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want debug", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format = %q, want json", cfg.Log.Format)
	}
	// Defaults preserved for unset fields.
	if cfg.Supervisor.HealthCheckInterval != 10*time.Second {
		t.Errorf("HealthCheckInterval = %v, want 10s (default)", cfg.Supervisor.HealthCheckInterval)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("SINGUGEN_CONFIG", "/nonexistent/path.yaml")
	t.Setenv("SINGUGEN_TELEGRAM_TOKEN", "env-token")
	t.Setenv("SINGUGEN_LOG_LEVEL", "error")
	t.Setenv("SINGUGEN_LOG_FORMAT", "json")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Telegram.Token != "env-token" {
		t.Errorf("Telegram.Token = %q, want env-token", cfg.Telegram.Token)
	}
	if cfg.Log.Level != "error" {
		t.Errorf("Log.Level = %q, want error", cfg.Log.Level)
	}
	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format = %q, want json", cfg.Log.Format)
	}
}

func TestLoad_EnvExpansion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	t.Setenv("MY_SECRET_TOKEN", "expanded-token")

	yaml := `
telegram:
  token: "${MY_SECRET_TOKEN}"
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SINGUGEN_CONFIG", path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Telegram.Token != "expanded-token" {
		t.Errorf("Telegram.Token = %q, want expanded-token", cfg.Telegram.Token)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")

	if err := os.WriteFile(path, []byte("{{{{not yaml"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SINGUGEN_CONFIG", path)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid YAML, got nil")
	}
}
