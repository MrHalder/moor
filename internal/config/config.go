package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

const (
	appName       = "moor"
	configFile    = "config.yaml"
	maxConfigSize = 1 << 20 // 1 MB
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

	// Check file size before reading to prevent OOM on crafted config
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return Config{}, fmt.Errorf("reading config %s: %w", path, err)
	}
	if info.Size() > maxConfigSize {
		return Config{}, fmt.Errorf("config file %s too large (%d bytes, max %d)", path, info.Size(), maxConfigSize)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config %s: %w", path, err)
	}

	// Apply defaults and enforce bounds
	if cfg.Settings.RefreshIntervalSecs < 1 || cfg.Settings.RefreshIntervalSecs > 3600 {
		cfg.Settings.RefreshIntervalSecs = 2
	}
	if cfg.Settings.GracePeriodSecs < 1 || cfg.Settings.GracePeriodSecs > 300 {
		cfg.Settings.GracePeriodSecs = 3
	}
	switch cfg.Settings.DefaultOutput {
	case "table", "json":
		// valid
	default:
		cfg.Settings.DefaultOutput = "table"
	}

	return cfg, nil
}

// Save writes the config to disk, creating the directory if needed.
// Refuses to follow symlinks to prevent arbitrary file overwrites.
func Save(cfg Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config dir %s: %w", dir, err)
	}

	// Verify the config directory is not a symlink
	if err := rejectSymlink(dir); err != nil {
		return fmt.Errorf("config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	path := ConfigPath()

	// Remove symlink if present (prevents writing through a symlink)
	if err := rejectSymlink(path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("removing symlink at %s: %w", path, removeErr)
		}
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config %s: %w", path, err)
	}

	return nil
}

// Reset overwrites config with defaults.
func Reset() error {
	return Save(DefaultConfig())
}

// rejectSymlink returns an error if path is a symlink.
func rejectSymlink(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink (refusing to follow for security)", path)
	}
	return nil
}
