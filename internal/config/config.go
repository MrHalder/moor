package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

const (
	appName    = "moor"
	configFile = "config.yaml"
)

// overrideDir allows tests to redirect config storage. Empty means use xdg default.
var overrideDir string

// configDir returns the moor config directory path.
func configDir() string {
	if overrideDir != "" {
		return overrideDir
	}
	return filepath.Join(xdg.ConfigHome, appName)
}

// ConfigPath returns the full path to the config file.
func ConfigPath() string {
	return filepath.Join(configDir(), configFile)
}

// Load reads the config from disk. Returns defaults if file doesn't exist.
func Load() (Config, error) {
	path := ConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return Config{}, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config %s: %w", path, err)
	}

	// Apply defaults for missing settings
	if cfg.Settings.RefreshIntervalSecs == 0 {
		cfg.Settings.RefreshIntervalSecs = 2
	}
	if cfg.Settings.GracePeriodSecs == 0 {
		cfg.Settings.GracePeriodSecs = 3
	}
	if cfg.Settings.DefaultOutput == "" {
		cfg.Settings.DefaultOutput = "table"
	}

	return cfg, nil
}

// Save writes the config to disk, creating the directory if needed.
func Save(cfg Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config dir %s: %w", dir, err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	path := ConfigPath()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config %s: %w", path, err)
	}

	return nil
}

// Reset overwrites config with defaults.
func Reset() error {
	return Save(DefaultConfig())
}
