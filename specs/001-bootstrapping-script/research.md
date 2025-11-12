# Research: Bootstrapping Script

**Feature**: 001-bootstrapping-script  
**Date**: 2025-11-11  
**Status**: Phase 0 Complete

## Overview

This document captures all technical decisions, research findings, and rationale for the bootstrapping script implementation. Each decision includes alternatives considered and reasons for the chosen approach.

---

## Decision 1: Language Choice - Bash

**Decision**: Use Bash 4.0+ for the bootstrapping script

**Rationale**:
- **Ubiquity**: Bash is pre-installed on all target platforms (macOS, Linux)
- **Simplicity**: Direct integration with command-line tools (az, kubectl, gh, git)
- **Transparency**: Platform engineers can read and understand the script easily
- **Debugging**: Easy to test individual commands and functions
- **No compilation**: Can be executed directly without build steps
- **Shell best practices**: Well-established patterns for error handling, validation, logging

**Alternatives Considered**:
1. **Python**: 
   - Rejected: Adds dependency management complexity (virtualenv, pip packages)
   - Not pre-installed on all systems
   - Overkill for primarily command orchestration tasks
   
2. **Go**:
   - Rejected: Requires compilation and distribution of binaries
   - Adds complexity for what is essentially CLI orchestration
   - Harder for platform engineers to customize/debug
   
3. **Node.js**:
   - Rejected: Not typically pre-installed on enterprise systems
   - Adds npm dependency management overhead

**Implementation Notes**:
- Use `set -euo pipefail` for strict error handling
- Use shellcheck for static analysis
- Follow Google Shell Style Guide
- Support both Bash 4.x (Linux) and Bash 5.x (macOS with Homebrew)

---

## Decision 2: Credential Management - Environment Variables with Fallback

**Decision**: Primary credential source is environment variables (AZURE_TOKEN, GITHUB_TOKEN), with fallback to interactive prompts

**Rationale**:
- **CI/CD compatible**: Non-interactive mode can use env vars
- **Security**: Credentials not stored in script or config files
- **Flexibility**: Interactive mode for manual runs, env vars for automation
- **Standard practice**: Follows 12-factor app principles
- **Tool integration**: Azure CLI and GitHub CLI respect their own token mechanisms

**Alternatives Considered**:
1. **Configuration file**:
   - Rejected: Security risk (credentials on disk)
   - Requires secure file permissions management
   
2. **Always interactive**:
   - Rejected: Cannot run in CI/CD pipelines
   - Not suitable for automated deployments
   
3. **OAuth flows**:
   - Rejected: Too complex for a bootstrapping script
   - Requires additional dependencies

**Implementation Notes**:
- Check for `AZURE_SUBSCRIPTION_ID`, `AZURE_TENANT_ID` environment variables
- For GitHub token, check `GITHUB_TOKEN` or use `gh auth status`
- If env vars missing and interactive mode, prompt with clear instructions
- In non-interactive mode, exit with error if credentials missing
- Never log or echo credential values

---

## Decision 3: Azure CLI for Cluster Management

**Decision**: Use Azure CLI (`az`) commands for all AKS cluster operations

**Rationale**:
- **Official tool**: Microsoft's supported interface for Azure
- **Complete feature set**: All AKS operations available
- **JSON output**: Easy to parse with `jq`
- **Wide adoption**: Platform engineers already familiar with az CLI
- **Authentication**: Built-in auth flows (device code, service principal, managed identity)

**Alternatives Considered**:
1. **Terraform**:
   - Rejected: Overkill for simple cluster creation
   - Would require state management
   - Adds another tool dependency
   
2. **Azure REST API**:
   - Rejected: Too low-level, requires manual auth handling
   - More complex than CLI

**Implementation Notes**:
- Verify `az` CLI version >= 2.50.0 (for latest AKS features)
- Use `az aks create` with explicit parameters for reproducibility
- Default cluster config: Standard_DS2_v2 VMs, 3 nodes, system node pool
- Use `az aks get-credentials` to merge kubeconfig
- Set `--output json` and parse with `jq` for reliable output parsing

---

## Decision 4: GitHub CLI for Repository Management

**Decision**: Use GitHub CLI (`gh`) for repository creation and management

**Rationale**:
- **Official tool**: GitHub's supported CLI
- **Simplified auth**: Integrated authentication flow
- **Idempotency support**: Can check if repo exists before creating
- **Template support**: Can clone from templates or create empty repos
- **PR automation**: Can create PRs programmatically (useful for future enhancements)

**Alternatives Considered**:
1. **Git + GitHub API (curl)**:
   - Rejected: More complex auth handling
   - Would need to implement retry logic, pagination, etc.
   
2. **Direct git operations**:
   - Rejected: Cannot create GitHub repos, only push to existing ones
   
3. **Hub (predecessor to gh)**:
   - Rejected: Deprecated in favor of gh CLI

**Implementation Notes**:
- Verify `gh` CLI version >= 2.0.0
- Use `gh repo create` with `--public` or `--private` flag (configurable)
- Use `gh repo clone` to populate templates
- Check authentication with `gh auth status` before operations
- Support `--org` flag to create repos in organizations vs. user account

---

## Decision 5: Helm for Kalypso Scheduler Installation

**Decision**: Use Helm chart to install Kalypso Scheduler on the cluster

**Rationale**:
- **Existing chart**: Project already has `helm/kalypso-scheduler/` chart
- **Configuration management**: Helm values files for customization
- **Kubernetes best practice**: Standard deployment method for operators
- **Upgrades**: Helm tracks releases and supports upgrades/rollbacks
- **Dependencies**: Can manage Flux installation as dependency if needed

**Alternatives Considered**:
1. **kubectl apply -f**:
   - Rejected: No release tracking or rollback capability
   - Would need to manually manage CRDs, namespaces, etc.
   
2. **Kustomize**:
   - Rejected: Less flexible for configuration variants
   - No release management

**Implementation Notes**:
- Check for `helm` CLI >= 3.0.0
- Install from local chart path or remote repository (configurable)
- Create namespace if not exists: `kubectl create namespace kalypso-system`
- Use `helm upgrade --install` for idempotency
- Wait for deployment rollout: `helm wait --timeout=5m`
- Store Helm values in generated config for reproducibility

---

## Decision 6: Idempotency Strategy

**Decision**: Implement idempotency by checking resource existence before creation

**Rationale**:
- **Safe re-runs**: Script can be run multiple times without errors
- **Resume capability**: Can continue from failure point
- **Debugging**: Can test individual steps without full cleanup
- **User experience**: Reduces anxiety about running script multiple times

**Implementation Approach**:
- **Cluster**: Check with `az aks show` before `az aks create`
- **Repositories**: Check with `gh repo view` before `gh repo create`
- **Helm release**: Check with `helm list` before `helm upgrade --install`
- **Kubeconfig**: Check if context exists before fetching credentials
- Use state file (`.bootstrap-state.json`) to track created resources

**Alternatives Considered**:
1. **Always fail on existing resources**:
   - Rejected: Poor user experience, requires manual cleanup
   
2. **Always delete and recreate**:
   - Rejected: Dangerous, could destroy user data
   
3. **No idempotency**:
   - Rejected: Makes debugging and development difficult

**Implementation Notes**:
- Store state in `$HOME/.kalypso/bootstrap-state.json`
- Include creation timestamps and resource IDs
- Provide `--force` flag to ignore state and recreate
- Provide `--clean` flag to delete all tracked resources

---

## Decision 7: Error Handling and Rollback

**Decision**: Implement explicit error handling with cleanup of resources created in current run

**Rationale**:
- **Safety**: Don't leave partial installations that could confuse users
- **Cost control**: Don't leave Azure resources running and accumulating charges
- **Clean slate**: Allow users to retry after fixing issues
- **Clarity**: Failed run should not interfere with next run

**Rollback Strategy**:
1. Track all created resources in state file
2. On error, prompt user: "Rollback created resources? [y/N]"
3. In non-interactive mode, rollback automatically if `--auto-rollback` flag set
4. Delete in reverse order of creation:
   - Helm release (if installed)
   - Kubeconfig context (if added)
   - AKS cluster (if created)
   - GitHub repositories (if created - with confirmation!)
5. Remove state file after successful rollback

**Alternatives Considered**:
1. **No rollback**:
   - Rejected: Leaves users with cleanup burden
   
2. **Always rollback on error**:
   - Rejected: User might want to inspect failed state for debugging
   
3. **Transaction-like with savepoints**:
   - Rejected: Too complex for bash script, cloud APIs don't support transactions

**Implementation Notes**:
- Use trap to catch errors: `trap 'handle_error $?' ERR`
- Implement `cleanup()` function that reads state file
- Provide `--skip-rollback` flag to disable rollback
- Log all cleanup actions for transparency
- For repositories, ask user to confirm deletion (can contain user data)

---

## Decision 8: Template Structure for Minimal Repositories

**Decision**: Embed YAML templates in script's `templates/` directory, populate using `envsubst` or `sed`

**Rationale**:
- **Self-contained**: No external dependencies for templates
- **Version controlled**: Templates evolve with script
- **Simple substitution**: Bash-native string replacement
- **Validation**: Templates can be validated before release

**Control-Plane Repository Template**:
```
templates/control-plane/
├── main/                         # Main branch (promotion source)
│   ├── .environments/
│   │   └── dev.yaml             # Environment definition
│   ├── .github/
│   │   └── workflows/
│   │       ├── ci.yaml          # CI workflow (quality checks)
│   │       ├── cd.yaml          # CD workflow (promotion)
│   │       └── check-promote.yaml # Post-deployment validation
│   ├── templates/
│   │   └── default-template.yaml # Default Kalypso template
│   ├── workloads/
│   │   └── sample-workload-registration.yaml # Example workload
│   └── README.md                # Main branch documentation
├── dev/                         # Dev branch (environment-specific)
│   ├── cluster-types/
│   │   └── default.yaml         # Cluster type definition
│   ├── configs/
│   │   └── default-config.yaml  # Environment config
│   ├── scheduling-policies/
│   │   └── default-policy.yaml  # Scheduling policy
│   ├── base-repo.yaml           # Reference to main branch commit
│   ├── gitops-repo.yaml         # Reference to GitOps repo
│   └── README.md                # Dev branch documentation
```

**GitOps Repository Template**:
```
templates/gitops/
├── main/                        # Main branch (minimal/empty)
│   └── README.md               # Explains repo purpose
├── dev/                        # Dev branch (environment-specific)
│   ├── .github/
│   │   └── workflows/
│   │       └── check-promote.yaml # Workflow triggered by PRs from control-plane
│   └── README.md               # Dev branch documentation
```

**Alternatives Considered**:
1. **Fetch templates from remote repo**:
   - Rejected: Creates external dependency, network requirement
   
2. **Generate templates programmatically**:
   - Rejected: Harder to maintain, harder to validate
   
3. **Use Helm/Kustomize for templates**:
   - Rejected: Overkill for simple YAML files

**Implementation Notes**:
- Use environment variable substitution for dynamic values
- Variables: `${CLUSTER_NAME}`, `${NAMESPACE}`, `${ENVIRONMENT}`, `${GITHUB_ORG}`
- Validate YAML syntax with `kubectl --dry-run=client`
- Include comments in templates explaining fields

---

## Decision 9: Validation Strategy

**Decision**: Multi-level validation: prerequisites, inputs, resources, installation

**Validation Levels**:

1. **Prerequisites** (before any operations):
   - Check for required CLIs: `az`, `kubectl`, `git`, `gh`, `helm`, `jq`
   - Check CLI versions meet minimum requirements
   - Check authentication status for Azure and GitHub
   - Check disk space for kubeconfig and repos
   
2. **Input validation**:
   - Validate cluster name format (DNS-compatible)
   - Validate repository names (GitHub naming rules)
   - Validate Azure region availability
   - Validate resource group name
   
3. **Resource validation** (during operations):
   - Verify cluster is running before attempting installation
   - Verify repositories are accessible and cloneable
   - Verify namespace exists before Helm install
   
4. **Installation validation** (post-installation):
   - Check all pods are Running: `kubectl get pods -n kalypso-system`
   - Check CRDs are registered: `kubectl get crds | grep scheduler.kalypso`
   - Verify basic functionality: create a test Workload CRD
   - Check Flux components if applicable

**Rationale**:
- **Early failure**: Catch issues before costly operations
- **Clear errors**: Specific messages about what's wrong
- **User guidance**: Tell users how to fix validation failures
- **Confidence**: Comprehensive validation ensures successful installation

**Implementation Notes**:
- Create `validate_prerequisites()` function
- Exit early with exit code 1 if validation fails
- Use colored output: red for errors, yellow for warnings, green for success
- Provide `--skip-validation` flag for advanced users (dangerous!)

---

## Decision 10: Logging and Output

**Decision**: Structured logging with multiple verbosity levels and log file

**Logging Approach**:
- **Log levels**: ERROR, WARN, INFO, DEBUG
- **Default**: INFO level (shows major steps and success/failure)
- **Verbose mode**: DEBUG level with `--verbose` or `-v` flag
- **Quiet mode**: Only errors with `--quiet` or `-q` flag
- **Log file**: Always write full DEBUG log to `$HOME/.kalypso/bootstrap-$(date +%s).log`

**Output Format**:
```
[INFO]  2025-11-11 10:30:45 - Checking prerequisites...
[DEBUG] 2025-11-11 10:30:45 - Found az CLI version 2.54.0
[INFO]  2025-11-11 10:30:46 - ✓ All prerequisites met
[INFO]  2025-11-11 10:30:47 - Creating AKS cluster 'my-cluster'...
[WARN]  2025-11-11 10:32:15 - Cluster creation taking longer than expected
[INFO]  2025-11-11 10:35:30 - ✓ Cluster created successfully
[ERROR] 2025-11-11 10:36:00 - Failed to create repository: API rate limit exceeded
```

**Rationale**:
- **Debugging**: Full logs help troubleshoot failures
- **User experience**: Clean output during normal operation
- **Compliance**: Audit trail of bootstrap operations
- **Support**: Users can share log files when asking for help

**Implementation Notes**:
- Implement `log()` function with level parameter
- Use ANSI color codes for terminal output (with `--no-color` flag)
- Redirect all command output to log file, show summary in terminal
- Include timing information for long operations
- Provide log file path at script completion

---

## Decision 11: Configuration Options

**Decision**: Support three configuration methods: CLI flags, config file, interactive prompts

**Priority Order**:
1. CLI flags (highest priority)
2. Configuration file
3. Interactive prompts (only if missing and tty available)
4. Defaults

**CLI Flags**:
```bash
bootstrap.sh \
  --cluster-name NAME \
  --cluster-create | --cluster-existing CONTEXT \
  --cluster-region REGION \
  --cluster-nodes COUNT \
  --cluster-vm-size SIZE \
  --control-plane-repo URL | --control-plane-create \
  --gitops-repo URL | --gitops-create \
  --github-org ORG \
  --namespace NAMESPACE \
  --non-interactive \
  --auto-rollback \
  --verbose | --quiet \
  --config FILE
```

**Config File Format** (YAML):
```yaml
cluster:
  name: my-kalypso-cluster
  create: true  # or false for existing
  region: eastus
  nodeCount: 3
  vmSize: Standard_DS2_v2
  
repositories:
  controlPlane:
    create: true
    url: ""  # if create: false
  gitops:
    create: true
    url: ""  # if create: false
  githubOrg: ""  # optional
  
installation:
  namespace: kalypso-system
  helmChart: ./helm/kalypso-scheduler
  
options:
  nonInteractive: false
  autoRollback: false
  verbose: false
```

**Rationale**:
- **Flexibility**: Different users have different preferences
- **Repeatability**: Config file enables identical repeated runs
- **CI/CD**: Non-interactive mode with CLI flags for automation
- **Discoverability**: Interactive mode for learning

**Implementation Notes**:
- Use `getopts` for flag parsing (portable)
- Use `yq` or `jq` for YAML parsing if config file provided
- Validate all configuration before starting operations
- Print effective configuration before proceeding (with confirmation)

---

## Decision 12: Documentation Structure

**Decision**: Generate comprehensive BOOTSTRAP.md manual during script development

**Documentation Sections**:

1. **Quick Start**: 3-minute speedrun for experienced users
2. **Prerequisites**: Detailed requirements and installation links
3. **Authentication**: How to set up Azure and GitHub credentials
4. **Usage Scenarios**:
   - Create everything new
   - Bring your own cluster
   - Bring your own repositories
   - Combination scenarios
5. **Configuration Reference**: All CLI flags and config file options
6. **Architecture**: What the script creates and why
7. **Troubleshooting**: Common errors and solutions
8. **Advanced Topics**:
   - Running in CI/CD
   - Custom Helm values
   - Multi-cluster setups
9. **Uninstallation**: How to clean up resources
10. **FAQ**: Frequently asked questions

**Rationale**:
- **Self-service**: Reduces support burden
- **Onboarding**: New users can get started quickly
- **Reference**: Comprehensive flag documentation
- **Troubleshooting**: Common issues documented upfront

**Implementation Notes**:
- Write documentation first (README-driven development)
- Include code examples for each scenario
- Link to external docs (Azure, GitHub, Kubernetes) where appropriate
- Keep quick start under 10 commands
- Include screenshots or ASCII diagrams for architecture

---

## Decision 13: Platform Support - macOS and Linux

**Decision**: Support macOS (Darwin) and Linux (Debian/Ubuntu, RHEL/CentOS/Fedora)

**Platform-Specific Considerations**:

**macOS**:
- Default Bash version: 3.2 (too old), require Bash 4+ from Homebrew
- Azure CLI: Install via Homebrew (`brew install azure-cli`)
- GitHub CLI: Install via Homebrew (`brew install gh`)
- Detection: `uname -s` returns "Darwin"

**Linux**:
- Bash 4+ typically included
- Azure CLI: Install via package manager or install script
- GitHub CLI: Install via package manager or GitHub releases
- Detection: `uname -s` returns "Linux"

**Cross-Platform Utilities**:
- Use `kubectl`, `helm` binaries (same on both platforms)
- Avoid GNU-specific flags (use portable options)
- Test `sed` and `awk` commands on both platforms
- Use `$(command)` not backticks for command substitution

**Rationale**:
- **Coverage**: Addresses both primary developer platforms
- **Practicality**: >90% of users on these platforms
- **Maintainability**: Keep matrix small and testable

**Explicitly Not Supported**:
- Windows (users can use WSL2 for Linux environment)
- BSD variants (niche use case)

**Implementation Notes**:
- Detect OS with `OS=$(uname -s)`
- Check Bash version: `${BASH_VERSINFO[0]} -ge 4`
- Document Homebrew installation for macOS users
- Provide OS-specific installation commands in docs
- Test on: macOS 13+, Ubuntu 22.04+, RHEL 9+

---

## Decision 14: State Management

**Decision**: Use JSON state file in `$HOME/.kalypso/bootstrap-state.json`

**State File Schema**:
```json
{
  "version": "1.0",
  "timestamp": "2025-11-11T10:30:00Z",
  "cluster": {
    "created": true,
    "name": "my-kalypso-cluster",
    "resourceGroup": "kalypso-rg",
    "region": "eastus",
    "kubeContext": "my-kalypso-cluster"
  },
  "repositories": {
    "controlPlane": {
      "created": true,
      "name": "my-control-plane",
      "owner": "myuser",
      "url": "https://github.com/myuser/my-control-plane"
    },
    "gitops": {
      "created": true,
      "name": "my-gitops",
      "owner": "myuser",
      "url": "https://github.com/myuser/my-gitops"
    }
  },
  "installation": {
    "namespace": "kalypso-system",
    "helmRelease": "kalypso-scheduler",
    "version": "0.1.0"
  }
}
```

**State File Usage**:
- **Idempotency**: Check state before creating resources
- **Rollback**: Know what to delete if cleanup requested
- **Resume**: Continue from last successful step
- **Audit**: Record of what was created

**Rationale**:
- **Persistence**: Survives script termination
- **Portability**: JSON is universal
- **Tooling**: Easy to parse with `jq`
- **Human-readable**: Users can inspect state

**Implementation Notes**:
- Create state directory on first run: `mkdir -p $HOME/.kalypso`
- Update state atomically (write to temp file, then move)
- Validate JSON before writing with `jq`
- Provide `--state-file` flag to override location
- Add `bootstrap.sh --status` command to show current state
- Add `bootstrap.sh --clean` to remove state and created resources

---

## Summary of Key Technologies

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| Script Language | Bash | 4.0+ | Main implementation |
| Static Analysis | ShellCheck | 0.9+ | Linting and best practices |
| Azure Management | Azure CLI | 2.50+ | Cluster operations |
| GitHub Management | GitHub CLI | 2.0+ | Repository operations |
| Kubernetes | kubectl | 1.25+ | Cluster interaction |
| Package Management | Helm | 3.0+ | Kalypso installation |
| JSON Processing | jq | 1.6+ | State and output parsing |
| Documentation | Markdown | N/A | User manual |

---

## Open Questions and Future Research

1. **Multi-cluster support**: How should the script handle creating multiple clusters?
   - Current: Single cluster per run
   - Future: Config file with multiple cluster definitions?

2. **Flux installation**: Should script install Flux if not present?
   - Current: Assume Flux installed separately or as Helm dependency
   - Future: Detect and install Flux automatically?

3. **Repository templates**: Should we support custom template repositories?
   - Current: Embedded templates in script
   - Future: `--template-repo` flag to clone from custom template?

4. **Upgrades**: How to handle upgrading existing installations?
   - Current: Out of scope for bootstrapping
   - Future: Separate `upgrade.sh` script?

5. **Multi-tenancy**: How to support multiple Kalypso installations in one cluster?
   - Current: Single installation per cluster
   - Future: Namespace isolation strategy?

---

## Research Complete

All technical decisions have been documented with rationale and alternatives. Implementation can proceed to Phase 1 (Data Model and Contracts).
