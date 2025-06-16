package k8s

import (
	"context"
)

// MockKubernetesClient is a mock implementation of KubernetesClient for testing
type MockKubernetesClient struct {
	// Configuration
	Config KubernetesClientConfig

	// Mock responses for methods
	EnsureNamespaceExistsFunc   func(ctx context.Context, namespace string) (bool, error)
	DeleteNamespaceFunc         func(ctx context.Context, namespace string) error
	InstallHelmFunc             func(ctx context.Context, opts HelmInstallOptions) error
	UninstallHelmFunc           func(ctx context.Context, releaseName, namespace string) error
	GetHelmReleaseStatusFunc    func(ctx context.Context, releaseName, namespace string) (string, error)
	DetectProviderFunc          func(ctx context.Context) (Provider, error)
	VerifyClusterConnectionFunc func(ctx context.Context) error
	IsAgentStackInstalledFunc   func(ctx context.Context) (bool, error)
	GetAgentPodsStatusFunc      func(ctx context.Context) (running int, total int, err error)
	CreateSSHKeySecretFunc      func(ctx context.Context, namespace, secretName, keyPath string) error

	// Call tracking for assertions
	Calls struct {
		EnsureNamespaceExists   int
		DeleteNamespace         int
		InstallHelm             int
		UninstallHelm           int
		GetHelmReleaseStatus    int
		DetectProvider          int
		VerifyClusterConnection int
		IsAgentStackInstalled   int
		GetAgentPodsStatus      int
		CreateSSHKeySecret      int
	}
}

// NewMockClient creates a new mock KubernetesClient with default implementations
func NewMockClient() *MockKubernetesClient {
	return &MockKubernetesClient{
		EnsureNamespaceExistsFunc: func(ctx context.Context, namespace string) (bool, error) {
			return true, nil
		},
		DeleteNamespaceFunc: func(ctx context.Context, namespace string) error {
			return nil
		},
		InstallHelmFunc: func(ctx context.Context, opts HelmInstallOptions) error {
			return nil
		},
		UninstallHelmFunc: func(ctx context.Context, releaseName, namespace string) error {
			return nil
		},
		GetHelmReleaseStatusFunc: func(ctx context.Context, releaseName, namespace string) (string, error) {
			return "mock-status", nil
		},
		DetectProviderFunc: func(ctx context.Context) (Provider, error) {
			return ProviderOrbstack, nil
		},
		VerifyClusterConnectionFunc: func(ctx context.Context) error {
			return nil
		},
		IsAgentStackInstalledFunc: func(ctx context.Context) (bool, error) {
			return true, nil
		},
		GetAgentPodsStatusFunc: func(ctx context.Context) (int, int, error) {
			return 3, 3, nil
		},
		CreateSSHKeySecretFunc: func(ctx context.Context, namespace, secretName, keyPath string) error {
			return nil
		},
	}
}

// EnsureNamespaceExists implements KubernetesClient.EnsureNamespaceExists
func (m *MockKubernetesClient) EnsureNamespaceExists(ctx context.Context, namespace string) (bool, error) {
	m.Calls.EnsureNamespaceExists++
	return m.EnsureNamespaceExistsFunc(ctx, namespace)
}

// DeleteNamespace implements KubernetesClient.DeleteNamespace
func (m *MockKubernetesClient) DeleteNamespace(ctx context.Context, namespace string) error {
	m.Calls.DeleteNamespace++
	return m.DeleteNamespaceFunc(ctx, namespace)
}

// InstallHelm implements KubernetesClient.InstallHelm
func (m *MockKubernetesClient) InstallHelm(ctx context.Context, opts HelmInstallOptions) error {
	m.Calls.InstallHelm++
	return m.InstallHelmFunc(ctx, opts)
}

// UninstallHelm implements KubernetesClient.UninstallHelm
func (m *MockKubernetesClient) UninstallHelm(ctx context.Context, releaseName, namespace string) error {
	m.Calls.UninstallHelm++
	return m.UninstallHelmFunc(ctx, releaseName, namespace)
}

// GetHelmReleaseStatus implements KubernetesClient.GetHelmReleaseStatus
func (m *MockKubernetesClient) GetHelmReleaseStatus(ctx context.Context, releaseName, namespace string) (string, error) {
	m.Calls.GetHelmReleaseStatus++
	return m.GetHelmReleaseStatusFunc(ctx, releaseName, namespace)
}

// DetectProvider implements KubernetesClient.DetectProvider
func (m *MockKubernetesClient) DetectProvider(ctx context.Context) (Provider, error) {
	m.Calls.DetectProvider++
	return m.DetectProviderFunc(ctx)
}

// VerifyClusterConnection implements KubernetesClient.VerifyClusterConnection
func (m *MockKubernetesClient) VerifyClusterConnection(ctx context.Context) error {
	m.Calls.VerifyClusterConnection++
	return m.VerifyClusterConnectionFunc(ctx)
}

// IsAgentStackInstalled implements KubernetesClient.IsAgentStackInstalled
func (m *MockKubernetesClient) IsAgentStackInstalled(ctx context.Context) (bool, error) {
	m.Calls.IsAgentStackInstalled++
	return m.IsAgentStackInstalledFunc(ctx)
}

// GetAgentPodsStatus implements KubernetesClient.GetAgentPodsStatus
func (m *MockKubernetesClient) GetAgentPodsStatus(ctx context.Context) (int, int, error) {
	m.Calls.GetAgentPodsStatus++
	return m.GetAgentPodsStatusFunc(ctx)
}

// CreateSSHKeySecret implements KubernetesClient.CreateSSHKeySecret
func (m *MockKubernetesClient) CreateSSHKeySecret(ctx context.Context, namespace, secretName, keyPath string) error {
	m.Calls.CreateSSHKeySecret++
	return m.CreateSSHKeySecretFunc(ctx, namespace, secretName, keyPath)
}
