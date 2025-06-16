package stack

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/alecthomas/kong"
	"github.com/buildkite/go-buildkite/v4"
	"github.com/mcncl/kez/internal/api"
	"github.com/mcncl/kez/internal/github"
	"github.com/mcncl/kez/internal/k8s"
	"github.com/mcncl/kez/internal/logger"
)

// CreateCmd represents the 'stack create' command
type CreateCmd struct {
	Version string `help:"Specify a version of agent-stack-k8s to use (defaults to interactive selection)"`
	Name    string `help:"Specify a name for the stack (default: agent-stack-k8s)" short:"n"`
	Quiet   bool   `help:"Suppress non-essential output" short:"q"`
}

// ClusterOption represents a selectable cluster option in the UI
type ClusterOption struct {
	Name     string
	UUID     string
	IsRecent bool
	Original buildkite.Cluster
}

// Run executes the stack create command
func (c *CreateCmd) Run(ctx *kong.Context) error {
	// Set up output configuration based on quiet flag
	var output OutputConfig
	if c.Quiet {
		output = NewQuietOutput()
	} else {
		output = DefaultOutput()
	}

	if !output.QuietMode {
		logger.Info("Creating Buildkite agent stack in Kubernetes")
	}

	// Initialize API client
	client, err := api.NewClient()
	if err != nil {
		logger.Error("Failed to initialize API client", "error", err)
		return fmt.Errorf("failed to initialize API client: %w", err)
	}

	// Initialize the release name based on the flag or get it interactively
	releaseName := "agent-stack-k8s"
	if c.Name != "" {
		releaseName = c.Name
	} else {
		// Prompt the user for a stack name
		namePrompt := &survey.Input{
			Message: "Enter a name for the stack:",
			Default: "agent-stack-k8s",
		}

		err := survey.AskOne(namePrompt, &releaseName)
		if err != nil {
			logger.Error("Stack name input was cancelled", "error", err)
			return fmt.Errorf("stack name input was cancelled: %w", err)
		}
	}

	// Get clusters from Buildkite
	clusters, err := client.ListClusters(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	if len(clusters) == 0 {
		return fmt.Errorf("no clusters found in your Buildkite organization. Please create a cluster first")
	}

	// Get recent clusters
	recentClusters := client.GetRecentClusters()

	// Create cluster options for selection
	var clusterOptions []ClusterOption

	// Add recent clusters with a special prefix
	recentMap := make(map[string]bool)
	for _, recent := range recentClusters {
		recentMap[recent.UUID] = true

		// Find the full cluster info
		for _, cluster := range clusters {
			if cluster.ID == recent.UUID {
				clusterOptions = append(clusterOptions, ClusterOption{
					Name:     fmt.Sprintf("🔄 %s", cluster.Name),
					UUID:     cluster.ID,
					IsRecent: true,
					Original: cluster,
				})
				break
			}
		}
	}

	// Add a visual separator if we have recent clusters
	if len(recentClusters) > 0 {
		clusterOptions = append(clusterOptions, ClusterOption{
			Name:     "─────────────────────────",
			UUID:     "",
			IsRecent: false,
		})
	}

	// Add all other clusters
	for _, cluster := range clusters {
		// Skip if already in recent options
		if recentMap[cluster.ID] {
			continue
		}

		clusterOptions = append(clusterOptions, ClusterOption{
			Name:     cluster.Name,
			UUID:     cluster.ID,
			IsRecent: false,
			Original: cluster,
		})
	}

	// Create list of selectable options (excluding separators)
	var selectableOptions []ClusterOption
	var optionNames []string

	for _, opt := range clusterOptions {
		if opt.UUID != "" { // Only include actual clusters, not separators
			selectableOptions = append(selectableOptions, opt)
			optionNames = append(optionNames, opt.Name)
		}
	}

	// Prompt for cluster selection
	var selectedOptionIndex int
	prompt := &survey.Select{
		Message:  "Select a cluster:",
		Options:  optionNames,
		PageSize: 15,
	}

	err = survey.AskOne(prompt, &selectedOptionIndex)
	if err != nil {
		return fmt.Errorf("cluster selection was cancelled: %w", err)
	}

	// Get the selected cluster
	selectedCluster := selectableOptions[selectedOptionIndex].Original

	printClusterSelected(selectedCluster.Name, selectedCluster.ID, output)

	// Store the selected cluster in recent clusters
	if err := client.AddRecentCluster(selectedCluster); err != nil && !output.QuietMode {
		fmt.Fprintf(output.Writer, "Warning: Failed to save cluster to recent list: %v\n", err)
	}

	// Determine the version to use
	version := c.Version
	if version == "" {
		// Fetch available versions from GitHub if not specified
		if !output.QuietMode {
			fmt.Fprintln(output.Writer, "\n🔍 Fetching available agent-stack-k8s versions...")
		}
		releases, err := github.GetAgentStackReleases()
		if err != nil {
			if !output.QuietMode {
				fmt.Fprintf(output.Writer, "⚠️  Warning: Failed to fetch releases: %v\n", err)
				fmt.Fprintln(output.Writer, "Using the default version instead.")
			}
			version = "0.28.0-beta2" // Default fallback version
		} else {
			// Prepare options for selection
			var optionNames []string
			for _, release := range releases {
				optionNames = append(optionNames, github.FormatReleaseOption(release))
			}

			// Prompt for version selection
			var selectedVersionIndex int
			prompt := &survey.Select{
				Message:  "Select agent-stack-k8s version:",
				Options:  optionNames,
				PageSize: 15,
			}

			err = survey.AskOne(prompt, &selectedVersionIndex)
			if err != nil {
				return fmt.Errorf("version selection was cancelled: %w", err)
			}

			// Get the selected version
			selectedRelease := releases[selectedVersionIndex]
			version = github.GetChartVersion(selectedRelease.TagName)
			printVersionSelected(version, output)
		}
	} else {
		// Ensure user-provided version doesn't have 'v' prefix
		if strings.HasPrefix(version, "v") {
			version = version[1:]
		}
		printVersionSpecified(version, output)
	}

	// Prompt for agent token
	var agentToken string
	tokenPrompt := &survey.Password{
		Message: "Enter Buildkite agent token (press Enter to create a new token):",
	}

	// Custom validator that allows empty strings
	validator := func(val interface{}) error {
		// Allow empty strings (will create a token) and non-empty strings
		return nil
	}

	err = survey.AskOne(tokenPrompt, &agentToken, survey.WithValidator(validator))
	if err != nil {
		return fmt.Errorf("token input was cancelled: %w", err)
	}

	// If token is empty, create a new one
	if agentToken == "" {
		if !output.QuietMode {
			fmt.Fprintln(output.Writer, "\n🔑 Creating a new agent token...")
		}

		// Default token description
		defaultDescription := "kez-" + version

		// Prompt for token description (default or custom)
		var tokenDescription string
		descPrompt := &survey.Input{
			Message: "Enter token description (press Enter for default):",
			Default: defaultDescription,
		}

		err = survey.AskOne(descPrompt, &tokenDescription)
		if err != nil {
			return fmt.Errorf("token description input was cancelled: %w", err)
		}

		// Create the token with the chosen description
		ctx := context.Background()
		tokenObj, err := client.CreateTokenWithDescription(ctx, selectedCluster.ID, tokenDescription)
		if err != nil {
			return fmt.Errorf("failed to create token: %w", err)
		}
		agentToken = tokenObj.Token
		printTokenCreated(tokenDescription, tokenObj.ID, output)
	}

	// Ask about SSH keys for git checkout actions
	var useSSHKeys bool
	sshPrompt := &survey.Confirm{
		Message: "Configure SSH credentials for git checkout actions?",
		Default: true,
	}

	err = survey.AskOne(sshPrompt, &useSSHKeys)
	if err != nil {
		return fmt.Errorf("SSH configuration was cancelled: %w", err)
	}

	var secretName string
	if useSSHKeys {
		// Determine user's SSH directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to determine user home directory: %w", err)
		}
		sshDir := filepath.Join(homeDir, ".ssh")

		// Check if the SSH directory exists
		if _, err := os.Stat(sshDir); os.IsNotExist(err) {
			if !output.QuietMode {
				fmt.Fprintf(output.Writer, "⚠️ SSH directory not found at %s\n", sshDir)
			}

			// Ask if they want to generate a new key
			var generateKey bool
			generatePrompt := &survey.Confirm{
				Message: "SSH directory not found. Generate a new SSH key?",
				Default: true,
			}

			err = survey.AskOne(generatePrompt, &generateKey)
			if err != nil {
				return fmt.Errorf("key generation choice was cancelled: %w", err)
			}

			if generateKey {
				// Generate a new key
				if err := generateSSHKey(sshDir, output); err != nil {
					return fmt.Errorf("failed to generate SSH key: %w", err)
				}
			} else {
				if !output.QuietMode {
					fmt.Fprintln(output.Writer, "⚠️ Continuing without SSH keys. Checkout actions may not work properly.")
				}
				useSSHKeys = false
			}
		}

		if useSSHKeys {
			// List SSH keys
			keyFiles, err := listSSHKeys(sshDir)
			if err != nil {
				return fmt.Errorf("failed to list SSH keys: %w", err)
			}

			if len(keyFiles) == 0 {
				if !output.QuietMode {
					fmt.Fprintln(output.Writer, "⚠️ No SSH keys found in your .ssh directory.")
				}
				useSSHKeys = false
			} else {
				// Format key options to show just the filename
				keyOptions := make([]string, len(keyFiles))
				for i, k := range keyFiles {
					keyOptions[i] = filepath.Base(k)
				}

				// Let user select a specific key
				var selectedKey string
				keyPrompt := &survey.Select{
					Message: "Select an SSH key to use:",
					Options: keyOptions,
				}

				err = survey.AskOne(keyPrompt, &selectedKey)
				if err != nil {
					return fmt.Errorf("key selection was cancelled: %w", err)
				}

				// Find the full path of the selected key
				var selectedKeyPath string
				for i, k := range keyOptions {
					if k == selectedKey {
						selectedKeyPath = keyFiles[i]
						break
					}
				}

				// Create a secret name based on the release name
				secretName = fmt.Sprintf("git-ssh-key-%s", releaseName)

				if !output.QuietMode {
					fmt.Fprintf(output.Writer, "🔑 Creating Kubernetes secret '%s' with SSH key...\n", secretName)
				}

				// Ensure the buildkite namespace exists
				_, err := k8s.EnsureNamespaceExists("buildkite")
				if err != nil {
					return fmt.Errorf("failed to create namespace: %w", err)
				}

				// Create the Kubernetes secret with kubectl
				createSecretCmd := exec.Command(
					"kubectl", "create", "secret", "generic", secretName,
					"--from-file=SSH_PRIVATE_RSA_KEY="+selectedKeyPath,
					"-n", "buildkite",
					"--dry-run=client",
					"-o", "yaml",
				)

				// Pipe the output to kubectl apply
				applyCmd := exec.Command("kubectl", "apply", "-f", "-")

				// Connect the commands
				pipe, err := createSecretCmd.StdoutPipe()
				if err != nil {
					return fmt.Errorf("failed to create pipe for kubectl commands: %w", err)
				}
				applyCmd.Stdin = pipe

				// Set stderr for both commands
				if output.QuietMode {
					// In quiet mode, suppress command output
					createSecretCmd.Stderr = nil
					applyCmd.Stderr = nil
					applyCmd.Stdout = nil
				} else {
					createSecretCmd.Stderr = output.Writer
					applyCmd.Stderr = output.Writer
					applyCmd.Stdout = output.Writer
				}

				// Run the commands
				if err := createSecretCmd.Start(); err != nil {
					return fmt.Errorf("failed to start kubectl create secret command: %w", err)
				}

				if err := applyCmd.Start(); err != nil {
					return fmt.Errorf("failed to start kubectl apply command: %w", err)
				}

				if err := createSecretCmd.Wait(); err != nil {
					return fmt.Errorf("kubectl create secret command failed: %w", err)
				}

				if err := applyCmd.Wait(); err != nil {
					return fmt.Errorf("kubectl apply command failed: %w", err)
				}

				printSSHKeySecretCreated(output)
			}
		}
	}

	// Release name has already been set above, no need to reset it here

	// Confirm installation
	var proceed bool
	confirmPrompt := &survey.Confirm{
		Message: fmt.Sprintf("Ready to install stack '%s' for cluster '%s'. Proceed?", releaseName, selectedCluster.Name),
		Default: true,
	}

	err = survey.AskOne(confirmPrompt, &proceed)
	if err != nil {
		return fmt.Errorf("confirmation was cancelled: %w", err)
	}

	if !proceed {
		if !output.QuietMode {
			fmt.Fprintln(output.Writer, "Installation cancelled.")
		}
		return nil
	}

	// Run Helm command
	if !output.QuietMode {
		fmt.Fprintln(output.Writer, "\n🚀 Installing agent stack...")
	}
	orgSlug := client.GetOrgSlug()

	// Prepare Helm options for installation
	helmOpts := k8s.HelmInstallOptions{
		ReleaseName:     releaseName,
		ChartReference:  fmt.Sprintf("oci://ghcr.io/buildkite/helm/agent-stack-k8s:%s", version),
		Namespace:       "buildkite",
		CreateNamespace: true,
		Values: map[string]string{
			"agentToken":          agentToken,
			"config.org":          orgSlug,
			"config.cluster-uuid": selectedCluster.ID,
		},
		JSONValues: map[string]string{
			"config.tags": "[\"queue=kubernetes\"]",
		},
	}

	// Install using the k8s package
	if err := k8s.InstallWithHelm(helmOpts); err != nil {
		return fmt.Errorf("helm installation failed: %w", err)
	}

	printAgentStackInstalled(releaseName, selectedCluster.Name, selectedCluster.ID, orgSlug, version, output)

	// Display SSH key usage instructions if we created a secret
	if secretName != "" && !output.QuietMode {
		fmt.Fprintln(output.Writer, "\n📝 Using SSH keys in your pipelines:")
		fmt.Fprintln(output.Writer, "To use the SSH key in your pipelines, add the following to your pipeline.yaml:")
		fmt.Fprintln(output.Writer, "```yaml")
		fmt.Fprintln(output.Writer, "  plugins:")
		fmt.Fprintln(output.Writer, "    - kubernetes:")
		fmt.Fprintln(output.Writer, "        gitEnvFrom:")
		fmt.Fprintln(output.Writer, "        - secretRef:")
		fmt.Fprintf(output.Writer, "            name: %s\n", secretName)
		fmt.Fprintln(output.Writer, "```")
	}

	return nil
}

// listSSHKeys identifies SSH private keys in the specified directory
func listSSHKeys(sshDir string) ([]string, error) {
	var keyFiles []string

	files, err := os.ReadDir(sshDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH directory: %w", err)
	}

	// Common private key files and patterns to look for
	commonKeys := []string{
		"id_rsa", "id_dsa", "id_ecdsa", "id_ed25519",
		"identity", "github_rsa",
	}

	// Look for common key files first
	for _, keyName := range commonKeys {
		path := filepath.Join(sshDir, keyName)
		if _, err := os.Stat(path); err == nil {
			keyFiles = append(keyFiles, path)
		}
	}

	// If no common keys found, look for files that might be keys
	if len(keyFiles) == 0 {
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			name := file.Name()
			// Skip public keys, config files, and known_hosts
			if strings.HasSuffix(name, ".pub") ||
				name == "config" ||
				name == "known_hosts" ||
				name == "authorized_keys" {
				continue
			}

			// Check file permissions to identify potential key files
			path := filepath.Join(sshDir, name)
			info, err := os.Stat(path)
			if err != nil {
				continue
			}

			// Check if permissions are restricted (private keys shouldn't be readable by others)
			mode := info.Mode().Perm()
			if mode&0077 == 0 { // No permissions for group/others
				keyFiles = append(keyFiles, path)
			}
		}
	}

	return keyFiles, nil
}

// generateSSHKey creates a new SSH key pair in the specified directory
func generateSSHKey(sshDir string, output OutputConfig) error {
	// Create the .ssh directory if it doesn't exist
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	keyPath := filepath.Join(sshDir, "id_rsa")

	// Generate the key using ssh-keygen
	cmd := exec.Command("ssh-keygen",
		"-t", "rsa",
		"-b", "4096",
		"-f", keyPath,
		"-N", "", // Empty passphrase
		"-C", fmt.Sprintf("buildkite-agent-%s", time.Now().Format("20060102")),
	)

	if output.QuietMode {
		// In quiet mode, suppress command output
		cmd.Stdout = nil
		cmd.Stderr = nil
	} else {
		cmd.Stdout = output.Writer
		cmd.Stderr = output.Writer
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate SSH key: %w", err)
	}

	// Print the public key and instructions
	pubKeyPath := keyPath + ".pub"
	pubKey, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	printSSHKeyGenerated(output)

	if !output.QuietMode {
		fmt.Fprintln(output.Writer, "\nPublic key (add this to your GitHub/GitLab account):")
		fmt.Fprintf(output.Writer, "\n%s\n", pubKey)
		fmt.Fprintln(output.Writer, "\nInstructions:")
		fmt.Fprintln(output.Writer, "1. Copy the public key above")
		fmt.Fprintln(output.Writer, "2. Add it to your GitHub/GitLab account in the SSH keys section")
		fmt.Fprintln(output.Writer, "3. Test with: ssh -T git@github.com")
	}

	return nil
}
