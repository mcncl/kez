# kez - Buildkite Agent Stack Manager

A command-line tool for managing Buildkite agent stacks in Kubernetes clusters.

## Features

- **Interactive cluster selection** - Browse and select from your Buildkite clusters
- **Version management** - Fetch and select agent-stack-k8s versions from GitHub releases
- **Stack lifecycle management** - Create, monitor, and delete agent stacks
- **SSH key support** - Generate and manage SSH keys for private repository access
- **Multi-stack support** - Handle multiple agent stacks in the same cluster
- **Token management** - Automatic cleanup of Buildkite agent tokens

## Installation

### Build from Source

```bash
go build -o kez
```

### Requirements

- Go 1.21 or later
- Access to a Kubernetes cluster with `kubectl` configured
- Helm 3.x installed
- Buildkite API token

## Usage

### Initial Setup

Configure your Buildkite API token:

```bash
kez configure
```

This will prompt you for your Buildkite API token and organization slug.

### Stack Management

#### Create an Agent Stack

Create a new Buildkite agent stack:

```bash
kez stack create
```

This interactive command will:
1. Fetch your available Buildkite clusters
2. Prompt you to select a cluster
3. Fetch available agent-stack-k8s versions from GitHub
4. Prompt you to select a version
5. Optionally generate SSH keys for private repositories
6. Install the agent stack using Helm

#### Specify Options

You can specify options to skip interactive prompts:

```bash
# Use a specific version
kez stack create --version=0.28.1

# Suppress non-essential output
kez stack create --quiet

# Specify a custom stack name
kez stack create --name=my-custom-stack
```

#### Check Stack Status

View the status of your agent stacks:

```bash
kez stack status
```

For detailed information:

```bash
kez stack status --verbose
```

#### Delete an Agent Stack

Remove an agent stack:

```bash
kez stack delete
```

The tool will automatically discover installed stacks and prompt you to select which one to delete. You can also specify:

```bash
# Delete a specific stack
kez stack delete --name=my-stack

# Delete all stacks
kez stack delete --all

# Skip confirmation prompts
kez stack delete --force
```

### Advanced Usage

#### SSH Key Management

The tool will interactively prompt you to configure SSH keys for accessing private repositories. It can:
- Use existing SSH keys from your ~/.ssh directory
- Generate a new SSH key pair if needed
- Store the private key as a Kubernetes secret
- Display the public key for you to add to your Git provider

#### Multiple Stacks

You can run multiple agent stacks in the same cluster by giving them different names:

```bash
kez stack create --name=production-agents
kez stack create --name=development-agents
```

#### Configuration

The tool stores configuration in `~/.config/kez/config.json`, including:
- Buildkite API token and organization
- Recently used clusters
- Agent token information for cleanup

## Commands Reference

### Global Options

- `--help` - Show help information
- `--version` - Show version information

### `kez configure`

Set up Buildkite API credentials.

### `kez stack create`

Create a new agent stack.

**Options:**
- `--version` - Specify agent-stack-k8s version
- `--name` - Custom stack name (default: auto-generated)
- `--quiet` - Suppress non-essential output

### `kez stack status`

Show status of installed agent stacks.

**Options:**
- `--verbose` - Show detailed information
- `--refresh` - Force refresh of status information

### `kez stack delete`

Delete an agent stack.

**Options:**
- `--name` - Specify stack name to delete
- `--all` - Delete all agent stacks
- `--force` - Skip confirmation prompts
- `--timeout` - Timeout for delete operations (default: 60s)
- `--no-wait` - Skip waiting for pod termination

## Development

### Build Commands

```bash
# Build the binary
go build -o kez

# Run tests
go test ./...

# Format code
gofmt -w .
```

### Project Structure

- `cmd/` - Command implementations
- `internal/api/` - Buildkite API client
- `internal/config/` - Configuration management
- `internal/k8s/` - Kubernetes utilities
- `internal/logger/` - Logging utilities

## License

[Your license information here]
