package k8s

import (
	"context"
	"errors"
	"testing"
)

func TestMockKubernetesClient(t *testing.T) {
	// Create a new mock client
	client := NewMockClient()
	
	// Set up custom behavior for a method
	expectedError := errors.New("test error")
	client.VerifyClusterConnectionFunc = func(ctx context.Context) error {
		return expectedError
	}
	
	// Call the method
	err := client.VerifyClusterConnection(context.Background())
	
	// Check the result
	if err != expectedError {
		t.Errorf("Expected error %v, but got %v", expectedError, err)
	}
	
	// Check that the call was tracked
	if client.Calls.VerifyClusterConnection != 1 {
		t.Errorf("Expected 1 call to VerifyClusterConnection, but got %d", client.Calls.VerifyClusterConnection)
	}
	
	// Test another method
	created, err := client.EnsureNamespaceExists(context.Background(), "test-namespace")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !created {
		t.Errorf("Expected namespace to be created")
	}
	if client.Calls.EnsureNamespaceExists != 1 {
		t.Errorf("Expected 1 call to EnsureNamespaceExists, but got %d", client.Calls.EnsureNamespaceExists)
	}
}