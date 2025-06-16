package stack

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/alecthomas/kong"
	"github.com/mcncl/kez/internal/api"
	"github.com/mcncl/kez/internal/k8s"
)

// StatusCmd represents the 'stack status' command
type StatusCmd struct {
	Verbose bool `help:"Show more detailed information" short:"v"`
	Refresh bool `help:"Force refresh of all status information" short:"r"`
}

// Run executes the stack status command
func (c *StatusCmd) Run(ctx *kong.Context) error {
	fmt.Println("Checking Buildkite agent stack status...")

	// Initialize API client
	client, err := api.NewClient()
	if err != nil {
		return fmt.Errorf("failed to initialize API client: %w", err)
	}

	// Check if we have a running Kubernetes context
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
	}
	
	if provider == k8s.ProviderUnknown {
		fmt.Printf("✅ Connected to Kubernetes context: %s\n", currentContext)
	} else {
		fmt.Printf("✅ Connected to Kubernetes context: %s (%s)\n", currentContext, provider)
	}

	// Check if the agent stack is installed
	stackInstalled, err := k8s.IsAgentStackInstalled()
	if err != nil {
		return fmt.Errorf("failed to check if agent stack is installed: %w", err)
	}

	if !stackInstalled {
		fmt.Println("❌ Buildkite namespace not found. No agent stack is installed.")
		
		// Offer to create a new stack
		var createNew bool
		prompt := &survey.Confirm{
			Message: "Would you like to create a new agent stack?",
			Default: true,
		}
		
		if err := survey.AskOne(prompt, &createNew); err != nil {
			return fmt.Errorf("prompt cancelled: %w", err)
		}
		
		if createNew {
			// Create a new CreateCmd and run it
			createCmd := &CreateCmd{}
			return createCmd.Run(ctx)
		}
		
		return nil
	}

	fmt.Println("✅ Buildkite namespace exists")

	// Check for installed Helm releases
	helmPath, err := exec.LookPath("helm")
	if err != nil {
		fmt.Println("⚠️ Helm not found in PATH. Limited status information available.")
	} else {
		// List all releases in the buildkite namespace
		listCmd := exec.Command(helmPath, "list", "-n", "buildkite", "-o", "json")
		listOutput, err := listCmd.CombinedOutput()
		
		if err != nil {
			fmt.Printf("⚠️ Failed to list Helm releases: %s\n", err)
		} else if len(listOutput) > 0 && strings.TrimSpace(string(listOutput)) != "[]" {
			// Extract stack names from JSON output
			stackList := []string{}
			jsonStr := string(listOutput)
			
			// Simple JSON parsing - look for "name":"stackname" patterns
			namePattern := `"name":"`
			for {
				nameIdx := strings.Index(jsonStr, namePattern)
				if nameIdx == -1 {
					break
				}
				
				// Move past the pattern
				nameStart := nameIdx + len(namePattern)
				nameEnd := strings.Index(jsonStr[nameStart:], `"`)
				if nameEnd == -1 {
					break
				}
				
				stackName := jsonStr[nameStart : nameStart+nameEnd]
				stackList = append(stackList, stackName)
				
				// Move past this match for next iteration
				jsonStr = jsonStr[nameStart+nameEnd:]
			}
			
			if len(stackList) == 0 {
				fmt.Println("❌ No Buildkite agent stacks found")
			} else {
				fmt.Printf("✅ Found %d Buildkite agent stack(s): %s\n", len(stackList), strings.Join(stackList, ", "))
				
				// Show details for each stack
				for _, stackName := range stackList {
					if c.Verbose {
						fmt.Printf("\n=== Stack: %s ===\n", stackName)
						helmCmd := exec.Command(helmPath, "status", stackName, "-n", "buildkite")
						helmOutput, err := helmCmd.CombinedOutput()
						if err == nil {
							fmt.Println(string(helmOutput))
						}
						fmt.Println("==================")
					}
					
					// Extract the version using helm list for this specific stack
					versionCmd := exec.Command(helmPath, "list", "-n", "buildkite", "--filter", stackName, "-o", "json")
					versionOutput, err := versionCmd.CombinedOutput()
					if err == nil {
						versionStr := string(versionOutput)
						if strings.Contains(versionStr, "app_version") {
							parts := strings.Split(versionStr, "app_version")
							if len(parts) > 1 {
								versionPart := parts[1]
								if colonIdx := strings.Index(versionPart, ":"); colonIdx != -1 {
									versionPart = versionPart[colonIdx+1:]
									if quoteIdx := strings.Index(versionPart, "\""); quoteIdx != -1 {
										endQuoteIdx := strings.Index(versionPart[quoteIdx+1:], "\"")
										if endQuoteIdx != -1 {
											version := versionPart[quoteIdx+1 : quoteIdx+1+endQuoteIdx]
											fmt.Printf("📋 Stack '%s' Version: %s\n", stackName, version)
										}
									}
								}
							}
						}
					}
				}
			}
		} else {
			fmt.Println("❌ No Buildkite agent stacks found")
		}
	}

	// Check if Buildkite agents are running
	fmt.Println("\n🔍 Checking for Buildkite agents...")
	
	// Get detailed pod output for verbose mode if needed
	var podsOutput []byte
	if c.Verbose {
		podsCmd := exec.Command(kubectlPath, "get", "pods", "-n", "buildkite", "--selector=app.kubernetes.io/component=agent", "-o", "wide")
		podsOutput, _ = podsCmd.CombinedOutput()
	}
	
	// Use our k8s utility to get pod status
	runningCount, totalPods, err := k8s.GetAgentPodsStatus()
	if err != nil {
		if strings.Contains(err.Error(), "buildkite not found") {
			fmt.Println("❌ No Buildkite namespace found")
		} else {
			return fmt.Errorf("failed to get agent pod status: %w", err)
		}
	}
	
	if totalPods == 0 {
		fmt.Println("❌ No Buildkite agent pods found")
	} else {
		if runningCount == 0 {
			fmt.Println("❌ No Buildkite agents are running")
		} else if runningCount < totalPods {
			fmt.Printf("⚠️ %d/%d Buildkite agents are running\n", runningCount, totalPods)
		} else {
			fmt.Printf("✅ All %d Buildkite agents are running\n", runningCount)
		}
		
		if c.Verbose && len(podsOutput) > 0 {
			fmt.Println("\n=== Agent Pods ===")
			fmt.Println(string(podsOutput))
			fmt.Println("================")
		}
	}

	// Get cluster information from Buildkite
	clusterCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	recentClusters := client.GetRecentClusters()
	if len(recentClusters) > 0 {
		// Attempt to list clusters to find the current one
		clusters, err := client.ListClusters(clusterCtx)
		if err != nil {
			fmt.Println("⚠️ Unable to fetch Buildkite clusters: ", err)
		} else {
			// Try to find a cluster UUID match first
			// This assumes we can find a cluster UUID from the running agent pods
			// As a fallback, we'll just show the most recent cluster
			
			mostRecentCluster := recentClusters[len(recentClusters)-1]
			foundCluster := false
			
			for _, cluster := range clusters {
				if cluster.ID == mostRecentCluster.UUID {
					fmt.Printf("\n📋 Connected to Buildkite Cluster: %s (%s)\n", cluster.Name, cluster.ID)
					fmt.Printf("📋 Organization: %s\n", client.GetOrgSlug())
					
					// If we're in verbose mode, show more details
					if c.Verbose {
						fmt.Println("\n=== Cluster Details ===")
						w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
						fmt.Fprintf(w, "Name:\t%s\n", cluster.Name)
						fmt.Fprintf(w, "UUID:\t%s\n", cluster.ID)
						fmt.Fprintf(w, "Organization:\t%s\n", client.GetOrgSlug())
						fmt.Fprintf(w, "Created:\t%s\n", cluster.CreatedAt.Format("2006-01-02 15:04:05"))
						w.Flush()
						fmt.Println("=====================")
					}
					
					foundCluster = true
					break
				}
			}
			
			if !foundCluster {
				fmt.Printf("\n⚠️ Could not identify the current Buildkite cluster\n")
				fmt.Printf("Most recent cluster used: %s (%s)\n", mostRecentCluster.Name, mostRecentCluster.UUID)
			}
		}
	} else {
		fmt.Println("\n⚠️ No recent Buildkite clusters found in configuration")
	}

	// Check connection to Buildkite API
	fmt.Println("\n🔍 Verifying Buildkite API connection...")
	apiCtx, apiCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer apiCancel()
	
	_, err = client.ListClusters(apiCtx)
	if err != nil {
		fmt.Println("❌ Failed to connect to Buildkite API: ", err)
	} else {
		fmt.Println("✅ Successfully connected to Buildkite API")
	}
	
	// Provide overall status summary
	fmt.Println("\n=== Summary ===")
	fmt.Printf("Kubernetes Context: %s\n", currentContext)
	if provider != k8s.ProviderUnknown {
		fmt.Printf("Provider: %s\n", provider)
	}
	
	// Determine the agent stack status
	if !stackInstalled {
		fmt.Println("Agent Stack: Not installed")
	} else if runningCount == 0 {
		fmt.Println("Agent Stack: Installed but not running")
	} else if runningCount < totalPods {
		fmt.Printf("Agent Stack: Partially running (%d/%d agents)\n", runningCount, totalPods)
	} else {
		fmt.Println("Agent Stack: Running")
	}
	
	// Add Buildkite API status
	if apiCtx.Err() == nil { // Check if context was canceled due to error
		fmt.Println("Buildkite API: Connected")
	} else {
		fmt.Println("Buildkite API: Not connected")
	}
	
	return nil
}
