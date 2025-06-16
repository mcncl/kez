package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Helper function to override the config file path for testing
func overrideConfigPath(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "kez", "test_config.json")

	originalConfigFilePath := configFilePath

	configFilePath = func() (string, error) {
		return tempFile, nil
	}

	t.Cleanup(func() {
		configFilePath = originalConfigFilePath
	})

	return tempFile
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Buildkite.Token != "" {
		t.Errorf("Expected default Buildkite token to be empty, got '%s'", cfg.Buildkite.Token)
	}
	if cfg.Buildkite.OrgSlug != "" {
		t.Errorf("Expected default Buildkite org slug to be empty, got '%s'", cfg.Buildkite.OrgSlug)
	}
	if cfg.Kubernetes.PreferredProvider != "orbstack" {
		t.Errorf("Expected default K8s provider to be 'orbstack', got '%s'", cfg.Kubernetes.PreferredProvider)
	}
	if len(cfg.RecentClusters) != 0 {
		t.Errorf("Expected default recent clusters to be empty, got %d items", len(cfg.RecentClusters))
	}
}

func TestSaveLoadCycle(t *testing.T) {
	tempConfigFile := overrideConfigPath(t)
	configDir := filepath.Dir(tempConfigFile)

	_ = os.RemoveAll(configDir)

	expectedConfig := &Config{
		Buildkite: BuildkiteConfig{
			Token:   "test-token-123",
			OrgSlug: "test-org-456",
		},
		Kubernetes: KubernetesConfig{
			PreferredProvider: "kind",
		},
		RecentClusters: []RecentCluster{
			{UUID: "uuid-abc", Name: "ClusterABC", OrgSlug: "org-abc"},
			{UUID: "uuid-xyz", Name: "ClusterXYZ", OrgSlug: "org-xyz"},
		},
	}

	err := Save(expectedConfig)
	if err != nil {
		t.Fatalf("Save() failed unexpectedly: %v", err)
	}

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Fatalf("Save() did not create the config directory '%s'", configDir)
	}
	if _, err := os.Stat(tempConfigFile); os.IsNotExist(err) {
		t.Fatalf("Save() did not create the config file '%s'", tempConfigFile)
	}

	loadedConfig, err := Load()
	if err != nil {
		t.Fatalf("Load() failed unexpectedly after Save(): %v", err)
	}

	if !reflect.DeepEqual(expectedConfig, loadedConfig) {
		t.Errorf("Loaded config does not match saved config.\nExpected: %+v\nGot:      %+v", expectedConfig, loadedConfig)
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	_ = overrideConfigPath(t)

	loadedConfig, err := Load()
	if err != nil {
		// Load should *not* error if the file simply doesn't exist
		t.Fatalf("Load() failed unexpectedly for a non-existent file: %v", err)
	}

	defaultConfig := DefaultConfig()
	if !reflect.DeepEqual(defaultConfig, loadedConfig) {
		t.Errorf("Load() from non-existent file did not return default config.\nExpected: %+v\nGot:      %+v", defaultConfig, loadedConfig)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	tempConfigFile := overrideConfigPath(t)
	configDir := filepath.Dir(tempConfigFile)

	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatalf("Failed to create temp config directory '%s': %v", configDir, err)
	}
	file, err := os.Create(tempConfigFile)
	if err != nil {
		t.Fatalf("Failed to create empty temp config file '%s': %v", tempConfigFile, err)
	}
	err = file.Close()
	if err != nil {
		t.Fatalf("Failed to close temp config file '%s': %v", tempConfigFile, err)
	}

	loadedConfig, err := Load()
	if err != nil {
		// Load should *not* error for an empty file
		t.Fatalf("Load() failed unexpectedly for an empty file: %v", err)
	}

	// Check if it returned the default configuration
	defaultConfig := DefaultConfig()
	if !reflect.DeepEqual(defaultConfig, loadedConfig) {
		t.Errorf("Load() from empty file did not return default config.\nExpected: %+v\nGot:      %+v", defaultConfig, loadedConfig)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tempConfigFile := overrideConfigPath(t)
	configDir := filepath.Dir(tempConfigFile)

	// Manually create the directory
	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatalf("Failed to create temp config directory '%s': %v", configDir, err)
	}

	invalidJSON := []byte(`{"buildkite": {"token": "abc", org_slug: "def"}}`)
	err := os.WriteFile(tempConfigFile, invalidJSON, 0600)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON to temp file '%s': %v", tempConfigFile, err)
	}

	_, err = Load()
	if err == nil {
		// Load *should* error if the JSON is invalid
		t.Errorf("Load() should have failed for invalid JSON, but it succeeded.")
	}
}
