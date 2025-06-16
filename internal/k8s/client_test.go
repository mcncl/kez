package k8s

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	// Create a new client with default config
	config := KubernetesClientConfig{
		PreferredProvider: ProviderOrbstack,
		Namespace:         "buildkite",
	}
	
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	
	// Check that the client is not nil
	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}
	
	// Check that the client implements the KubernetesClient interface
	// This is a compile-time check, but we'll do a simple runtime check too
	if _, ok := client.(KubernetesClient); !ok {
		t.Fatal("Client does not implement KubernetesClient interface")
	}
}

func TestClientInterfaces(t *testing.T) {
	// This test verifies that our interface hierarchy is correct
	
	// Create a mock client
	mockClient := NewMockClient()
	
	// Verify it satisfies the KubernetesClient interface
	var k8sClient KubernetesClient = mockClient
	
	// Just do a simple method call to verify the interface works
	_, err := k8sClient.DetectProvider(nil)
	if err != nil {
		t.Errorf("Unexpected error from DetectProvider: %v", err)
	}
	
	// Create a kubectl client
	kubectlClient, err := NewKubectlClient(KubernetesClientConfig{})
	if err != nil {
		// Skip test if kubectl is not available
		t.Skip("kubectl not available, skipping test")
	}
	
	// Verify it satisfies the KubernetesClient interface
	k8sClient = kubectlClient
	
	// No need to call methods, this is just a compile-time check
	_ = k8sClient
}