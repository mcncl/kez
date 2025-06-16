package k8s_test

import (
	"context"
	"fmt"
	"log"

	"github.com/mcncl/kez/internal/k8s"
)

func ExampleKubernetesClient() {
	// Set up config for the client
	config := k8s.KubernetesClientConfig{
		PreferredProvider: k8s.ProviderOrbstack, // Preferentially use Orbstack
		Namespace:         "buildkite",          // Default namespace
	}

	// Create a new Kubernetes client
	client, err := k8s.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Use the client to ensure a namespace exists
	ctx := context.Background()
	created, err := client.EnsureNamespaceExists(ctx, "buildkite")
	if err != nil {
		log.Fatalf("Failed to ensure namespace exists: %v", err)
	}

	if created {
		fmt.Println("Namespace was created")
	} else {
		fmt.Println("Namespace already exists")
	}

	// Install a Helm chart
	helmOpts := k8s.HelmInstallOptions{
		ReleaseName:     "agent-stack",
		ChartReference:  "oci://ghcr.io/buildkite/helm/agent-stack-k8s:0.28.0",
		Namespace:       "buildkite",
		CreateNamespace: true,
		Values: map[string]string{
			"agentToken":          "my-agent-token",
			"config.org":          "my-org",
			"config.cluster-uuid": "my-cluster-id",
		},
		JSONValues: map[string]string{
			"config.tags": "[\"queue=kubernetes\"]",
		},
	}

	err = client.InstallHelm(ctx, helmOpts)
	if err != nil {
		log.Fatalf("Failed to install Helm chart: %v", err)
	}

	// Get agent pod status
	running, total, err := client.GetAgentPodsStatus(ctx)
	if err != nil {
		log.Fatalf("Failed to get agent pod status: %v", err)
	}

	fmt.Printf("%d out of %d pods running\n", running, total)

	// Output: (This will not be verified since it depends on the actual system state)
}