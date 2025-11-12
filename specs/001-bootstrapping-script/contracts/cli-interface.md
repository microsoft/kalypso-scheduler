# CLI Contract: bootstrap.sh

**Feature**: 001-bootstrapping-script  
**Date**: 2025-11-11  
**Version**: 1.0.0

## Overview

This document specifies the command-line interface for the Kalypso Scheduler bootstrapping script. The CLI follows standard Unix conventions and supports both interactive and non-interactive modes.

---

## Script Invocation

```bash
./scripts/bootstrap/bootstrap.sh [OPTIONS]
```

---

## Command-Line Options

### Cluster Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--cluster-name NAME` | string | - | Name for the AKS cluster (required if creating cluster) |
| `--cluster-create` | boolean | `true` | Create a new AKS cluster |
| `--cluster-existing CONTEXT` | string | - | Use existing cluster (kubectl context name) |
| `--cluster-region REGION` | string | `eastus` | Azure region for cluster creation |
| `--cluster-resource-group RG` | string | `{cluster-name}-rg` | Azure resource group name |
| `--cluster-nodes COUNT` | integer | `3` | Number of nodes in the cluster |
| `--cluster-vm-size SIZE` | string | `Standard_DS2_v2` | Azure VM size for nodes |
| `--cluster-k8s-version VERSION` | string | latest | Kubernetes version for the cluster |

**Validation**:
- `--cluster-name` must match `^[a-z0-9-]{1,63}$`
- If `--cluster-existing` is set, `--cluster-create` is ignored
- If `--cluster-create` is true, `--cluster-name` is required
- `--cluster-nodes` must be between 1 and 100
- `--cluster-region` must be a valid Azure region

**Examples**:
```bash
# Create new cluster with defaults
./bootstrap.sh --cluster-name my-kalypso

# Create cluster with custom configuration
./bootstrap.sh --cluster-name my-kalypso --cluster-nodes 5 --cluster-vm-size Standard_D4s_v3

# Use existing cluster
./bootstrap.sh --cluster-existing my-existing-cluster-context
```

---

### Repository Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--control-plane-create` | boolean | `true` | Create new control-plane repository |
| `--control-plane-repo URL` | string | - | Use existing control-plane repository |
| `--control-plane-name NAME` | string | `{cluster-name}-control-plane` | Name for new control-plane repo |
| `--gitops-create` | boolean | `true` | Create new gitops repository |
| `--gitops-repo URL` | string | - | Use existing gitops repository |
| `--gitops-name NAME` | string | `{cluster-name}-gitops` | Name for new gitops repo |
| `--github-org ORG` | string | - | GitHub organization (defaults to user account) |
| `--repo-visibility VISIBILITY` | string | `private` | Repository visibility: `public` or `private` |

**Validation**:
- If `--control-plane-repo` is set, `--control-plane-create` is ignored
- If `--gitops-repo` is set, `--gitops-create` is ignored
- Repository URLs must match `^https://github\.com/[^/]+/[^/]+$`
- Repository names must match GitHub naming rules: `^[a-zA-Z0-9._-]+$`
- `--repo-visibility` must be either `public` or `private`

**Examples**:
```bash
# Create both repositories with defaults
./bootstrap.sh --cluster-name my-kalypso

# Use existing control-plane, create gitops
./bootstrap.sh --cluster-name my-kalypso \
  --control-plane-repo https://github.com/myorg/my-control-plane

# Create repos in organization
./bootstrap.sh --cluster-name my-kalypso --github-org mycompany

# Create public repositories
./bootstrap.sh --cluster-name my-kalypso --repo-visibility public
```

---

### Installation Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--namespace NAMESPACE` | string | `kalypso-system` | Kubernetes namespace for Kalypso Scheduler |
| `--helm-chart CHART` | string | `./helm/kalypso-scheduler` | Helm chart location (path or repo/chart) |
| `--helm-version VERSION` | string | latest | Helm chart version |
| `--helm-values FILE` | string | - | Custom Helm values file |
| `--skip-install` | boolean | `false` | Skip Kalypso Scheduler installation |

**Validation**:
- `--namespace` must match Kubernetes namespace rules: `^[a-z0-9-]{1,63}$`
- `--helm-chart` must be a valid path or Helm chart reference
- `--helm-values` must be a valid YAML file if specified

**Examples**:
```bash
# Install with defaults
./bootstrap.sh --cluster-name my-kalypso

# Install with custom namespace and values
./bootstrap.sh --cluster-name my-kalypso \
  --namespace kalypso-prod \
  --helm-values ./my-values.yaml

# Set up infrastructure but skip installation
./bootstrap.sh --cluster-name my-kalypso --skip-install
```

---

### Execution Control Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--non-interactive` | boolean | `false` | Run in non-interactive mode (fail if inputs missing) |
| `--auto-rollback` | boolean | `false` | Automatically rollback on error (non-interactive mode) |
| `--skip-validation` | boolean | `false` | Skip prerequisite validation (dangerous!) |
| `--skip-confirmation` | boolean | `false` | Skip confirmation prompts |
| `--config FILE` | string | - | Load configuration from YAML file |
| `--state-file FILE` | string | `$HOME/.kalypso/bootstrap-state.json` | Path to state file |
| `--force` | boolean | `false` | Ignore existing state and recreate resources |

**Validation**:
- `--config` must be a valid YAML file if specified
- `--state-file` parent directory must exist or be creatable

**Examples**:
```bash
# Non-interactive mode with config file (for CI/CD)
./bootstrap.sh --config ./bootstrap-config.yaml --non-interactive

# Force recreation ignoring existing state
./bootstrap.sh --cluster-name my-kalypso --force

# Skip confirmation prompts
./bootstrap.sh --cluster-name my-kalypso --skip-confirmation
```

---

### Logging and Output Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--verbose` / `-v` | boolean | `false` | Enable verbose output (DEBUG level) |
| `--quiet` / `-q` | boolean | `false` | Suppress non-error output |
| `--no-color` | boolean | `false` | Disable colored output |
| `--log-file FILE` | string | `$HOME/.kalypso/bootstrap-{timestamp}.log` | Path to log file |

**Examples**:
```bash
# Verbose output for debugging
./bootstrap.sh --cluster-name my-kalypso --verbose

# Quiet mode for scripting
./bootstrap.sh --cluster-name my-kalypso --quiet

# Custom log file location
./bootstrap.sh --cluster-name my-kalypso --log-file ./bootstrap.log
```

---

### Utility Commands

| Command | Description |
|---------|-------------|
| `--status` | Show current bootstrap state (from state file) |
| `--clean` | Clean up all resources tracked in state file |
| `--validate-only` | Run prerequisite validation and exit |
| `--help` / `-h` | Display help message |
| `--version` | Display script version |

**Examples**:
```bash
# Check current state
./bootstrap.sh --status

# Clean up all resources
./bootstrap.sh --clean

# Validate prerequisites without running bootstrap
./bootstrap.sh --validate-only
```

---

## Configuration File Format

When using `--config FILE`, the file must be in YAML format:

```yaml
# bootstrap-config.yaml
cluster:
  name: my-kalypso-cluster
  create: true
  region: eastus
  resourceGroup: my-kalypso-rg
  nodeCount: 3
  vmSize: Standard_DS2_v2
  kubernetesVersion: "1.28"  # optional

repositories:
  controlPlane:
    create: true
    name: my-control-plane
    # url: https://github.com/myorg/my-control-plane  # if create: false
  gitops:
    create: true
    name: my-gitops
    # url: https://github.com/myorg/my-gitops  # if create: false
  githubOrg: ""  # optional, uses user account if empty
  visibility: private  # or public

installation:
  namespace: kalypso-system
  helmChart: ./helm/kalypso-scheduler
  helmVersion: ""  # optional, uses latest if empty
  helmValues: ""  # optional path to values file
  skipInstall: false

options:
  nonInteractive: false
  autoRollback: false
  skipValidation: false
  skipConfirmation: false
  verbose: false
  quiet: false
  noColor: false
  force: false
```

**Notes**:
- CLI flags override config file values
- Config file can contain partial configuration; missing values are prompted or use defaults
- Use `--non-interactive` to enforce all required values are in config file

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success - bootstrap completed successfully |
| `1` | General error - see log for details |
| `2` | Prerequisite validation failed |
| `3` | Configuration validation failed |
| `4` | Cluster operation failed |
| `5` | Repository operation failed |
| `6` | Installation failed |
| `7` | User cancelled operation |
| `64` | Invalid command-line arguments |

---

## Environment Variables

The script respects the following environment variables:

| Variable | Purpose | Example |
|----------|---------|---------|
| `AZURE_SUBSCRIPTION_ID` | Azure subscription to use | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| `AZURE_TENANT_ID` | Azure tenant ID | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` |
| `GITHUB_TOKEN` | GitHub authentication token | `ghp_xxxxxxxxxxxxx` |
| `KUBECONFIG` | Kubeconfig file location | `$HOME/.kube/config` |
| `NO_COLOR` | Disable colored output | `1` or `true` |
| `DEBUG` | Enable debug mode | `1` or `true` |

**Notes**:
- If credentials are not in environment variables, script will prompt or use CLI authentication
- `KUBECONFIG` follows standard kubectl behavior
- `NO_COLOR` follows standard Unix convention for accessibility

---

## Interactive Prompts

When required values are missing and script is running interactively (not `--non-interactive`), the script prompts for input:

### Cluster Prompts

```
Create new AKS cluster or use existing? [create/existing]: create
Cluster name (DNS-compatible, lowercase): my-kalypso
Azure region [eastus]: westus2
Number of nodes [3]: 5
VM size [Standard_DS2_v2]: Standard_D4s_v3
```

### Repository Prompts

```
Create new control-plane repository or use existing? [create/existing]: create
Control-plane repository name [my-kalypso-control-plane]: 
Create in organization or user account? [org/user]: user
Repository visibility [private/public]: private

Create new gitops repository or use existing? [create/existing]: existing
GitOps repository URL: https://github.com/myorg/my-gitops
```

### Installation Prompts

```
Kubernetes namespace [kalypso-system]: 
Install Kalypso Scheduler? [Y/n]: Y
```

### Confirmation Prompt

Before executing operations, the script displays the effective configuration and asks for confirmation:

```
=== Bootstrap Configuration ===
Cluster:
  Name: my-kalypso
  Create: true
  Region: westus2
  Nodes: 5
  VM Size: Standard_D4s_v3

Repositories:
  Control-Plane: Create new (my-kalypso-control-plane)
  GitOps: Use existing (https://github.com/myorg/my-gitops)

Installation:
  Namespace: kalypso-system
  Helm Chart: ./helm/kalypso-scheduler

Proceed with bootstrap? [y/N]: y
```

**Notes**:
- Use `--skip-confirmation` to bypass this prompt
- Default answers are shown in brackets `[default]`
- Press Ctrl+C to cancel at any time

---

## Output Format

### Progress Indicators

```
[INFO]  2025-11-11 10:30:45 - Starting Kalypso Scheduler bootstrap
[INFO]  2025-11-11 10:30:45 - Checking prerequisites...
[INFO]  2025-11-11 10:30:46 - ✓ All prerequisites met
[INFO]  2025-11-11 10:30:46 - Creating AKS cluster 'my-kalypso'...
[INFO]  2025-11-11 10:35:30 - ✓ Cluster created successfully
[INFO]  2025-11-11 10:35:31 - Creating control-plane repository...
[INFO]  2025-11-11 10:35:35 - ✓ Repository created: https://github.com/myuser/my-kalypso-control-plane
[INFO]  2025-11-11 10:35:36 - Installing Kalypso Scheduler...
[INFO]  2025-11-11 10:37:00 - ✓ Kalypso Scheduler installed successfully
[INFO]  2025-11-11 10:37:01 - Running health checks...
[INFO]  2025-11-11 10:37:05 - ✓ All health checks passed

=== Bootstrap Complete ===
Cluster: my-kalypso (eastus)
Control-Plane Repo: https://github.com/myuser/my-kalypso-control-plane
GitOps Repo: https://github.com/myuser/my-kalypso-gitops
Namespace: kalypso-system
Kubeconfig Context: my-kalypso

Next Steps:
1. Configure kubectl: kubectl config use-context my-kalypso
2. View pods: kubectl get pods -n kalypso-system
3. Create your first workload: See docs/getting-started.md

Log file: /Users/user/.kalypso/bootstrap-1699700345.log
State file: /Users/user/.kalypso/bootstrap-state.json
```

### Error Output

```
[ERROR] 2025-11-11 10:36:00 - Failed to create repository: API rate limit exceeded
[ERROR] 2025-11-11 10:36:00 - Bootstrap failed at step: repository creation
[ERROR] 2025-11-11 10:36:00 - See log file for details: /Users/user/.kalypso/bootstrap-1699700345.log

Rollback created resources? [y/N]: y

[INFO]  2025-11-11 10:36:15 - Rolling back...
[INFO]  2025-11-11 10:36:16 - Deleting AKS cluster 'my-kalypso'...
[WARN]  2025-11-11 10:40:00 - Cluster deletion initiated (async operation)
[INFO]  2025-11-11 10:40:01 - Rollback complete

Exit code: 5 (Repository operation failed)
```

---

## Usage Examples

### Example 1: Create Everything New

```bash
./scripts/bootstrap/bootstrap.sh \
  --cluster-name dev-kalypso \
  --cluster-region eastus \
  --control-plane-create \
  --gitops-create
```

### Example 2: Use Existing Cluster and Repos

```bash
./scripts/bootstrap/bootstrap.sh \
  --cluster-existing my-k8s-cluster \
  --control-plane-repo https://github.com/myorg/control-plane \
  --gitops-repo https://github.com/myorg/gitops
```

### Example 3: Create in Organization with Custom Settings

```bash
./scripts/bootstrap/bootstrap.sh \
  --cluster-name prod-kalypso \
  --cluster-region westus2 \
  --cluster-nodes 10 \
  --cluster-vm-size Standard_D8s_v3 \
  --github-org mycompany \
  --namespace kalypso-production \
  --helm-values ./prod-values.yaml \
  --repo-visibility private
```

### Example 4: CI/CD Non-Interactive Mode

```bash
# Export credentials
export AZURE_SUBSCRIPTION_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxx"

# Run with config file
./scripts/bootstrap/bootstrap.sh \
  --config ./ci-bootstrap-config.yaml \
  --non-interactive \
  --auto-rollback \
  --no-color \
  --log-file ./bootstrap-ci.log
```

### Example 5: Validate Prerequisites Only

```bash
./scripts/bootstrap/bootstrap.sh --validate-only
```

### Example 6: Check Status and Clean Up

```bash
# Check what's been created
./scripts/bootstrap/bootstrap.sh --status

# Clean up all resources
./scripts/bootstrap/bootstrap.sh --clean
```

---

## State File Structure

The state file (`$HOME/.kalypso/bootstrap-state.json`) has the following structure:

```json
{
  "version": "1.0",
  "timestamp": "2025-11-11T10:30:00Z",
  "status": "completed",
  "cluster": {
    "created": true,
    "name": "my-kalypso",
    "resourceGroup": "my-kalypso-rg",
    "region": "eastus",
    "kubeContext": "my-kalypso"
  },
  "repositories": {
    "controlPlane": {
      "created": true,
      "name": "my-kalypso-control-plane",
      "owner": "myuser",
      "url": "https://github.com/myuser/my-kalypso-control-plane"
    },
    "gitops": {
      "created": true,
      "name": "my-kalypso-gitops",
      "owner": "myuser",
      "url": "https://github.com/myuser/my-kalypso-gitops"
    }
  },
  "installation": {
    "namespace": "kalypso-system",
    "helmRelease": "kalypso-scheduler",
    "version": "0.1.0"
  }
}
```

This state file is used for:
- Idempotency checks (don't recreate existing resources)
- Rollback operations (know what to delete)
- Status queries (`--status` command)
- Cleanup operations (`--clean` command)

---

## CLI Contract Complete

This specification defines the complete command-line interface for the bootstrapping script, including all options, validation rules, examples, and expected behavior.
