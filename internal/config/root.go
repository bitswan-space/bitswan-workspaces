package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"path/filepath"
)

type Config struct {
	ActiveWorkspace string `toml:"active_workspace"`
}

// ConfigPath returns the hardcoded path to the configuration file.
func ConfigPath() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "bitswan", "config.toml")
}

// GetConfig reads the configuration file and returns a Config object.
func GetConfig() (*Config, error) {
	configPath := ConfigPath()
	var config Config

	// Check if the config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return an empty Config if the file doesn't exist
		return &Config{}, nil
	}

	// Open and decode the config file
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	if _, err := toml.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &config, nil
}

// Save writes the Config object to the configuration file.
func (c *Config) Save() error {
	configPath := ConfigPath()

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write the configuration to the file
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to open config file for writing: %w", err)
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(c); err != nil {
		return fmt.Errorf("failed to write to config file: %w", err)
	}

	return nil
}

// GetWorkspaceName retrieves the active workspace name from the configuration file.
func GetWorkspaceName() (string, error) {
	config, err := GetConfig()
	if err != nil {
		return "", err
	}

	return config.ActiveWorkspace, nil
}
