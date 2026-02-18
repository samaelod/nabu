package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	LogLines  int    `json:"log_lines"`
	LogsDir   string `json:"logs_dir"`
	RecentDir string `json:"recent_dir"`
}

var (
	defaultConfig *Config
	once          sync.Once
)

func Default() *Config {
	return &Config{
		LogLines:  1000,
		LogsDir:   "logs",
		RecentDir: "recent",
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()

	if path == "" {
		// Try default locations
		defaultPaths := []string{
			"nabu.json",
			".nabu.json",
			filepath.Join(os.Getenv("HOME"), ".config", "nabu", "config.json"),
		}

		for _, p := range defaultPaths {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}

		if path == "" {
			return cfg, nil
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Apply defaults for any zero values
	if cfg.LogLines <= 0 {
		cfg.LogLines = 1000
	}
	if cfg.LogsDir == "" {
		cfg.LogsDir = "logs"
	}
	if cfg.RecentDir == "" {
		cfg.RecentDir = "recent"
	}

	return cfg, nil
}

// LoadDefault loads the config once and caches it
func LoadDefault() (*Config, error) {
	var err error
	once.Do(func() {
		defaultConfig, err = Load("")
	})
	if err != nil {
		return Default(), err
	}
	return defaultConfig, nil
}
