package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for SinguGen.
type Config struct {
	Telegram   TelegramConfig   `yaml:"telegram"`
	Supervisor SupervisorConfig `yaml:"supervisor"`
	Agent      AgentConfig      `yaml:"agent"`
	SelfUpdate SelfUpdateConfig `yaml:"self_update"`
	Log        LogConfig        `yaml:"log"`
}

type SelfUpdateConfig struct {
	Enabled       bool          `yaml:"enabled"`
	AutoPush      bool          `yaml:"auto_push"`
	PushBranch    string        `yaml:"push_branch"`
	BuildTimeout  time.Duration `yaml:"build_timeout"`
	TestTimeout   time.Duration `yaml:"test_timeout"`
	ProtectedDirs []string      `yaml:"protected_dirs"`
}

type TelegramConfig struct {
	Token     string  `yaml:"token"`
	AllowFrom []int64 `yaml:"allow_from"`
}

type SupervisorConfig struct {
	HealthCheckInterval time.Duration `yaml:"healthcheck_interval"`
	MaxRestarts         int           `yaml:"max_restarts"`
	RestartWindow       time.Duration `yaml:"restart_window"`
	ChildBinary         string        `yaml:"child_binary"`
}

type AgentConfig struct {
	WorkspacePath   string        `yaml:"workspace_path"`
	DataPath        string        `yaml:"data_path"`
	ClaudeBinary    string        `yaml:"claude_binary"`
	ClaudeModel     string        `yaml:"claude_model"`
	ClaudeTimeout   time.Duration `yaml:"claude_timeout"`
	ClaudeMaxRetries int           `yaml:"claude_max_retries"`
	QueueSize        int           `yaml:"queue_size"`
	MemoryPath       string        `yaml:"memory_path"`
	IdleTimeout      time.Duration `yaml:"idle_timeout"`
	DreamOnShutdown  bool          `yaml:"dream_on_shutdown"`
	MaxDreamDuration time.Duration `yaml:"max_dream_duration"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func defaults() Config {
	return Config{
		Supervisor: SupervisorConfig{
			HealthCheckInterval: 10 * time.Second,
			MaxRestarts:         5,
			RestartWindow:       2 * time.Minute,
			ChildBinary:         "/usr/local/bin/singugen-agent",
		},
		Agent: AgentConfig{
			WorkspacePath:    "data/workspace",
			DataPath:         "data",
			ClaudeBinary:     "claude",
			ClaudeTimeout:    3 * time.Minute,
			ClaudeMaxRetries: 10,
			QueueSize:        64,
			MemoryPath:       "data/memory",
			IdleTimeout:      15 * time.Minute,
			DreamOnShutdown:  true,
			MaxDreamDuration: 5 * time.Minute,
		},
		SelfUpdate: SelfUpdateConfig{
			Enabled:       false,
			PushBranch:    "self-update",
			BuildTimeout:  2 * time.Minute,
			TestTimeout:   2 * time.Minute,
			ProtectedDirs: []string{"cmd/singugen", "internal/supervisor"},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// Path returns the config file path from SINGUGEN_CONFIG env or default.
func Path() string {
	if v := os.Getenv("SINGUGEN_CONFIG"); v != "" {
		return v
	}
	return "configs/singugen.yaml"
}

// Load reads config from YAML file, expands env vars, applies env overrides.
// Missing config file is not an error — defaults are used.
func Load() (*Config, error) {
	cfg := defaults()
	path := Path()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			applyEnv(&cfg)
			return &cfg, nil
		}
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	expanded := os.ExpandEnv(string(data))
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	applyEnv(&cfg)
	return &cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("SINGUGEN_TELEGRAM_TOKEN"); v != "" {
		cfg.Telegram.Token = v
	}
	if v := os.Getenv("SINGUGEN_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("SINGUGEN_LOG_FORMAT"); v != "" {
		cfg.Log.Format = v
	}
}
