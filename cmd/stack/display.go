package stack

import (
	"fmt"
	"io"
	"os"

	"github.com/mcncl/kez/internal/utils"
)

// OutputConfig controls output verbosity
type OutputConfig struct {
	// QuietMode suppresses non-essential output
	QuietMode bool
	
	// Writer is where output is written (usually os.Stdout)
	Writer io.Writer
}

// DefaultOutput returns the default output configuration
func DefaultOutput() OutputConfig {
	return OutputConfig{
		QuietMode: false,
		Writer:    os.Stdout,
	}
}

// NewQuietOutput returns an output configuration with quiet mode enabled
func NewQuietOutput() OutputConfig {
	return OutputConfig{
		QuietMode: true,
		Writer:    os.Stdout,
	}
}

// printClusterSelected prints a message indicating a cluster was selected
func printClusterSelected(name, id string, output OutputConfig) {
	if output.QuietMode {
		return
	}
	fmt.Fprintf(output.Writer, "\n%s\n", utils.FormatSuccess("Selected cluster: " + utils.FormatResourceName(name, id)))
}

// printVersionSelected prints a message indicating a version was selected
func printVersionSelected(version string, output OutputConfig) {
	if output.QuietMode {
		return
	}
	fmt.Fprintf(output.Writer, "\n%s\n", utils.FormatSuccess("Selected version: " + version))
}

// printVersionSpecified prints a message indicating a version was specified
func printVersionSpecified(version string, output OutputConfig) {
	if output.QuietMode {
		return
	}
	fmt.Fprintf(output.Writer, "\n%s\n", utils.FormatSuccess("Using specified version: " + version))
}

// printTokenCreated prints a message indicating a token was created
func printTokenCreated(description, id string, output OutputConfig) {
	if output.QuietMode {
		return
	}
	fmt.Fprintf(output.Writer, "%s\n", utils.FormatSuccess("New token created with description: " + description + " (ID: " + utils.TruncateID(id, "", false) + ")"))
}

// printSSHKeySecretCreated prints a message indicating an SSH key secret was created
func printSSHKeySecretCreated(output OutputConfig) {
	if output.QuietMode {
		return
	}
	fmt.Fprintf(output.Writer, "%s\n", utils.FormatSuccess("SSH key secret created successfully!"))
}

// printAgentStackInstalled prints a message indicating an agent stack was installed
func printAgentStackInstalled(name, clusterName, clusterID, orgSlug, version string, output OutputConfig) {
	// Always print essential status messages, even in quiet mode
	fmt.Fprintf(output.Writer, "\n%s\n", utils.FormatSuccess("Agent stack installed successfully! ✨"))
	fmt.Fprintf(output.Writer, "Stack Name: %s\n", name)
	fmt.Fprintf(output.Writer, "Cluster: %s\n", utils.FormatResourceName(clusterName, clusterID))
	fmt.Fprintf(output.Writer, "Organization: %s\n", orgSlug)
	fmt.Fprintf(output.Writer, "Version: %s\n", version)
}

// printSSHKeyGenerated prints a message indicating an SSH key was generated
func printSSHKeyGenerated(output OutputConfig) {
	if output.QuietMode {
		return
	}
	fmt.Fprintf(output.Writer, "\n%s\n", utils.FormatSuccess("SSH key generated successfully!"))
}