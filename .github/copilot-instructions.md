# Kalypso Scheduler - AI Coding Agent Instructions

## Project Overview

Kalypso Scheduler is a **Kubebuilder-based Kubernetes operator** that orchestrates workload assignments across cluster types using declarative GitOps workflows. It transforms high-level control plane abstractions (Workloads, ClusterTypes, SchedulingPolicies) into low-level manifests in GitOps repositories.

**Architecture**: Multi-controller reconciliation pipeline where resources flow: `WorkloadRegistration → Workload → DeploymentTarget → SchedulingPolicy → Assignment → AssignmentPackage → GitOpsRepo (PR creation)`

## Critical Development Patterns

### 1. Kubebuilder Framework - The Foundation

All code generation and scaffolding MUST use Kubebuilder:

```bash
# ALWAYS regenerate after changing API types or adding RBAC markers
make manifests generate

# Standard development workflow
make fmt vet test
```

**Never manually edit**:
- `config/crd/bases/*.yaml` - Generated from API types
- `**/zz_generated.deepcopy.go` - Generated DeepCopy methods
- RBAC files - Generated from `+kubebuilder:rbac` markers

### 2. Controller Architecture - Reconciliation Chain

Each controller reconciles ONE CRD type but watches related resources:

**Example: SchedulingPolicyReconciler**
```go
// Watches its own type + triggers reconciliation when dependencies change
ctrl.NewControllerManagedBy(mgr).
    For(&schedulerv1alpha1.SchedulingPolicy{}).
    Owns(&schedulerv1alpha1.Assignment{}).
    Watches(&schedulerv1alpha1.ClusterType{}, 
        handler.EnqueueRequestsFromMapFunc(r.findPolicies)).
    Watches(&schedulerv1alpha1.DeploymentTarget{},
        handler.EnqueueRequestsFromMapFunc(r.findPolicies))
```

**Pattern**: When ClusterType or DeploymentTarget changes, ALL SchedulingPolicies in that namespace re-reconcile to recompute assignments.

### 3. Field Indexing - Enable Efficient Queries

Controllers create field indexes in `SetupWithManager()`:

```go
// Index assignments by clusterType field
mgr.GetFieldIndexer().IndexField(ctx, &schedulerv1alpha1.Assignment{}, 
    ClusterTypeField, func(rawObj client.Object) []string {
        return []string{rawObj.(*schedulerv1alpha1.Assignment).Spec.ClusterType}
    })

// Query using the index
r.List(ctx, assignments, client.MatchingFields{ClusterTypeField: "production"})
```

**When to add**: Any field used frequently in `List()` queries across controllers.

### 4. Label-Based Cross-Resource Relationships

Labels connect resources across the reconciliation chain:

```go
// Constants in api/v1alpha1/deploymenttarget_types.go
const (
    WorkspaceLabel = "workspace"  // Links DeploymentTarget to WorkloadRegistration
    WorkloadLabel  = "workload"   // Links DeploymentTarget to Workload
)

// Usage in WorkloadReconciler
deploymentTarget.Labels = map[string]string{
    WorkloadLabel:   workload.Name,
    WorkspaceLabel:  workspaceLabel,
}
```

**Pattern**: `FluxOwnerLabel` ("kustomize.toolkit.fluxcd.io/name") tracks which Flux Kustomization owns a resource - used to derive workspace from WorkloadRegistration name.

### 5. Status Management - Always Update

Every reconciliation MUST update status with conditions:

```go
// Set condition based on outcome
condition := metav1.Condition{
    Type:   "Ready",
    Status: metav1.ConditionTrue,  // or False
    Reason: "DeploymentTargetsCreated",
}
meta.SetStatusCondition(&workload.Status.Conditions, condition)

// ALWAYS update status
updateErr := r.Status().Update(ctx, workload)
if updateErr != nil {
    return ctrl.Result{RequeueAfter: time.Second * 3}, updateErr
}
```

**Error Handling Pattern**: Controllers have `manageFailure()` helper that sets Ready=False condition with error message before returning.

### 6. Template Processing - Go Templates with Sprig

The `scheduler/templater.go` processes CRD templates using Go's `text/template` + Sprig functions:

```go
// Template variables available in Template CRDs
type dataType struct {
    DeploymentTargetName string
    Namespace            string
    Environment          string
    Workspace            string
    Workload             string
    Labels               map[string]string
    ClusterType          string
    ConfigData           map[string]interface{}
}

// Custom template functions
funcMap := template.FuncMap{
    "toYaml":    toYAML,      // Convert to YAML
    "stringify": stringify,    // Convert to string
    "hash":      hash,         // Hash for content addressing
    "unquote":   unquote,      // Remove quotes
}
```

**Example Template CR** (from README):
```yaml
manifests:
  - apiVersion: argoproj.io/v1alpha1
    kind: Application
    metadata:
      name: "{{ .DeploymentTargetName}}"
      namespace: argocd
    spec:
      destination:
        namespace: "{{ .Namespace}}"
      source:
        repoURL: "{{ .Repo}}"
        path: "{{ .Path}}"
```

### 7. Flux Integration - GitOps Delivery

Controllers create Flux resources to fetch workloads from Git:

```go
// controllers/flux.go - Create GitRepository + Kustomization
flux.CreateFluxReferenceResources(
    name, namespace, targetNamespace, 
    url, branch, path, commit)
```

**Constants**: `DefaulFluxNamespace = "flux-system"`, `RepoSecretName = "gh-repo-secret"` (hardcoded GitHub token reference)

### 8. GitHub PR Creation - Final Output

`GitOpsRepoReconciler` aggregates all AssignmentPackages and creates PRs via `scheduler/githubrepo.go`:

```go
type GitHubRepo interface {
    CreatePR(message string, content RepoContentType) (*github.PullRequest, error)
}

// Organizes manifests by ClusterType/DeploymentTarget
type RepoContentType struct {
    ClusterTypes map[string]*ClusterTypeContentType
}
```

**Pattern**: Waits for all SchedulingPolicies and Assignments to have `Ready=True` before creating PR.

## Testing Approach

**Unit Tests**: `scheduler/*_test.go` - Test business logic (scheduling, templating, validation) in isolation

```bash
# Run tests with envtest (simulated Kubernetes API)
make test
```

**Integration Tests**: `controllers/suite_test.go` - Use Ginkgo/Gomega with envtest for controller reconciliation

**Test Data**: `scheduler/testData/` contains sample manifests for template processing tests

## Common Development Tasks

### Adding a New CRD

```bash
# Scaffold new API
kubebuilder create api --group scheduler --version v1alpha1 --kind NewResource

# Edit api/v1alpha1/newresource_types.go - add fields with markers
# Edit controllers/newresource_controller.go - implement Reconcile()

# Regenerate manifests and code
make manifests generate

# Update RBAC if controller accesses other resources
# Add +kubebuilder:rbac markers in controller file
```

### Modifying Existing CRD

1. Edit `api/v1alpha1/*_types.go` - add fields, validation markers
2. Run `make manifests generate` - regenerates CRDs and DeepCopy
3. Update `config/crd/bases/*.yaml` will be auto-updated
4. Update controller logic if needed
5. Update Helm chart if CRD schema changed: `make helm-build`

### Debugging Controllers Locally

```bash
# Run operator locally (uses ~/.kube/config)
make run

# Or build and run specific controller with debug logging
go run ./main.go --zap-log-level=debug
```

## Project-Specific Conventions

### Naming Patterns

- **DeploymentTarget names**: `{workload-name}-{target-name}` (e.g., `hello-world-app-functional-test`)
- **Assignment names**: `{workload}-{deploymentTarget}-{clusterType}` (e.g., `app-dev-edge-small`)
- **Flux resource names**: Match the resource they're fetching (WorkloadRegistration name, Environment name)

### File Organization

```
api/v1alpha1/          # CRD Go types (12 CRDs defined in PROJECT file)
controllers/           # Reconcilers (one per CRD with controller=true)
scheduler/             # Business logic (scheduling, templating, GitHub ops)
config/crd/bases/      # Generated CRD YAML
config/rbac/           # Generated RBAC + custom editor/viewer roles
helm/kalypso-scheduler/# Helm chart for deployment
```

### Environment Namespaces

Each Environment CR creates a namespace on the control plane cluster where Flux delivers environment-specific resources (ClusterTypes, SchedulingPolicies, ConfigMaps). **DeploymentTargets have `spec.environment` field** that MUST match the namespace they're in.

## API Version Status

**Current**: v1alpha1 (unstable, breaking changes allowed)
**Stability Goal**: See `.specify/memory/constitution.md` for v1beta1/v1 transition requirements

---

**Constitution Reference**: See `.specify/memory/constitution.md` for comprehensive governance, testing requirements, and development standards.

**README**: See `README.md` for architecture diagrams and example CRD manifests.

## Active Technologies
- Bash 4.0+ (for maximum portability across macOS/Linux) + Azure CLI, kubectl, git, GitHub CLI (gh), jq, curl (001-bootstrapping-script)
- Local filesystem for configuration state, GitHub for repositories, Azure for cluster state (001-bootstrapping-script)

## Recent Changes
- 001-bootstrapping-script: Added Bash 4.0+ (for maximum portability across macOS/Linux) + Azure CLI, kubectl, git, GitHub CLI (gh), jq, curl
