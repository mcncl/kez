package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/buildkite/go-buildkite/v4"
	bk "github.com/mcncl/kez/internal/buildkite" // Alias import
	"github.com/mcncl/kez/internal/config"
)

// Allow mocking the SDK client creation in tests
var buildkiteNewClient = buildkite.NewClient // <-- Add this line

// Client wraps the Buildkite API client and configuration.
type Client struct {
	config *config.Config
	client *buildkite.Client
}

// NewClient creates a new API client instance.
// It loads configuration and initializes the Buildkite Go client.
func NewClient() (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration for API client: %w", err)
	}

	if cfg.Buildkite.Token == "" {
		return nil, fmt.Errorf("buildkite API token is not configured. Please run 'kez configure'")
	}
	if cfg.Buildkite.OrgSlug == "" {
		return nil, fmt.Errorf("buildkite organisation slug is not configured. Please run 'kez configure'")
	}

	// Customize the underlying HTTP client if needed (e.g., timeouts)
	httpClient := &http.Client{
		Timeout: 30 * time.Second, // Example timeout
	}

	// Create the actual Buildkite client using the SDK's constructor
	// (or the mocked version during tests)
	client, err := buildkiteNewClient( // <-- Use the variable here
		buildkite.WithTokenAuth(cfg.Buildkite.Token),
		buildkite.WithHTTPClient(httpClient),
		// Add other options like WithBaseURL or WithUserAgent here if needed
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create buildkite client: %w", err)
	}

	return &Client{
		config: cfg,
		client: client,
	}, nil
}

// ListClusters fetches the list of clusters for the configured organization.
func (c *Client) ListClusters(ctx context.Context) ([]buildkite.Cluster, error) {
	if c.client == nil || c.config == nil {
		return nil, fmt.Errorf("API client not properly initialized")
	}
	// Use the aliased internal/buildkite package function
	clusters, err := bk.ListClusters(ctx, c.config.Buildkite.OrgSlug, c.client)
	if err != nil {
		// Add more context to the error
		return nil, fmt.Errorf("failed to list buildkite clusters for org '%s': %w", c.config.Buildkite.OrgSlug, err)
	}
	return clusters, nil
}

// GetOrgSlug returns the configured organization slug.
func (c *Client) GetOrgSlug() string {
	if c.config == nil {
		return "" // Or handle as an error if config must exist
	}
	return c.config.Buildkite.OrgSlug
}

// AddRecentCluster adds a cluster to the recent list in the config and saves it.
func (c *Client) AddRecentCluster(cluster buildkite.Cluster) error {
	if c.config == nil {
		return fmt.Errorf("config not loaded, cannot add recent cluster")
	}

	newRecent := config.RecentCluster{
		UUID:    cluster.ID,
		Name:    cluster.Name,
		OrgSlug: c.config.Buildkite.OrgSlug,
	}

	// Avoid duplicates - check if UUID already exists
	found := false
	index := -1
	for i, recent := range c.config.RecentClusters {
		if recent.UUID == newRecent.UUID {
			found = true
			index = i
			break
		}
	}

	if found {
		// Preserve the token ID if it exists
		if c.config.RecentClusters[index].TokenID != "" {
			newRecent.TokenID = c.config.RecentClusters[index].TokenID
			newRecent.TokenVal = c.config.RecentClusters[index].TokenVal
		}
		// Replace the existing entry
		c.config.RecentClusters[index] = newRecent
	} else {
		c.config.RecentClusters = append(c.config.RecentClusters, newRecent)
		maxRecent := 10
		if len(c.config.RecentClusters) > maxRecent {
			c.config.RecentClusters = c.config.RecentClusters[len(c.config.RecentClusters)-maxRecent:]
		}
		fmt.Printf("Added cluster '%s' (%s) to recent list.\n", newRecent.Name, newRecent.UUID)
	}

	// Save the updated config
	if err := config.Save(c.config); err != nil {
		return fmt.Errorf("failed to save config after adding recent cluster: %w", err)
	}

	return nil
}

// GetRecentClusters returns the list of recently used clusters from the config.
func (c *Client) GetRecentClusters() []config.RecentCluster {
	if c.config == nil {
		return []config.RecentCluster{} // Return empty slice if config not loaded
	}
	return c.config.RecentClusters
}

// CreateToken creates a new cluster token for the specified cluster with default versioned description.
func (c *Client) CreateToken(ctx context.Context, clusterID, version string) (buildkite.ClusterToken, error) {
	if c.client == nil || c.config == nil {
		return buildkite.ClusterToken{}, fmt.Errorf("API client not properly initialized")
	}

	token, err := bk.CreateToken(ctx, c.client, c.config.Buildkite.OrgSlug, clusterID, version)
	if err != nil {
		return buildkite.ClusterToken{}, fmt.Errorf("failed to create token for cluster '%s': %w", clusterID, err)
	}

	// Update recent clusters with token information
	for i, cluster := range c.config.RecentClusters {
		if cluster.UUID == clusterID {
			c.config.RecentClusters[i].TokenID = token.ID
			c.config.RecentClusters[i].TokenVal = token.Token

			// Save the updated config
			if err := config.Save(c.config); err != nil {
				fmt.Printf("Warning: Failed to save token ID to config: %v\n", err)
			}
			break
		}
	}

	return token, nil
}

// CreateTokenWithDescription creates a new cluster token for the specified cluster with a custom description.
func (c *Client) CreateTokenWithDescription(ctx context.Context, clusterID, description string) (buildkite.ClusterToken, error) {
	if c.client == nil || c.config == nil {
		return buildkite.ClusterToken{}, fmt.Errorf("API client not properly initialized")
	}

	token, err := bk.CreateTokenWithDescription(ctx, c.client, c.config.Buildkite.OrgSlug, clusterID, description)
	if err != nil {
		return buildkite.ClusterToken{}, fmt.Errorf("failed to create token for cluster '%s': %w", clusterID, err)
	}

	// Update recent clusters with token information
	for i, cluster := range c.config.RecentClusters {
		if cluster.UUID == clusterID {
			c.config.RecentClusters[i].TokenID = token.ID
			c.config.RecentClusters[i].TokenVal = token.Token

			// Save the updated config
			if err := config.Save(c.config); err != nil {
				fmt.Printf("Warning: Failed to save token ID to config: %v\n", err)
			}
			break
		}
	}

	return token, nil
}

// ListTokens fetches all tokens for a given cluster
func (c *Client) ListTokens(ctx context.Context, clusterID string) ([]buildkite.ClusterToken, error) {
	if c.client == nil || c.config == nil {
		return nil, fmt.Errorf("API client not properly initialized")
	}

	tokens, err := bk.ListTokens(ctx, c.client, c.config.Buildkite.OrgSlug, clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens for cluster '%s': %w", clusterID, err)
	}

	return tokens, nil
}

// DeleteToken deletes a token from a cluster
func (c *Client) DeleteToken(ctx context.Context, clusterID, tokenID string) error {
	if c.client == nil || c.config == nil {
		return fmt.Errorf("API client not properly initialized")
	}

	err := bk.DeleteToken(ctx, c.client, c.config.Buildkite.OrgSlug, clusterID, tokenID)
	if err != nil {
		return fmt.Errorf("failed to delete token '%s' for cluster '%s': %w", tokenID, clusterID, err)
	}

	return nil
}

// FindClusterByName returns a recent cluster by name (partial match)
func (c *Client) FindClusterByName(name string) ([]config.RecentCluster, error) {
	if c.config == nil {
		return nil, fmt.Errorf("config not loaded, cannot find cluster")
	}

	var matches []config.RecentCluster
	normalizedName := strings.ToLower(name)

	for _, cluster := range c.config.RecentClusters {
		// Check if the stack name is contained in the cluster name
		if strings.Contains(strings.ToLower(cluster.Name), normalizedName) {
			matches = append(matches, cluster)
		}
	}

	return matches, nil
}

// RemoveTokenFromCluster removes token information from a cluster in the recent list
func (c *Client) RemoveTokenFromCluster(clusterID, tokenID string) error {
	if c.config == nil {
		return fmt.Errorf("config not loaded, cannot update cluster token")
	}

	// Find the cluster and clear token info
	updated := false
	for i, cluster := range c.config.RecentClusters {
		if cluster.UUID == clusterID && cluster.TokenID == tokenID {
			c.config.RecentClusters[i].TokenID = ""
			c.config.RecentClusters[i].TokenVal = ""
			updated = true
			break
		}
	}

	if updated {
		// Save the updated config
		if err := config.Save(c.config); err != nil {
			return fmt.Errorf("failed to save config after removing token: %w", err)
		}
		return nil
	}

	return fmt.Errorf("token %s not found for cluster %s", tokenID, clusterID)
}
