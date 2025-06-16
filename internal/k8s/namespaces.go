package k8s

import (
	"fmt"
	"os"
	"os/exec"
)

// EnsureNamespaceExists checks if a Kubernetes namespace exists and creates it if it doesn't.
// Returns true if the namespace was created, false if it already existed.
func EnsureNamespaceExists(namespace string) (bool, error) {
	// Check if namespace exists
	checkCmd := exec.Command("kubectl", "get", "namespace", namespace, "--no-headers", "--ignore-not-found")
	output, err := checkCmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check namespace: %w", err)
	}

	// If namespace doesn't exist (empty output), create it
	if len(output) == 0 {
		fmt.Printf("🔨 Creating namespace '%s'...\n", namespace)
		createCmd := exec.Command("kubectl", "create", "namespace", namespace)
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