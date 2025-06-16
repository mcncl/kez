package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	// GitHub API URL for the agent-stack-k8s releases
	agentStackRepoURL = "https://api.github.com/repos/buildkite/agent-stack-k8s/releases"
)

// Release represents a GitHub release
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	IsPrerelease bool     `json:"prerelease"`
	Body        string    `json:"body"`
}

// GetAgentStackReleases fetches the available releases of agent-stack-k8s from GitHub
func GetAgentStackReleases() ([]Release, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create request
	req, err := http.NewRequest("GET", agentStackRepoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set User-Agent header to avoid GitHub API rate limiting
	req.Header.Set("User-Agent", "buildkite-support-k8s-cli")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned non-OK status: %d", resp.StatusCode)
	}

	// Parse response
	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	return releases, nil
}

// FormatReleaseOption formats a release for display in a selection menu
func FormatReleaseOption(release Release) string {
	// Format date as "YYYY-MM-DD"
	date := release.PublishedAt.Format("2006-01-02")
	
	// Add prerelease indicator if applicable
	prereleaseTag := ""
	if release.IsPrerelease {
		prereleaseTag = " [pre-release]"
	}
	
	// Clean up tag name - remove 'v' prefix if present
	tag := release.TagName
	if strings.HasPrefix(tag, "v") {
		tag = tag[1:]
	}
	
	return fmt.Sprintf("%s (%s)%s", tag, date, prereleaseTag)
}

// GetChartVersion extracts the OCI chart version from a GitHub release tag
func GetChartVersion(tagName string) string {
	// Remove 'v' prefix if present
	version := tagName
	if strings.HasPrefix(version, "v") {
		version = version[1:]
	}
	
	return version
}
