package k8s

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// kubectlClient implements KubernetesClient using kubectl CLI commands
type kubectlClient struct {
	config KubernetesClientConfig
}

// NewKubectlClient creates a new KubernetesClient that uses kubectl CLI commands
func NewKubectlClient(config KubernetesClientConfig) (KubernetesClient, error) {
	// Verify that kubectl is available
	_, err := exec.LookPath("kubectl")
	if err != nil {
		return nil, fmt.Errorf("kubectl not found in PATH: %w", err)
	}

	// Store the kubectl path in the client
	client := &kubectlClient{
		config: config,
	}

	return client, nil
}

// EnsureNamespaceExists implements KubernetesClient.EnsureNamespaceExists
func (c *kubectlClient) EnsureNamespaceExists(ctx context.Context, namespace string) (bool, error) {
	// Check if namespace exists
	checkCmd := exec.CommandContext(ctx, "kubectl", "get", "namespace", namespace, "--no-headers", "--ignore-not-found")
	output, err := checkCmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check namespace: %w", err)
	}

	// If namespace doesn't exist (empty output), create it
	if len(output) == 0 {
		fmt.Printf("🔨 Creating namespace '%s'...\n", namespace)
		createCmd := exec.CommandContext(ctx, "kubectl", "create", "namespace", namespace)
		createCmd.Stdout = os.Stdout
		createCmd.Stderr = os.Stderr
		if err := createCmd.Run(); err != nil {
			return false, fmt.Errorf("failed to create namespace: %w", err)
		}
		fmt.Printf("✅ Namespace '%s' created successfully\n", namespace)
		return true, nil
	}

	return false, nil
}

// DeleteNamespace implements KubernetesClient.DeleteNamespace
func (c *kubectlClient) DeleteNamespace(ctx context.Context, namespace string) error {
	// Check if namespace exists
	checkCmd := exec.CommandContext(ctx, "kubectl", "get", "namespace", namespace, "--no-headers", "--ignore-not-found")
	output, err := checkCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check namespace: %w", err)
	}

	// If namespace exists, delete it
	if len(output) > 0 {
		fmt.Printf("🗑️ Deleting namespace '%s'...\n", namespace)
		deleteCmd := exec.CommandContext(ctx, "kubectl", "delete", "namespace", namespace, "--wait=false")
		deleteCmd.Stdout = os.Stdout
		deleteCmd.Stderr = os.Stderr
		if err := deleteCmd.Run(); err != nil {
			return fmt.Errorf("failed to delete namespace: %w", err)
		}
		fmt.Printf("✅ Namespace '%s' deletion initiated\n", namespace)
	}

	return nil
}

// InstallHelm implements KubernetesClient.InstallHelm
func (c *kubectlClient) InstallHelm(ctx context.Context, opts HelmInstallOptions) error {
	// Build the helm command
	args := []string{
		"upgrade",
		"--install",
		opts.ReleaseName,
		opts.ChartReference,
		"--namespace", opts.Namespace,
	}

	// Add create-namespace flag if specified
	if opts.CreateNamespace {
		args = append(args, "--create-namespace")
	}

	// Add all --set values
	for key, value := range opts.Values {
		args = append(args, "--set", fmt.Sprintf("%s=%s", key, value))
	}

	// Add all --set-json values
	for key, value := range opts.JSONValues {
		args = append(args, "--set-json", fmt.Sprintf("%s=%s", key, value))
	}

	// Execute the helm command
	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("🚀 Installing chart with Helm: %s\n", opts.ChartReference)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm installation failed: %w", err)
	}

	fmt.Printf("✅ Helm release '%s' installed successfully\n", opts.ReleaseName)
	return nil
}

// UninstallHelm implements KubernetesClient.UninstallHelm
func (c *kubectlClient) UninstallHelm(ctx context.Context, releaseName, namespace string) error {
	cmd := exec.CommandContext(ctx, "helm", "uninstall", releaseName, "--namespace", namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("🗑️ Uninstalling Helm release: %s\n", releaseName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm uninstallation failed: %w", err)
	}

	fmt.Printf("✅ Helm release '%s' uninstalled successfully\n", releaseName)
	return nil
}

// GetHelmReleaseStatus implements KubernetesClient.GetHelmReleaseStatus
func (c *kubectlClient) GetHelmReleaseStatus(ctx context.Context, releaseName, namespace string) (string, error) {
	cmd := exec.CommandContext(ctx, "helm", "status", releaseName, "--namespace", namespace, "--output", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get Helm release status: %w", err)
	}

	// For now, we're just returning the raw output
	// In a real implementation, you'd parse the JSON and return a more structured result
	return string(output), nil
}

// DetectProvider implements KubernetesClient.DetectProvider
func (c *kubectlClient) DetectProvider(ctx context.Context) (Provider, error) {
	// If a preferred provider is set, use that
	if c.config.PreferredProvider != ProviderUnknown {
		// Verify that the preferred provider is available
		// This is just a basic check - a real implementation would do more
		return c.config.PreferredProvider, nil
	}

	// Get current context
	contextCmd := exec.CommandContext(ctx, "kubectl", "config", "current-context")
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
		infoCmd := exec.CommandContext(ctx, "kubectl", "cluster-info")
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

// VerifyClusterConnection implements KubernetesClient.VerifyClusterConnection
func (c *kubectlClient) VerifyClusterConnection(ctx context.Context) error {
	// Try to access the cluster
	infoCmd := exec.CommandContext(ctx, "kubectl", "cluster-info")
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

// IsAgentStackInstalled implements KubernetesClient.IsAgentStackInstalled
func (c *kubectlClient) IsAgentStackInstalled(ctx context.Context) (bool, error) {
	// Check for the buildkite namespace
	nsCmd := exec.CommandContext(ctx, "kubectl", "get", "namespace", "buildkite", "--no-headers", "--ignore-not-found")
	nsOutput, err := nsCmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to check for buildkite namespace: %w", err)
	}

	if len(strings.TrimSpace(string(nsOutput))) == 0 {
		return false, nil
	}

	// Check for any agent pods
	podsCmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", "buildkite", "--selector=app.kubernetes.io/component=agent", "--no-headers")
	podsOutput, err := podsCmd.CombinedOutput()
	if err != nil {
		// If we can't get pods but namespace exists, stack might be partially installed
		return true, nil
	}

	return len(strings.TrimSpace(string(podsOutput))) > 0, nil
}

// GetAgentPodsStatus implements KubernetesClient.GetAgentPodsStatus
func (c *kubectlClient) GetAgentPodsStatus(ctx context.Context) (running, total int, err error) {
	// Get agent pods
	podsCmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", "buildkite",
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

// CreateSSHKeySecret implements KubernetesClient.CreateSSHKeySecret
func (c *kubectlClient) CreateSSHKeySecret(ctx context.Context, namespace, secretName, keyPath string) error {
	// Create the Kubernetes secret with kubectl
	createSecretCmd := exec.CommandContext(ctx, "kubectl", "create", "secret", "generic", secretName,
		"--from-file=SSH_PRIVATE_RSA_KEY="+keyPath,
		"-n", namespace,
		"--dry-run=client",
		"-o", "yaml",
	)

	// Pipe the output to kubectl apply
	applyCmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")

	// Connect the commands
	pipe, err := createSecretCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe for kubectl commands: %w", err)
	}
	applyCmd.Stdin = pipe

	// Set stderr for both commands
	createSecretCmd.Stderr = os.Stderr
	applyCmd.Stderr = os.Stderr
	applyCmd.Stdout = os.Stdout

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

	fmt.Printf("✅ SSH key secret '%s' created successfully!\n", secretName)
	return nil
}
