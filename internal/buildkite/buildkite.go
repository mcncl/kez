package buildkite

import (
	"context"

	"github.com/buildkite/go-buildkite/v4"
)

func ListClusters(ctx context.Context, org string, client *buildkite.Client) ([]buildkite.Cluster, error) {
	var clusters []buildkite.Cluster
	clusters, _, err := client.Clusters.List(ctx, org, &buildkite.ClustersListOptions{})
	if err != nil {
		return nil, err
	}

	return clusters, nil
}

func CreateToken(ctx context.Context, client *buildkite.Client, org, clusterID, version string) (buildkite.ClusterToken, error) {
	var token buildkite.ClusterToken
	token, _, err := client.ClusterTokens.Create(ctx, org, clusterID, buildkite.ClusterTokenCreateUpdate{
		Description: "kez-" + version,
	})
	if err != nil {
		return buildkite.ClusterToken{}, err
	}

	return token, nil
}

func CreateTokenWithDescription(ctx context.Context, client *buildkite.Client, org, clusterID, description string) (buildkite.ClusterToken, error) {
	var token buildkite.ClusterToken
	token, _, err := client.ClusterTokens.Create(ctx, org, clusterID, buildkite.ClusterTokenCreateUpdate{
		Description: description,
	})
	if err != nil {
		return buildkite.ClusterToken{}, err
	}

	return token, nil
}

// ListTokens returns all tokens for a given cluster
func ListTokens(ctx context.Context, client *buildkite.Client, org, clusterID string) ([]buildkite.ClusterToken, error) {
	tokens, _, err := client.ClusterTokens.List(ctx, org, clusterID, &buildkite.ClusterTokensListOptions{})
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// DeleteToken deletes a token by ID from a cluster
func DeleteToken(ctx context.Context, client *buildkite.Client, org, clusterID, tokenID string) error {
	_, err := client.ClusterTokens.Delete(ctx, org, clusterID, tokenID)
	return err
}
