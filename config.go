package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "."+appName), nil
}

func loadConfig(configDir string) (*Config, error) {
	configPath := filepath.Join(configDir, configFile)

	// Default configuration
	defaultConfig := &Config{
		ExcludePatterns: []string{
			"^ls.*",
			"^cd.*",
			"^pwd.*",
			"^clear.*",
			"^exit.*",
			"^history.*",
			".*password.*",
			".*secret.*",
			".*token.*",
			".*key.*",
			".*" + appName + ".*",
		},
		DatabasePath: filepath.Join(configDir, dbFile),
	}

	// Try to load existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config
			return saveConfig(configPath, defaultConfig)
		}
		return nil, err
	}

	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Ensure database path is set
	if config.DatabasePath == "" {
		config.DatabasePath = filepath.Join(configDir, dbFile)
	}

	return config, nil
}

func saveConfig(configPath string, config *Config) (*Config, error) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return nil, err
	}

	return config, nil
}
