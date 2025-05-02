package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the application's configuration.
type Config struct {
	Buildkite      BuildkiteConfig  `json:"buildkite"`
	Kubernetes     KubernetesConfig `json:"kubernetes"`
	RecentClusters []RecentCluster  `json:"recent_clusters"`
}

// BuildkiteConfig holds Buildkite specific settings.
type BuildkiteConfig struct {
	Token   string `json:"token"`
	OrgSlug string `json:"org_slug"`
}

// KubernetesConfig holds Kubernetes specific settings.
type KubernetesConfig struct {
	PreferredProvider string `json:"preferred_provider"`
}

// RecentCluster holds information about a recently used cluster.
type RecentCluster struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	OrgSlug string `json:"org_slug"`
}

// Default values for a new configuration.
func DefaultConfig() *Config {
	return &Config{
		Buildkite: BuildkiteConfig{
			Token:   "", // Needs to be set by user
			OrgSlug: "", // Needs to be set by user
		},
		Kubernetes: KubernetesConfig{
			PreferredProvider: "orbstack", // Default preference from plan
		},
		RecentClusters: []RecentCluster{},
	}
}

// configFilePath points to the function used to get the config file path.
// It's a variable to allow overriding during tests.
var configFilePath = func() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	// Using "kuberneasy" as the directory name within .config
	return filepath.Join(configDir, "kuberneasy", "config.json"), nil
}

// Load reads the configuration from the file system.
// If the file doesn't exist, it returns a default configuration.
func Load() (*Config, error) {
	path, err := configFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return default config and don't treat as error
			fmt.Printf("Config file not found at %s, using defaults.\n", path)
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var cfg Config
	// Handle empty file case
	if len(data) == 0 {
		fmt.Printf("Config file at %s is empty, using defaults.\n", path)
		return DefaultConfig(), nil
	}

	err = json.Unmarshal(data, &cfg)
	if err != nil {
		// If unmarshalling fails, perhaps the format is invalid.
		// Consider warning the user and returning defaults or erroring out.
		// For now, let's error out.
		return nil, fmt.Errorf("failed to parse config file %s (invalid JSON?): %w", path, err)
	}
	fmt.Printf("Configuration loaded from %s\n", path)
	return &cfg, nil
}

// Save writes the configuration to the file system.
// It creates the necessary directories if they don't exist.
func Save(cfg *Config) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}

	// Ensure the directory exists
	dir := filepath.Dir(path)
	// Use 0750 for directory permissions (owner rwx, group rx, others ---)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Use 0600 for file permissions (owner rw, group ---, others ---)
	err = os.WriteFile(path, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write config file %s: %w", path, err)
	}
	fmt.Printf("Configuration saved to %s\n", path)
	return nil
}

// PromptForInput prompts the user for input with a given message.
func PromptForInput(prompt string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(input), nil
}
