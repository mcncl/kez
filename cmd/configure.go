package cmd

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/mcncl/kez/internal/config" // Import the config package
)

type ConfigureCmd struct {
	Force bool `kong:"help='Overwrite existing configuration.', short='f'"`
}

func (c *ConfigureCmd) Run(ctx *kong.Context) error {
	fmt.Println("Configuring Buildkite settings...")

	// Load existing or default configuration
	cfg, err := config.Load()
	if err != nil {
		// If loading fails significantly (e.g., bad JSON), stop configuration.
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Prompt for Buildkite Organisation Slug
	orgSlugPrompt := fmt.Sprintf("Enter Buildkite Organisation Slug [%s]: ", cfg.Buildkite.OrgSlug)
	orgSlug, err := config.PromptForInput(orgSlugPrompt)
	if err != nil {
		return err
	}
	// Only update if the user provided input
	if orgSlug != "" {
		cfg.Buildkite.OrgSlug = orgSlug
	} else if cfg.Buildkite.OrgSlug == "" {
		// If no input and no existing value, it's an error
		return fmt.Errorf("buildkite organisation slug cannot be empty")
	}

	// Prompt for Buildkite API Token
	// Don't show the existing token in the prompt for security
	tokenPrompt := "Enter Buildkite API Token (will not be shown): "
	// Use the secure password input function to mask the token input
	token, err := config.PromptForPassword(tokenPrompt)
	if err != nil {
		return err
	}
	// Only update if the user provided input
	if token != "" {
		cfg.Buildkite.Token = token
	} else if cfg.Buildkite.Token == "" {
		// If no input and no existing value, it's an error
		// We might allow empty token if they *only* want to set the org,
		// but typically configure implies setting both. Let's enforce it.
		return fmt.Errorf("buildkite API token cannot be empty")
	}

	// Save the updated configuration
	err = config.Save(cfg)
	if err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("Configuration saved successfully.")
	return nil
}
