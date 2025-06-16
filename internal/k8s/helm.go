package k8s

import (
	"fmt"
	"os"
	"os/exec"
)

// HelmInstallOptions represents the configuration options for installing a Helm chart
type HelmInstallOptions struct {
	// ReleaseName is the name of the Helm release
	ReleaseName string
	// ChartReference is the chart reference (e.g., oci://ghcr.io/buildkite/helm/agent-stack-k8s:0.28.0)
	ChartReference string
	// Namespace where the release should be installed
	Namespace string
	// CreateNamespace indicates whether the namespace should be created if it doesn't exist
	CreateNamespace bool
	// Values is a map of values to set on the chart (--set flag)
	Values map[string]string
	// JSONValues is a map of JSON values to set on the chart (--set-json flag)
	JSONValues map[string]string
}

// InstallWithHelm installs or upgrades a Helm chart using the provided options
func InstallWithHelm(opts HelmInstallOptions) error {
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
	cmd := exec.Command("helm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("🚀 Installing chart with Helm: %s\n", opts.ChartReference)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm installation failed: %w", err)
	}

	fmt.Printf("✅ Helm release '%s' installed successfully\n", opts.ReleaseName)
	return nil
}

// UninstallWithHelm uninstalls a Helm release
func UninstallWithHelm(releaseName, namespace string) error {
	cmd := exec.Command("helm", "uninstall", releaseName, "--namespace", namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("🗑️ Uninstalling Helm release: %s\n", releaseName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm uninstallation failed: %w", err)
	}

	fmt.Printf("✅ Helm release '%s' uninstalled successfully\n", releaseName)
	return nil
}

// GetHelmReleaseStatus gets the status of a Helm release
func GetHelmReleaseStatus(releaseName, namespace string) (string, error) {
	cmd := exec.Command("helm", "status", releaseName, "--namespace", namespace, "--output", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get Helm release status: %w", err)
	}

	// For now, we're just returning the raw output
	// In a real implementation, you'd parse the JSON and return a more structured result
	return string(output), nil
}