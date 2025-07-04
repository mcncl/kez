package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	
	"golang.org/x/term"
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
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	OrgSlug  string `json:"org_slug"`
	TokenID  string `json:"token_id,omitempty"`  // ID of the cluster token
	TokenVal string `json:"token_val,omitempty"` // Value of the token (for reference only)
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
// Returns the path to the configuration file at ~/.config/kez/config.json.
var configFilePath = func() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	// Using ~/.config/kez for config storage
	return filepath.Join(homeDir, ".config", "kez", "config.json"), nil
}

// Load reads the configuration from ~/.config/kez.
// If the file doesn't exist, it creates the directory and returns a default configuration.
func Load() (*Config, error) {
	path, err := configFilePath()
	if err != nil {
		return nil, err
	}

	// Ensure the config directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create config directory %s: %w", dir, err)
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

// Save writes the configuration to the file system in ~/.config/kez.
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

// PromptForPassword prompts the user for a password input with masking.
// The input will not be displayed on the screen as it's being typed.
func PromptForPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	
	// Read password without echoing to the terminal
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	
	// Print a newline since ReadPassword doesn't do it
	fmt.Println()
	
	return string(passwordBytes), nil
}
