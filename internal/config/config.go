package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
)

// Config represents the global configuration structure.
type Config struct {
	DefaultProject string `json:"default_project,omitempty"`
	DefaultFormat  string `json:"default_format,omitempty"`
}

const (
	// DefaultFormatModern is the default modern format.
	DefaultFormatModern = "modern"
	// DefaultFormatJSON is the JSON format.
	DefaultFormatJSON = "json"
	// DefaultFormatLSON is the L-SON format.
	DefaultFormatLSON = "lson"

	// ConfigFileName is the name of the config file.
	ConfigFileName = "config.json"
)

// Load loads the configuration from disk.
// Returns a default config if the file doesn't exist.
func Load() (*Config, error) {
	configPath, err := storage.ConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("config: failed to resolve config path: %w", err)
	}

	// If config doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return Default(), nil
	}

	var cfg Config
	if err := storage.ReadJSON(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("config: failed to load config: %w", err)
	}

	// Validate loaded config
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("config: invalid config: %w", err)
	}

	return &cfg, nil
}

// Save saves the configuration to disk using atomic write.
func Save(cfg *Config) error {
	// Validate before saving
	if err := Validate(cfg); err != nil {
		return fmt.Errorf("config: invalid config: %w", err)
	}

	configPath, err := storage.ConfigFilePath()
	if err != nil {
		return fmt.Errorf("config: failed to resolve config path: %w", err)
	}

	// Ensure config directory exists
	if err := storage.EnsureDir(configPath); err != nil {
		return fmt.Errorf("config: failed to create config directory: %w", err)
	}

	// Marshal JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config: failed to marshal JSON: %w", err)
	}

	// Use atomic write (config files don't need project-level locking)
	if err := storage.WriteAtomic(configPath, data); err != nil {
		return fmt.Errorf("config: failed to save config: %w", err)
	}

	return nil
}

// Get gets the configuration (creates default if doesn't exist).
func Get() (*Config, error) {
	return Load()
}

// Set sets a configuration value.
func Set(key, value string) error {
	cfg, err := Get()
	if err != nil {
		return fmt.Errorf("config: failed to load config: %w", err)
	}

	switch key {
	case "default_project":
		if value != "" && !isValidProjectKey(value) {
			return fmt.Errorf("config: invalid project key %q (must be uppercase alphanumeric or hyphen)", value)
		}
		cfg.DefaultProject = value
	case "default_format":
		if value != "" && !isValidFormat(value) {
			return fmt.Errorf("config: invalid format %q (must be modern, json, or lson)", value)
		}
		cfg.DefaultFormat = value
	default:
		return fmt.Errorf("config: unknown config key %q", key)
	}

	return Save(cfg)
}

// GetValue gets a configuration value.
func GetValue(key string) (string, error) {
	cfg, err := Get()
	if err != nil {
		return "", fmt.Errorf("config: failed to load config: %w", err)
	}

	switch key {
	case "default_project":
		return cfg.DefaultProject, nil
	case "default_format":
		return cfg.DefaultFormat, nil
	default:
		return "", fmt.Errorf("config: unknown config key %q", key)
	}
}

// isValidFormat validates that the format is one of the allowed values.
func isValidFormat(format string) bool {
	return format == DefaultFormatModern ||
		format == DefaultFormatJSON ||
		format == DefaultFormatLSON
}

// isValidProjectKey validates that the project key is uppercase alphanumeric or hyphen.
var projectKeyRegex = regexp.MustCompile(`^[A-Z0-9-]+$`)

func isValidProjectKey(key string) bool {
	return projectKeyRegex.MatchString(key)
}

// Validate validates the entire config struct.
func Validate(cfg *Config) error {
	if cfg.DefaultFormat != "" && !isValidFormat(cfg.DefaultFormat) {
		return fmt.Errorf("config: invalid default_format %q", cfg.DefaultFormat)
	}

	// Project key validation (uppercase alphanumeric or hyphen)
	if cfg.DefaultProject != "" {
		if !isValidProjectKey(cfg.DefaultProject) {
			return fmt.Errorf("config: invalid project key %q (must be uppercase alphanumeric or hyphen)", cfg.DefaultProject)
		}
	}

	return nil
}
