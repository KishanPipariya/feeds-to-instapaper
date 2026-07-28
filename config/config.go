package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Instapaper struct {
	Username string `toml:"username"`
	Password string `toml:"password"`
}

type Hooks struct {
	NewArticle []Hook `toml:"new_article"`
}

type Hook struct {
	Spawn []string `toml:"spawn"`
}

type Feeds struct {
	URLs                  []string `toml:"urls"`
	MaxConcurrency        int      `toml:"max_concurrency"`
	RequestTimeoutSeconds int      `toml:"request_timeout_seconds"`
	MaxResponseBytes      int64    `toml:"max_response_bytes"`
	MaxItems              int      `toml:"max_items"`
}

const (
	DefaultMaxConcurrency        = 4
	DefaultRequestTimeoutSeconds = 30
	DefaultMaxResponseBytes      = 10 * 1024 * 1024
	DefaultMaxItems              = 1000
)

type Config struct {
	Instapaper `toml:"instapaper"`
	Hooks      `toml:"hooks"`
	Feeds      `toml:"feeds"`
}

func Load() (*Config, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	configPath := filepath.Join(configDir, "feeds-to-instapaper", "config.toml")
	info, err := os.Lstat(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat config file %s: %w", configPath, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("config file %s must be a regular file", configPath)
	}
	if info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("config file %s must not be readable by group or other users; run chmod 600 %s", configPath, configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	config := Config{Feeds: Feeds{
		MaxConcurrency:        DefaultMaxConcurrency,
		RequestTimeoutSeconds: DefaultRequestTimeoutSeconds,
		MaxResponseBytes:      DefaultMaxResponseBytes,
		MaxItems:              DefaultMaxItems,
	}}
	_, err = toml.Decode(string(data), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.Instapaper.Username == "" || config.Instapaper.Password == "" {
		return nil, fmt.Errorf("Instapaper username and password are required")
	}
	if len(config.Feeds.URLs) == 0 {
		return nil, fmt.Errorf("at least one feed URL is required")
	}
	if config.Feeds.MaxConcurrency < 1 {
		return nil, fmt.Errorf("feeds.max_concurrency must be at least 1")
	}
	if config.Feeds.RequestTimeoutSeconds < 1 {
		return nil, fmt.Errorf("feeds.request_timeout_seconds must be at least 1")
	}
	if config.Feeds.MaxResponseBytes < 1 {
		return nil, fmt.Errorf("feeds.max_response_bytes must be at least 1")
	}
	if config.Feeds.MaxItems < 1 {
		return nil, fmt.Errorf("feeds.max_items must be at least 1")
	}
	return &config, nil
}
