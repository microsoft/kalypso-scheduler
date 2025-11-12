# Data Model: Bootstrapping Script

**Feature**: 001-bootstrapping-script  
**Date**: 2025-11-11  
**Status**: Phase 1 Complete

## Overview

The bootstrapping script manages several key entities and their relationships. This document defines the data structures, state transitions, and validation rules for each entity.

---

## Entity: Configuration

**Description**: Represents the user's desired bootstrap configuration, collected from CLI flags, config file, or interactive prompts.

### Fields

| Field | Type | Required | Default | Validation |
|-------|------|----------|---------|------------|
| `cluster.name` | string | Yes | - | Must match `^[a-z0-9-]{1,63}$` (DNS-compatible) |
| `cluster.create` | boolean | Yes | `true` | - |
| `cluster.existing_context` | string | Conditional | - | Required if `create=false`, must exist in kubeconfig |
| `cluster.region` | string | Conditional | `eastus` | Required if `create=true`, must be valid Azure region |
| `cluster.resource_group` | string | Conditional | `{cluster.name}-rg` | Required if `create=true`, matches `^[a-zA-Z0-9-_()]{1,90}$` |
| `cluster.node_count` | integer | No | `3` | Min: 1, Max: 100 |
| `cluster.vm_size` | string | No | `Standard_DS2_v2` | Must be valid Azure VM SKU |
| `repositories.control_plane.create` | boolean | Yes | `true` | - |
| `repositories.control_plane.url` | string | Conditional | - | Required if `create=false`, must be valid GitHub URL |
| `repositories.gitops.create` | boolean | Yes | `true` | - |
| `repositories.gitops.url` | string | Conditional | - | Required if `create=false`, must be valid GitHub URL |
| `repositories.github_org` | string | No | - | If set, must be valid GitHub org name |
| `installation.namespace` | string | No | `kalypso-system` | Must match Kubernetes namespace rules |
| `installation.helm_chart` | string | No | `./helm/kalypso-scheduler` | Must be valid path or Helm repo |
| `options.non_interactive` | boolean | No | `false` | - |
| `options.auto_rollback` | boolean | No | `false` | - |
| `options.verbose` | boolean | No | `false` | - |
| `options.skip_validation` | boolean | No | `false` | - |

### State Transitions

1. **Uninitialized** → **Parsed**: CLI flags parsed and config file loaded
2. **Parsed** → **Validated**: All validation rules pass
3. **Validated** → **Confirmed**: User confirms configuration (in interactive mode)
4. **Confirmed** → **Active**: Configuration is used for resource creation

### Validation Rules

- If `cluster.create=false`, then `cluster.existing_context` must be provided
- If `cluster.create=true`, then `cluster.region` and `cluster.resource_group` must be set
- If `repositories.control_plane.create=false`, then `repositories.control_plane.url` must be valid and accessible
- If `repositories.gitops.create=false`, then `repositories.gitops.url` must be valid and accessible
- If `options.non_interactive=true`, then all required fields without defaults must be provided
- `cluster.name` must not conflict with existing clusters in the same Azure subscription
- Repository names derived from configuration must be unique within the target GitHub user/org

---

## Entity: Bootstrap State

**Description**: Persisted state tracking resources created during bootstrap process, enabling idempotency and rollback.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | Yes | State schema version (currently "1.0") |
| `timestamp` | ISO8601 string | Yes | When bootstrap started |
| `status` | enum | Yes | One of: `in_progress`, `completed`, `failed`, `rolled_back` |
| `cluster.created` | boolean | Yes | Whether cluster was created by this script |
| `cluster.name` | string | No | Cluster name (if created) |
| `cluster.resource_group` | string | No | Azure resource group (if created) |
| `cluster.region` | string | No | Azure region (if created) |
| `cluster.kube_context` | string | No | Kubeconfig context name (if added) |
| `repositories.control_plane.created` | boolean | Yes | Whether control-plane repo was created |
| `repositories.control_plane.name` | string | No | Repository name (if created) |
| `repositories.control_plane.owner` | string | No | GitHub owner/org (if created) |
| `repositories.control_plane.url` | string | No | Full repository URL (if created) |
| `repositories.gitops.created` | boolean | Yes | Whether gitops repo was created |
| `repositories.gitops.name` | string | No | Repository name (if created) |
| `repositories.gitops.owner` | string | No | GitHub owner/org (if created) |
| `repositories.gitops.url` | string | No | Full repository URL (if created) |
| `installation.namespace` | string | No | Kubernetes namespace (if created) |
| `installation.helm_release` | string | No | Helm release name (if installed) |
| `installation.version` | string | No | Installed Kalypso version |
| `errors` | array[string] | No | Error messages if status is `failed` |

### State Transitions

1. **Not Exists** → **In Progress**: State file created when bootstrap starts
2. **In Progress** → **Completed**: All steps successful
3. **In Progress** → **Failed**: Any step fails
4. **Failed** → **Rolled Back**: Cleanup executed successfully
5. **Completed** → **Rolled Back**: User manually runs cleanup

### Persistence

- **Location**: `$HOME/.kalypso/bootstrap-state.json`
- **Format**: JSON with indentation for human readability
- **Atomicity**: Updates written to temp file then moved atomically
- **Backup**: Previous state backed up to `.bootstrap-state.json.bak` before update

---

## Entity: AKS Cluster

**Description**: Azure Kubernetes Service cluster where Kalypso Scheduler will be installed.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Cluster name (DNS-compatible) |
| `resource_group` | string | Yes | Azure resource group |
| `region` | string | Yes | Azure region (e.g., `eastus`, `westus2`) |
| `node_count` | integer | Yes | Initial node count |
| `vm_size` | string | Yes | VM SKU for nodes |
| `kubernetes_version` | string | No | Kubernetes version (defaults to latest stable) |
| `network_plugin` | string | No | Azure CNI or kubenet (default: `azure`) |
| `subscription_id` | string | Yes | Azure subscription ID |

### State Transitions

1. **Not Exists** → **Creating**: `az aks create` initiated
2. **Creating** → **Running**: Cluster provisioning complete
3. **Running** → **Accessible**: Kubeconfig merged, kubectl can connect
4. **Accessible** → **Ready**: All system pods running
5. **Ready** → **Deleting**: Cleanup initiated
6. **Deleting** → **Deleted**: `az aks delete` complete

### Validation Rules

- Name must be unique within resource group
- Resource group must exist or be created
- Region must support AKS
- VM size must be available in region
- Node count must be >= 1
- Subscription must have sufficient quota for requested resources

### External Dependencies

- Azure subscription with sufficient permissions (Contributor or Owner)
- Azure CLI authenticated
- Sufficient quota for VM size and node count

---

## Entity: GitHub Repository

**Description**: Git repository for either control-plane configuration or gitops delivery.

### Types

1. **Control-Plane Repository**: Contains Kalypso CRDs (Environments, ClusterTypes, SchedulingPolicies, ConfigMaps)
2. **GitOps Repository**: Contains generated deployment manifests and CI/CD workflows

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Repository name |
| `owner` | string | Yes | GitHub user or organization |
| `type` | enum | Yes | `control_plane` or `gitops` |
| `visibility` | enum | Yes | `public` or `private` (default: `private`) |
| `url` | string | Generated | Full HTTPS URL (e.g., `https://github.com/owner/name`) |
| `clone_url` | string | Generated | Clone URL for git operations |
| `default_branch` | string | No | Default branch name (default: `main`) |

### State Transitions

**For Created Repositories:**
1. **Not Exists** → **Creating**: `gh repo create` initiated
2. **Creating** → **Empty**: Repository created but no files
3. **Empty** → **Initialized**: Initial commit pushed with template files
4. **Initialized** → **Ready**: README and structure in place

**For Existing Repositories:**
1. **Unknown** → **Validating**: Checking if URL is accessible
2. **Validating** → **Valid**: Repository structure meets requirements
3. **Validating** → **Invalid**: Repository lacks required structure

### Validation Rules

**Control-Plane Repository**:
- Must be accessible via `gh` CLI or git
- Script will create two branches: `main` and `dev`
- Main branch will contain:
  - `.environments/` directory with environment definitions
  - `.github/workflows/` directory with CI/CD workflows (ci.yaml, cd.yaml, check-promote.yaml)
  - `templates/` directory with Kalypso templates
  - `workloads/` directory with sample workload registrations
- Dev branch will contain:
  - `cluster-types/` directory with cluster type definitions
  - `configs/` directory with environment-specific configs
  - `scheduling-policies/` directory with scheduling policies
  - `base-repo.yaml` file (reference to main branch commit)
  - `gitops-repo.yaml` file (reference to GitOps repository)

**GitOps Repository**:
- Must be accessible via `gh` CLI or git
- Script will create two branches: `main` and `dev`
- Main branch will contain:
  - `README.md` documentation
- Dev branch will contain:
  - `.github/workflows/check-promote.yaml` workflow
  - Cluster type directories (created by Kalypso, not by bootstrap script)

### Template Content

**Control-Plane Repository Template (Main Branch)**:
```
.environments/
  dev.yaml                # Environment definition
.github/
  workflows/
    ci.yaml              # CI workflow (quality checks, triggers CD)
    cd.yaml              # CD workflow (promotes to dev by updating base-repo.yaml)
    check-promote.yaml   # Post-deployment validation
templates/
  default-template.yaml  # Default Kalypso template for workload transformation
workloads/
  sample-workload-registration.yaml # Example workload registration
README.md                # Repository documentation and structure explanation
```

**Control-Plane Repository Template (Dev Branch)**:
```
cluster-types/
  default.yaml           # Default cluster type definition
configs/
  default-config.yaml    # Default configuration
scheduling-policies/
  default-policy.yaml    # Default scheduling policy
base-repo.yaml           # Reference to main branch commit SHA
gitops-repo.yaml         # Reference to GitOps repository URL and details
README.md                # Dev branch documentation
```

**GitOps Repository Template**:
```
.github/
  workflows/
    check-promote.yaml   # Workflow triggered by PR merges from control-plane
README.md                # Repository documentation and purpose
```

Note: Cluster type directories (e.g., `small/`, `large/`, `drone/`) are created by Kalypso Scheduler during operation, not by the bootstrap script.

---

## Entity: Kalypso Installation

**Description**: Deployed Kalypso Scheduler operator and associated resources on the cluster.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `namespace` | string | Yes | Kubernetes namespace for installation |
| `helm_release` | string | Yes | Helm release name |
| `helm_chart` | string | Yes | Chart location (path or repo/chart) |
| `version` | string | No | Chart version (defaults to latest) |
| `values` | map | No | Custom Helm values |

### State Transitions

1. **Not Installed** → **Installing**: `helm upgrade --install` initiated
2. **Installing** → **Deployed**: Helm reports release as deployed
3. **Deployed** → **Running**: All pods in Running state
4. **Running** → **Healthy**: All health checks pass
5. **Healthy** → **Verified**: Test workload processed successfully
6. **Any** → **Uninstalling**: `helm uninstall` initiated
7. **Uninstalling** → **Uninstalled**: Namespace and CRDs removed

### Validation Rules

- Namespace must exist or be created
- Cluster must be accessible via kubectl
- Helm chart must be valid and available
- All CRDs must install successfully
- All deployments must reach Ready state within timeout (default: 5 minutes)

### Health Checks

1. **Pod Health**: All pods in namespace are Running
2. **CRD Registration**: All Kalypso CRDs are registered (check with `kubectl get crds`)
3. **Controller Ready**: Controller deployment has desired replicas running
4. **Webhook Ready**: Validating webhooks are responding (if configured)
5. **Basic Functionality**: Can create a test Workload CRD

### Expected Resources

- Namespace: `kalypso-system` (or custom)
- CRDs: All 12 Kalypso CRDs (Workload, WorkloadRegistration, Environment, etc.)
- Deployment: `kalypso-scheduler-controller-manager`
- ServiceAccount: `kalypso-scheduler-controller-manager`
- ClusterRole: `kalypso-scheduler-manager-role`
- ClusterRoleBinding: `kalypso-scheduler-manager-rolebinding`
- Service: `kalypso-scheduler-controller-manager-metrics-service` (optional)

---

## Entity: Prerequisites

**Description**: Required tools and authentication for bootstrap script execution.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tools.bash.installed` | boolean | Yes | Bash 4.0+ available |
| `tools.bash.version` | string | No | Detected Bash version |
| `tools.az.installed` | boolean | Yes | Azure CLI installed |
| `tools.az.version` | string | No | Detected az CLI version |
| `tools.az.min_version` | string | Yes | Minimum required: `2.50.0` |
| `tools.kubectl.installed` | boolean | Yes | kubectl installed |
| `tools.kubectl.version` | string | No | Detected kubectl version |
| `tools.kubectl.min_version` | string | Yes | Minimum required: `1.25.0` |
| `tools.git.installed` | boolean | Yes | git installed |
| `tools.gh.installed` | boolean | Yes | GitHub CLI installed |
| `tools.gh.version` | string | No | Detected gh version |
| `tools.gh.min_version` | string | Yes | Minimum required: `2.0.0` |
| `tools.helm.installed` | boolean | Yes | Helm installed |
| `tools.helm.version` | string | No | Detected Helm version |
| `tools.helm.min_version` | string | Yes | Minimum required: `3.0.0` |
| `tools.jq.installed` | boolean | Yes | jq installed |
| `auth.azure.authenticated` | boolean | Yes | Azure CLI logged in |
| `auth.azure.subscription_id` | string | No | Active subscription |
| `auth.github.authenticated` | boolean | Yes | GitHub CLI authenticated |
| `auth.github.user` | string | No | Authenticated GitHub user |

### Validation Rules

- All tools must be installed
- All tool versions must meet minimum requirements
- Both Azure and GitHub authentication must be active
- Active Azure subscription must have sufficient permissions

### State Transitions

1. **Unchecked** → **Checking**: Validation process started
2. **Checking** → **Valid**: All prerequisites met
3. **Checking** → **Invalid**: One or more prerequisites missing/outdated
4. **Invalid** → **Valid**: User installs/updates required tools

---

## Entity: Log Entry

**Description**: Individual log message written during bootstrap execution.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `timestamp` | ISO8601 string | Yes | When log entry created |
| `level` | enum | Yes | `ERROR`, `WARN`, `INFO`, `DEBUG` |
| `message` | string | Yes | Log message content |
| `context` | map | No | Additional structured data |

### Validation Rules

- Level must be one of the four defined values
- Timestamp must be valid ISO8601
- Message should not contain sensitive data (tokens, passwords)

### Output Destinations

- **Console**: INFO and above (ERROR, WARN, INFO) by default
- **Log File**: All levels (ERROR, WARN, INFO, DEBUG)
- **State File**: Only ERROR messages are stored in state for troubleshooting

---

## Relationships

```
Configuration
  ├─> determines creation of AKS Cluster
  ├─> determines creation of GitHub Repositories (2x)
  └─> configures Kalypso Installation

Bootstrap State
  ├─> tracks AKS Cluster (if created)
  ├─> tracks GitHub Repositories (if created)
  └─> tracks Kalypso Installation (if installed)

Prerequisites
  └─> must be valid before Configuration can be executed

AKS Cluster
  └─> hosts Kalypso Installation

GitHub Repository (Control-Plane)
  └─> referenced by Kalypso Installation (Flux configuration)

GitHub Repository (GitOps)
  └─> receives generated manifests from Kalypso Scheduler

Kalypso Installation
  ├─> requires AKS Cluster (or existing cluster)
  ├─> requires Control-Plane Repository URL
  └─> requires GitOps Repository URL

Log Entry
  └─> generated by all operations, stored in state on error
```

---

## Data Flow

1. **Configuration Phase**:
   - User provides input (CLI flags / config file / interactive prompts)
   - Configuration entity is populated and validated
   - Prerequisites entity is validated
   - Bootstrap State entity is created with status `in_progress`

2. **Cluster Phase**:
   - If `cluster.create=true`: AKS Cluster entity created
   - Kubeconfig updated with cluster credentials
   - Bootstrap State updated with cluster information

3. **Repository Phase**:
   - If `repositories.control_plane.create=true`: GitHub Repository (control-plane) created
   - If `repositories.gitops.create=true`: GitHub Repository (gitops) created
   - Templates populated and initial commit pushed
   - Bootstrap State updated with repository URLs

4. **Installation Phase**:
   - Kalypso Installation entity created via Helm
   - Health checks executed
   - Bootstrap State updated with installation details
   - State status changed to `completed`

5. **Validation Phase**:
   - All entities verified to be in expected state
   - Test workload created to verify functionality
   - Final status reported to user

6. **Rollback Phase** (if error occurs):
   - Bootstrap State read to identify created resources
   - Resources deleted in reverse order
   - State status changed to `rolled_back`

---

## Data Model Complete

All entities, their fields, relationships, state transitions, and validation rules have been defined. Implementation can proceed to contract definition (CLI interface specification).
