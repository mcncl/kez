package stack

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/alecthomas/kong"
	"github.com/mcncl/kez/internal/api"
	"github.com/mcncl/kez/internal/config"
	"github.com/mcncl/kez/internal/k8s"
)

// DeleteCmd represents the 'stack delete' command
type DeleteCmd struct {
	Force      bool   `help:"Skip confirmation prompts" short:"f"`
	Timeout    int    `help:"Timeout in seconds for delete operations" default:"60"`
	Name       string `help:"Specify the stack name to delete" short:"n"`
	All        bool   `help:"Delete all Buildkite agent stacks in the cluster" short:"a"`
	NoWait     bool   `help:"Skip waiting for pod termination" short:"w"`
}

// Run executes the stack delete command
func (c *DeleteCmd) Run(ctx *kong.Context) error {
	fmt.Println("Deleting Buildkite agent stack from Kubernetes...")

	// Check if the buildkite namespace exists
	stackInstalled, err := k8s.IsAgentStackInstalled()
	if err != nil {
		return fmt.Errorf("failed to check if agent stack is installed: %w", err)
	}

	if !stackInstalled {
		fmt.Println("❌ No Buildkite agent stack is installed. Nothing to delete.")
		return nil
	}

	// Check for helm and get available stacks
	helmPath, err := exec.LookPath("helm")
	if err != nil {
		fmt.Println("⚠️ Helm not found in PATH. Will only remove Kubernetes resources directly.")
		// If helm isn't available and no name specified, we can't proceed
		if c.Name == "" && !c.All {
			return fmt.Errorf("helm not available and no stack name specified. Use --name to specify the stack name")
		}
	} else {
		// List installed stacks using helm
		fmt.Println("🔍 Checking for installed Buildkite agent stacks...")
		
		listCmd := exec.Command(helmPath, "list", "-n", "buildkite", "-o", "json")
		listOutput, err := listCmd.CombinedOutput()
		
		if err != nil {
			fmt.Printf("⚠️ Failed to list Helm releases: %s\n", err)
			if c.Name == "" && !c.All {
				return fmt.Errorf("failed to list helm releases and no stack name specified")
			}
		} else if len(listOutput) > 0 && strings.TrimSpace(string(listOutput)) != "[]" {
			// Extract stack names from JSON output
			stackList := []string{}
			for _, line := range strings.Split(string(listOutput), "\n") {
				if strings.Contains(line, "\"name\":") {
					parts := strings.Split(line, "\"")
					if len(parts) > 3 {
						stackList = append(stackList, parts[3])
					}
				}
			}
			
			if len(stackList) == 0 {
				fmt.Println("❌ No Buildkite agent stacks found in the buildkite namespace.")
				return nil
			}
			
			// If no name specified and not deleting all, prompt user to select
			if c.Name == "" && !c.All {
				if len(stackList) == 1 {
					// Only one stack, use it
					c.Name = stackList[0]
					fmt.Printf("ℹ️ Found one stack: %s\n", c.Name)
				} else if !c.Force {
					// Multiple stacks, prompt user to select
					fmt.Printf("Found %d Buildkite agent stacks:\n", len(stackList))
					listCmd = exec.Command(helmPath, "list", "-n", "buildkite")
					listCmd.Stdout = os.Stdout
					listCmd.Stderr = os.Stderr
					listCmd.Run()
					
					// Add "Delete all" option to the stack list
					options := append(stackList, "Delete all stacks")
					var selectedOption int
					prompt := &survey.Select{
						Message: "Select stack to delete:",
						Options: options,
					}
					
					if err := survey.AskOne(prompt, &selectedOption); err != nil {
						return fmt.Errorf("selection cancelled: %w", err)
					}
					
					if selectedOption == len(stackList) {
						// User selected "Delete all"
						c.All = true
						c.Name = ""
					} else {
						// User selected a specific stack
						c.Name = stackList[selectedOption]
						c.All = false
					}
				} else {
					// Force mode with multiple stacks but no name specified
					return fmt.Errorf("multiple stacks found but no specific stack name provided. Use --name to specify or --all to delete all")
				}
			} else if c.Name != "" && !c.All {
				// User specified a name, verify it exists
				found := false
				for _, stack := range stackList {
					if stack == c.Name {
						found = true
						break
					}
				}
				if !found {
					fmt.Printf("❌ No stack named '%s' found. Available stacks:\n", c.Name)
					listCmd = exec.Command(helmPath, "list", "-n", "buildkite")
					listCmd.Stdout = os.Stdout
					listCmd.Stderr = os.Stderr
					listCmd.Run()
					return fmt.Errorf("specified stack not found")
				}
			}
		} else {
			fmt.Println("❌ No Buildkite agent stacks found in the buildkite namespace.")
			return nil
		}
	}

	// Initialize API client (for recent clusters)
	client, err := api.NewClient()
	if err != nil {
		fmt.Println("⚠️ Failed to initialize API client. Limited operation details will be available.")
	}

	// Check Kubernetes connection
	fmt.Println("🔍 Checking Kubernetes connection...")
	err = k8s.VerifyClusterConnection()
	if err != nil {
		return fmt.Errorf("kubernetes connection check failed: %w", err)
	}

	// Get current context
	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		return fmt.Errorf("kubectl not found in PATH. Is it installed? Error: %w", err)
	}

	contextCmd := exec.Command(kubectlPath, "config", "current-context")
	contextBytes, err := contextCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes context: %w", err)
	}
	currentContext := strings.TrimSpace(string(contextBytes))

	// Detect the K8s provider
	provider, err := k8s.DetectProvider()
	if err != nil {
		fmt.Printf("⚠️ Unable to detect Kubernetes provider: %s\n", err)
		provider = k8s.ProviderUnknown
	}

	if provider == k8s.ProviderUnknown {
		fmt.Printf("✅ Connected to Kubernetes context: %s\n", currentContext)
	} else {
		fmt.Printf("✅ Connected to Kubernetes context: %s (%s)\n", currentContext, provider)
	}

	// Get agent pod status
	runningCount, totalPods, err := k8s.GetAgentPodsStatus()
	if err != nil {
		if !strings.Contains(err.Error(), "buildkite not found") {
			fmt.Printf("⚠️ Unable to get agent pod status: %s\n", err)
		}
	} else {
		if totalPods == 0 {
			fmt.Println("ℹ️ No Buildkite agent pods found")
		} else if runningCount == 0 {
			fmt.Printf("ℹ️ Found %d agent pods but none are running\n", totalPods)
		} else {
			fmt.Printf("ℹ️ Found %d/%d Buildkite agent pods running\n", runningCount, totalPods)
		}
	}

	// Get cluster information from Buildkite if available
	var clusterInfo string
	var clustersToDelete []config.RecentCluster
	
	if client != nil {
		clusterCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		recentClusters := client.GetRecentClusters()
		if len(recentClusters) > 0 {
			if c.All {
				// Collect all clusters to delete tokens from
				for _, cluster := range recentClusters {
					if cluster.TokenID != "" {
						clustersToDelete = append(clustersToDelete, cluster)
					}
				}
			} else {
				// Try to find cluster based on name
				matchedClusters, err := client.FindClusterByName(c.Name)
				if err == nil && len(matchedClusters) > 0 {
					for _, cluster := range matchedClusters {
						if cluster.TokenID != "" {
							clustersToDelete = append(clustersToDelete, cluster)
						}
					}
				}
				
				// If no specific match, use the most recent cluster
				if len(clustersToDelete) == 0 && len(recentClusters) > 0 {
					// As a fallback, look for any clusters with tokens
					for _, cluster := range recentClusters {
						if cluster.TokenID != "" {
							clustersToDelete = append(clustersToDelete, cluster)
							break // Just take the first one with a token
						}
					}
				}
			}
			
			// Set cluster info for the confirmation prompt
			if len(clustersToDelete) > 0 && !c.All {
				firstCluster := clustersToDelete[0]
				clusterInfo = fmt.Sprintf("%s (%s)", firstCluster.Name, firstCluster.UUID)
			} else if len(recentClusters) > 0 {
				mostRecentCluster := recentClusters[len(recentClusters)-1]
				
				// Attempt to list clusters to find details about the recent one
				clusters, err := client.ListClusters(clusterCtx)
				if err == nil {
					for _, cluster := range clusters {
						if cluster.ID == mostRecentCluster.UUID {
							clusterInfo = fmt.Sprintf("%s (%s)", cluster.Name, cluster.ID)
							break
						}
					}
				}
				
				if clusterInfo == "" {
					clusterInfo = fmt.Sprintf("%s (%s)", mostRecentCluster.Name, mostRecentCluster.UUID)
				}
			}
		}
	}

	// Confirm deletion
	if !c.Force {
		var proceed bool
		var message string
		
		if c.All {
			message = "Are you sure you want to delete ALL Buildkite agent stacks?"
		} else {
			message = fmt.Sprintf("Are you sure you want to delete the Buildkite agent stack '%s'?", c.Name)
		}

		if clusterInfo != "" {
			message += fmt.Sprintf(" (Cluster: %s)", clusterInfo)
		}

		prompt := &survey.Confirm{
			Message: message,
			Default: false,
		}

		if err := survey.AskOne(prompt, &proceed); err != nil {
			return fmt.Errorf("prompt cancelled: %w", err)
		}

		if !proceed {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	// Delete the helm release(s) if helm is available
	if helmPath != "" {
		if c.All {
			fmt.Println("🗑️ Uninstalling all Buildkite agent stack Helm releases...")
			
			// List all releases in the buildkite namespace
			listCmd := exec.Command(helmPath, "list", "-n", "buildkite", "--output", "json")
			listOutput, err := listCmd.CombinedOutput()
			if err != nil {
				fmt.Printf("⚠️ Failed to list Helm releases: %s\n", err)
				fmt.Println("Continuing with direct resource deletion...")
			} else {
				// Extract release names (simplified approach)
				releaseNames := []string{}
				
				// In a real implementation, properly parse the JSON
				// This is a simplified string-based approach
				for _, line := range strings.Split(string(listOutput), "\n") {
					if strings.Contains(line, "\"name\":") {
						parts := strings.Split(line, "\"")
						if len(parts) > 3 {
							releaseNames = append(releaseNames, parts[3])
						}
					}
				}
				
				if len(releaseNames) == 0 {
					fmt.Println("⚠️ No Helm releases found to uninstall")
				} else {
					for _, name := range releaseNames {
						fmt.Printf("🗑️ Uninstalling Helm release '%s'...\n", name)
						helmCmd := exec.Command(helmPath, "uninstall", name, "-n", "buildkite")
						helmCmd.Stdout = os.Stdout
						helmCmd.Stderr = os.Stderr
						
						if err := helmCmd.Run(); err != nil {
							fmt.Printf("⚠️ Failed to uninstall Helm release '%s': %s\n", name, err)
						} else {
							fmt.Printf("✅ Helm release '%s' uninstalled successfully\n", name)
						}
					}
				}
			}
		} else {
			// Delete a specific release
			fmt.Printf("🗑️ Uninstalling Helm release '%s'...\n", c.Name)
			helmCmd := exec.Command(helmPath, "uninstall", c.Name, "-n", "buildkite")
			helmCmd.Stdout = os.Stdout
			helmCmd.Stderr = os.Stderr

			if err := helmCmd.Run(); err != nil {
				fmt.Printf("⚠️ Failed to uninstall Helm release '%s': %s\n", c.Name, err)
				fmt.Println("Continuing with direct resource deletion...")
			} else {
				fmt.Printf("✅ Helm release '%s' uninstalled successfully\n", c.Name)
			}
		}
	}

	// Check for any SSH key secrets and delete them
	fmt.Println("🔍 Checking for SSH key secrets...")
	sshSecretCmd := exec.Command(kubectlPath, "get", "secrets", "-n", "buildkite", "--field-selector=type=Opaque", "-o", "custom-columns=NAME:.metadata.name", "--no-headers")
	secretOutput, err := sshSecretCmd.CombinedOutput()
	if err == nil {
		secrets := strings.Split(strings.TrimSpace(string(secretOutput)), "\n")
		var sshSecrets []string

		for _, secret := range secrets {
			if strings.Contains(secret, "git-ssh") || strings.Contains(secret, "ssh-key") {
				sshSecrets = append(sshSecrets, secret)
			}
		}

		if len(sshSecrets) > 0 {
			fmt.Printf("🗑️ Deleting %d SSH key secrets...\n", len(sshSecrets))
			for _, secret := range sshSecrets {
				deleteSecretCmd := exec.Command(kubectlPath, "delete", "secret", secret, "-n", "buildkite")
				if err := deleteSecretCmd.Run(); err != nil {
					fmt.Printf("⚠️ Failed to delete secret %s: %s\n", secret, err)
				} else {
					fmt.Printf("✓ Deleted secret: %s\n", secret)
				}
			}
		} else {
			fmt.Println("ℹ️ No SSH key secrets found")
		}
	}

	// Delete any remaining buildkite resources in the namespace
	fmt.Println("🗑️ Deleting any remaining Buildkite resources...")
	
	// List of resource types to check and delete
	resourceTypes := []string{
		"deployments", "statefulsets", "daemonsets", 
		"services", "configmaps", "secrets",
		"serviceaccounts", "roles", "rolebindings",
	}

	for _, resType := range resourceTypes {
		if c.All {
			// Delete all agent-stack resources
			checkCmd := exec.Command(kubectlPath, "get", resType, "-n", "buildkite", "-l", "app.kubernetes.io/part-of=agent-stack-k8s", "--no-headers")
			output, _ := checkCmd.CombinedOutput()
			
			if len(strings.TrimSpace(string(output))) > 0 {
				// Delete resources
				deleteCmd := exec.Command(kubectlPath, "delete", resType, "-n", "buildkite", "-l", "app.kubernetes.io/part-of=agent-stack-k8s")
				deleteCmd.Stdout = os.Stdout
				deleteCmd.Stderr = os.Stderr
				if err := deleteCmd.Run(); err != nil {
					fmt.Printf("⚠️ Failed to delete %s: %s\n", resType, err)
				}
			}
		} else {
			// Delete resources specific to the named release
			checkCmd := exec.Command(kubectlPath, "get", resType, "-n", "buildkite", 
				"-l", fmt.Sprintf("app.kubernetes.io/instance=%s", c.Name), "--no-headers")
			output, _ := checkCmd.CombinedOutput()
			
			if len(strings.TrimSpace(string(output))) > 0 {
				// Delete resources
				deleteCmd := exec.Command(kubectlPath, "delete", resType, "-n", "buildkite", 
					"-l", fmt.Sprintf("app.kubernetes.io/instance=%s", c.Name))
				deleteCmd.Stdout = os.Stdout
				deleteCmd.Stderr = os.Stderr
				if err := deleteCmd.Run(); err != nil {
					fmt.Printf("⚠️ Failed to delete %s: %s\n", resType, err)
				}
			}
		}
	}

	// Wait for pods to terminate (unless --no-wait was specified)
	if !c.NoWait {
		fmt.Printf("⏳ Waiting for pods to terminate (timeout: %ds)...\n", c.Timeout)
		
		timeoutDuration := time.Duration(c.Timeout) * time.Second
		startTime := time.Now()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		
		allTerminated := false
		
		for !allTerminated {
			// Check if timeout has been reached
			if time.Since(startTime) > timeoutDuration {
				fmt.Println("⚠️ Timed out waiting for pods to terminate")
				break
			}
			
			// Check for pods specific to the stack being deleted
			var checkPodsCmd *exec.Cmd
			if c.All {
				// Check for any agent-stack-k8s pods
				checkPodsCmd = exec.Command(kubectlPath, "get", "pods", "-n", "buildkite", 
					"-l", "app.kubernetes.io/part-of=agent-stack-k8s", 
					"--field-selector=status.phase!=Succeeded,status.phase!=Failed", "--no-headers")
			} else {
				// Check for pods specific to this release
				checkPodsCmd = exec.Command(kubectlPath, "get", "pods", "-n", "buildkite", 
					"-l", fmt.Sprintf("app.kubernetes.io/instance=%s", c.Name),
					"--field-selector=status.phase!=Succeeded,status.phase!=Failed", "--no-headers")
			}
			
			podsOutput, err := checkPodsCmd.CombinedOutput()
			if err != nil {
				// If the command fails (e.g., namespace doesn't exist, no resources found), consider pods terminated
				fmt.Println("✅ All pods terminated successfully")
				allTerminated = true
				break
			}
			
			remainingPods := strings.TrimSpace(string(podsOutput))
			if len(remainingPods) == 0 {
				fmt.Println("✅ All pods terminated successfully")
				allTerminated = true
				break
			}
			
			// Show progress - count remaining pods
			// Filter out empty lines and "No resources found" messages
			podLines := []string{}
			for _, line := range strings.Split(remainingPods, "\n") {
				line = strings.TrimSpace(line)
				if line != "" && !strings.Contains(line, "No resources found") {
					podLines = append(podLines, line)
				}
			}
			
			podCount := len(podLines)
			if podCount == 0 {
				fmt.Println("✅ All pods terminated successfully")
				allTerminated = true
				break
			}
			
			fmt.Printf("⏳ Still waiting for %d pod(s) to terminate...\n", podCount)
			
			// Wait before checking again
			select {
			case <-ticker.C:
				// Continue the loop on each tick
			}
		}
	} else {
		fmt.Println("ℹ️ Skipping wait for pod termination (--no-wait flag specified)")
	}
	// Only consider deleting the namespace if we're deleting all stacks
	if c.All {
		// Check if there are any remaining helm releases in the namespace
		if helmPath != "" {
			listCmd := exec.Command(helmPath, "list", "-n", "buildkite", "--output", "json")
			listOutput, err := listCmd.CombinedOutput()
			var hasRemainingReleases bool
			
			if err == nil && len(listOutput) > 0 && strings.Contains(string(listOutput), "\"name\":") {
				hasRemainingReleases = true
			}
			
			if !hasRemainingReleases {
				// Ask if the user wants to delete the namespace
				var deleteNamespace bool
				if !c.Force {
					nsPrompt := &survey.Confirm{
						Message: "Do you want to delete the entire 'buildkite' namespace?",
						Default: true,
					}
					if err := survey.AskOne(nsPrompt, &deleteNamespace); err != nil {
						return fmt.Errorf("prompt cancelled: %w", err)
					}
				} else {
					deleteNamespace = true
				}

				if deleteNamespace {
					fmt.Println("🗑️ Deleting the 'buildkite' namespace...")
					nsCmd := exec.Command(kubectlPath, "delete", "namespace", "buildkite", "--wait=false")
					nsCmd.Stdout = os.Stdout
					nsCmd.Stderr = os.Stderr
					if err := nsCmd.Run(); err != nil {
						fmt.Printf("⚠️ Failed to delete namespace: %s\n", err)
					} else {
						fmt.Println("✅ Namespace deletion initiated (this may continue in the background)")
					}
				}
			} else {
				fmt.Println("ℹ️ Not deleting 'buildkite' namespace as it contains other releases")
			}
		}
	}

	// Delete agent tokens from Buildkite API
	var deletedTokens int
	if client != nil && len(clustersToDelete) > 0 {
		fmt.Println("\n🗑️ Cleaning up Buildkite agent tokens...")
		
		for _, cluster := range clustersToDelete {
			if cluster.TokenID != "" {
				tokenCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				fmt.Printf("Deleting token for cluster '%s' (ID: %s)...\n", cluster.Name, cluster.UUID)
				
				err := client.DeleteToken(tokenCtx, cluster.UUID, cluster.TokenID)
				cancel()
				
				if err != nil {
					fmt.Printf("⚠️ Failed to delete token for cluster '%s': %s\n", cluster.Name, err)
				} else {
					fmt.Printf("✅ Successfully deleted token for cluster '%s'\n", cluster.Name)
					deletedTokens++
					
					// Update the config to remove the token ID
					err := client.RemoveTokenFromCluster(cluster.UUID, cluster.TokenID)
					if err != nil {
						fmt.Printf("⚠️ Warning: Failed to update config after token deletion: %s\n", err)
					}
				}
			}
		}
	} else if client != nil {
		fmt.Println("\nℹ️ No agent tokens found to clean up")
	}

	if c.All {
		fmt.Println("\n✨ All Buildkite agent stacks deleted successfully! ✨")
		if deletedTokens > 0 {
			fmt.Printf("Deleted %d agent tokens from Buildkite.\n", deletedTokens)
		}
	} else {
		fmt.Printf("\n✨ Buildkite agent stack '%s' deleted successfully! ✨\n", c.Name)
		if deletedTokens > 0 {
			fmt.Printf("Deleted %d agent token(s) from Buildkite.\n", deletedTokens)
		}
	}
	
	return nil
}
