package k8s

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Provider represents a Kubernetes provider like Orbstack, Minikube, etc.
type Provider string

const (
	ProviderUnknown   Provider = "unknown"
	ProviderOrbstack  Provider = "orbstack"
	ProviderMinikube  Provider = "minikube"
	ProviderKind      Provider = "kind"
	ProviderDockerDsk Provider = "docker-desktop"
)

// DetectProvider attempts to detect the local Kubernetes provider being used.
func DetectProvider() (Provider, error) {
	// First, check if kubectl is available
	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		return ProviderUnknown, fmt.Errorf("kubectl not found in PATH: %w", err)
	}

	// Get current context
	contextCmd := exec.Command(kubectlPath, "config", "current-context")
	contextOutput, err := contextCmd.CombinedOutput()
	if err != nil {
		return ProviderUnknown, fmt.Errorf("failed to get current context: %w", err)
	}

	context := strings.TrimSpace(string(contextOutput))

	// Check for known context patterns
	switch {
	case strings.Contains(context, "orbstack"):
		return ProviderOrbstack, nil
	case strings.Contains(context, "minikube"):
		return ProviderMinikube, nil
	case strings.Contains(context, "kind-"):
		return ProviderKind, nil
	case strings.Contains(context, "docker-desktop"):
		return ProviderDockerDsk, nil
	default:
		// Try to get more clues from cluster info
		infoCmd := exec.Command(kubectlPath, "cluster-info")
		infoOutput, err := infoCmd.CombinedOutput()
		if err != nil {
			// If we can't get info, just return unknown with the current context
			return ProviderUnknown, nil
		}

		info := strings.ToLower(string(infoOutput))
		switch {
		case strings.Contains(info, "orbstack"):
			return ProviderOrbstack, nil
		case strings.Contains(info, "minikube"):
			return ProviderMinikube, nil
		case strings.Contains(info, "kind"):
			return ProviderKind, nil
		case strings.Contains(info, "docker-desktop") || strings.Contains(info, "docker desktop"):
			return ProviderDockerDsk, nil
		default:
			return ProviderUnknown, nil
		}
	}
}

// VerifyClusterConnection checks if the Kubernetes cluster is accessible.
func VerifyClusterConnection() error {
	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		return fmt.Errorf("kubectl not found in PATH: %w", err)
	}

	versionCmd := exec.Command(kubectlPath, "version", "--client")
	versionCmd.Stdout = os.Stdout
	versionCmd.Stderr = os.Stderr
	if err := versionCmd.Run(); err != nil {
		return fmt.Errorf("failed to get kubectl version: %w", err)
	}

	// Try to access the cluster
	infoCmd := exec.Command(kubectlPath, "cluster-info")
	infoOutput, err := infoCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}

	// Check if we're actually connected
	if !strings.Contains(string(infoOutput), "Kubernetes control plane") && 
	   !strings.Contains(string(infoOutput), "Kubernetes master") {
		return fmt.Errorf("connected to cluster but did not get expected Kubernetes control plane info")
	}

	return nil
}

// IsAgentStackInstalled checks if the Buildkite agent stack is installed in the K8s cluster.
func IsAgentStackInstalled() (bool, error) {
	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		return false, fmt.Errorf("kubectl not found in PATH: %w", err)
	}

	// Check for the buildkite namespace
	nsCmd := exec.Command(kubectlPath, "get", "namespace", "buildkite", "--no-headers", "--ignore-not-found")
	nsOutput, err := nsCmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to check for buildkite namespace: %w", err)
	}

	if len(strings.TrimSpace(string(nsOutput))) == 0 {
		return false, nil
	}

	// Check for any agent pods
	podsCmd := exec.Command(kubectlPath, "get", "pods", "-n", "buildkite", "--selector=app.kubernetes.io/component=agent", "--no-headers")
	podsOutput, err := podsCmd.CombinedOutput()
	if err != nil {
		// If we can't get pods but namespace exists, stack might be partially installed
		return true, nil
	}

	return len(strings.TrimSpace(string(podsOutput))) > 0, nil
}

// GetAgentPodsStatus returns the status of Buildkite agent pods.
func GetAgentPodsStatus() (running, total int, err error) {
	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		return 0, 0, fmt.Errorf("kubectl not found in PATH: %w", err)
	}

	// Get agent pods
	podsCmd := exec.Command(kubectlPath, "get", "pods", "-n", "buildkite", 
		"--selector=app.kubernetes.io/component=agent", "--no-headers")
	podsOutput, err := podsCmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get agent pods: %w", err)
	}

	// Count running pods
	podLines := strings.Split(string(podsOutput), "\n")
	runningCount := 0
	totalCount := 0

	for _, line := range podLines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		
		totalCount++
		if strings.Contains(line, "Running") {
			runningCount++
		}
	}

	return runningCount, totalCount, nil
}
