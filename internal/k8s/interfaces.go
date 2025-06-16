package k8s

import "context"

// KubernetesClient defines the interface for interacting with Kubernetes
type KubernetesClient interface {
	// Namespace operations
	EnsureNamespaceExists(ctx context.Context, namespace string) (bool, error)
	DeleteNamespace(ctx context.Context, namespace string) error
	
	// Helm operations
	InstallHelm(ctx context.Context, opts HelmInstallOptions) error
	UninstallHelm(ctx context.Context, releaseName, namespace string) error
	GetHelmReleaseStatus(ctx context.Context, releaseName, namespace string) (string, error)
	
	// Provider operations
	DetectProvider(ctx context.Context) (Provider, error)
	VerifyClusterConnection(ctx context.Context) error
	
	// Agent stack operations
	IsAgentStackInstalled(ctx context.Context) (bool, error)
	GetAgentPodsStatus(ctx context.Context) (running int, total int, err error)
	
	// Secret operations
	CreateSSHKeySecret(ctx context.Context, namespace, secretName, keyPath string) error
}

// KubernetesClientConfig contains configuration for a KubernetesClient
type KubernetesClientConfig struct {
	// KubeconfigPath is the path to the kubeconfig file
	KubeconfigPath string
	
	// PreferredProvider is the preferred K8s provider, if any
	PreferredProvider Provider
	
	// Namespace is the default namespace for operations
	Namespace string
}

// NewClient creates a new KubernetesClient with the provided configuration
func NewClient(config KubernetesClientConfig) (KubernetesClient, error) {
	// Return a kubectl-based client by default
	return NewKubectlClient(config)
}