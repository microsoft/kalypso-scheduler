# Quick Start: Kalypso Scheduler Bootstrap

**Feature**: 001-bootstrapping-script  
**Date**: 2025-11-11  
**Audience**: Platform Engineers

## Overview

This quick start guide will help you bootstrap a complete Kalypso Scheduler environment in under 15 minutes. Choose the scenario that best fits your needs.

---

## Prerequisites

Before you begin, ensure you have the following installed and configured:

### Required Tools

| Tool | Minimum Version | Installation |
|------|-----------------|--------------|
| Bash | 4.0+ | Pre-installed on Linux; macOS users run `brew install bash` |
| Azure CLI | 2.50.0+ | [Install Guide](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) |
| kubectl | 1.25.0+ | [Install Guide](https://kubernetes.io/docs/tasks/tools/) |
| Helm | 3.0.0+ | [Install Guide](https://helm.sh/docs/intro/install/) |
| GitHub CLI | 2.0.0+ | [Install Guide](https://cli.github.com/) |
| git | 2.0+ | Pre-installed on most systems |
| jq | 1.6+ | `brew install jq` (macOS) or `apt install jq` (Ubuntu) |

### Authentication

**Azure**:
```bash
# Login to Azure
az login

# Set subscription (if you have multiple)
az account set --subscription "Your Subscription Name"

# Verify authentication
az account show
```

**GitHub**:
```bash
# Login to GitHub
gh auth login

# Verify authentication
gh auth status
```

**Alternative**: Set environment variables:
```bash
export AZURE_SUBSCRIPTION_ID="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxx"
```

---

## Quick Validation

Before running the bootstrap script, validate your prerequisites:

```bash
./scripts/bootstrap/bootstrap.sh --validate-only
```

Expected output:
```
[INFO] Checking prerequisites...
[INFO] ✓ Bash version 5.2.15
[INFO] ✓ Azure CLI version 2.54.0
[INFO] ✓ kubectl version 1.28.0
[INFO] ✓ Helm version 3.13.0
[INFO] ✓ GitHub CLI version 2.38.0
[INFO] ✓ git version 2.42.0
[INFO] ✓ jq version 1.7
[INFO] ✓ Azure authenticated (Subscription: My Subscription)
[INFO] ✓ GitHub authenticated (User: myusername)
[INFO] ✓ All prerequisites met
```

---

## Scenario 1: Create Everything New (Recommended for First-Time Users)

**What it does**: Creates a new AKS cluster, control-plane repository, gitops repository, and installs Kalypso Scheduler.

**Time**: ~15 minutes

### Step 1: Run Bootstrap Script

```bash
./scripts/bootstrap/bootstrap.sh \
  --cluster-name my-kalypso \
  --cluster-region eastus \
  --control-plane-create \
  --gitops-create
```

### Step 2: Wait for Completion

The script will:
1. ✓ Validate prerequisites (30 seconds)
2. ✓ Create AKS cluster (5-7 minutes)
3. ✓ Create control-plane repository (30 seconds)
4. ✓ Create gitops repository (30 seconds)
5. ✓ Install Kalypso Scheduler (2-3 minutes)
6. ✓ Run health checks (30 seconds)

### Step 3: Verify Installation

```bash
# Configure kubectl
kubectl config use-context my-kalypso

# Check Kalypso pods
kubectl get pods -n kalypso-system

# Expected output:
# NAME                                                     READY   STATUS    RESTARTS   AGE
# kalypso-scheduler-controller-manager-xxxxxxxxxx-xxxxx    2/2     Running   0          2m
```

### Step 4: View Created Resources

```bash
# Check state
./scripts/bootstrap/bootstrap.sh --status
```

Output shows:
- Cluster name and region
- Repository URLs
- Kubeconfig context
- Installation namespace

### Step 5: Next Steps

Your Kalypso Scheduler is now ready! See **"What's Next?"** section below.

---

## Scenario 2: Use Existing AKS Cluster

**What it does**: Uses your existing cluster and creates new repositories.

**Time**: ~5 minutes

**Prerequisites**: You must have an existing AKS cluster with `kubectl` access.

### Step 1: Get Cluster Context

```bash
# List available contexts
kubectl config get-contexts

# Note your cluster context name (e.g., "my-existing-cluster")
```

### Step 2: Run Bootstrap Script

```bash
./scripts/bootstrap/bootstrap.sh \
  --cluster-existing my-existing-cluster \
  --control-plane-create \
  --gitops-create
```

### Step 3: Verify Installation

```bash
kubectl get pods -n kalypso-system
```

---

## Scenario 3: Bring Your Own Repositories

**What it does**: Creates new cluster and uses your existing control-plane and gitops repositories.

**Time**: ~10 minutes

**Prerequisites**: You must have existing repositories that follow Kalypso structure.

### Step 1: Verify Repository Structure

Your control-plane repository should have:
```
environments/
cluster-types/
scheduling-policies/
config-maps/
```

Your gitops repository should have:
```
.github/workflows/
clusters/
```

### Step 2: Run Bootstrap Script

```bash
./scripts/bootstrap/bootstrap.sh \
  --cluster-name my-kalypso \
  --cluster-region eastus \
  --control-plane-repo https://github.com/myorg/my-control-plane \
  --gitops-repo https://github.com/myorg/my-gitops
```

---

## Scenario 4: Fully Customized Setup

**What it does**: Maximum control over all configuration options.

### Step 1: Create Configuration File

```bash
cat > bootstrap-config.yaml <<EOF
cluster:
  name: prod-kalypso
  create: true
  region: westus2
  nodeCount: 5
  vmSize: Standard_D4s_v3

repositories:
  controlPlane:
    create: true
    name: kalypso-control-plane-prod
  gitops:
    create: true
    name: kalypso-gitops-prod
  githubOrg: mycompany
  visibility: private

installation:
  namespace: kalypso-production
  helmValues: ./my-helm-values.yaml

options:
  skipConfirmation: false
  verbose: true
EOF
```

### Step 2: Run Bootstrap Script

```bash
./scripts/bootstrap/bootstrap.sh --config bootstrap-config.yaml
```

---

## Interactive Mode (Default)

If you run the script without all required parameters, it will prompt you interactively:

```bash
./scripts/bootstrap/bootstrap.sh
```

Example interaction:
```
Create new AKS cluster or use existing? [create/existing]: create
Cluster name (DNS-compatible, lowercase): my-kalypso
Azure region [eastus]: westus2
Number of nodes [3]: 3
VM size [Standard_DS2_v2]: 

Create new control-plane repository or use existing? [create/existing]: create
Control-plane repository name [my-kalypso-control-plane]: 
Create in organization or user account? [org/user]: user

Create new gitops repository or use existing? [create/existing]: create
GitOps repository name [my-kalypso-gitops]: 

Kubernetes namespace [kalypso-system]: 

=== Bootstrap Configuration ===
Cluster:
  Name: my-kalypso
  Create: true
  Region: westus2
  Nodes: 3
  VM Size: Standard_DS2_v2

Repositories:
  Control-Plane: Create new (my-kalypso-control-plane)
  GitOps: Create new (my-kalypso-gitops)

Installation:
  Namespace: kalypso-system

Proceed with bootstrap? [y/N]: y
```

---

## Troubleshooting

### Problem: Prerequisites Validation Fails

**Solution**: Install or update the missing tool:

```bash
# Example: Update Azure CLI
brew upgrade azure-cli  # macOS
# or
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash  # Debian/Ubuntu

# Example: Install GitHub CLI
brew install gh  # macOS
# or
type -p curl >/dev/null || sudo apt install curl -y
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
sudo chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
sudo apt update
sudo apt install gh -y
```

### Problem: Cluster Creation Takes Too Long

**Symptoms**: Cluster creation exceeds 10 minutes

**Solution**: This is normal for AKS cluster provisioning. Wait patiently or check Azure Portal for status.

### Problem: Repository Already Exists

**Symptoms**: Error message "Repository 'name' already exists"

**Solutions**:
1. Use existing repository: `--control-plane-repo https://github.com/user/name`
2. Use different name: `--control-plane-name my-kalypso-cp-2`
3. Delete existing repo: `gh repo delete user/name`

### Problem: GitHub API Rate Limit

**Symptoms**: Error message "API rate limit exceeded"

**Solution**: Wait 1 hour or authenticate with a token that has higher limits:
```bash
gh auth login
```

### Problem: Insufficient Azure Quota

**Symptoms**: Error message "Quota exceeded" or "Not enough cores"

**Solutions**:
1. Use smaller VM size: `--cluster-vm-size Standard_B2s`
2. Reduce node count: `--cluster-nodes 1`
3. Request quota increase in Azure Portal

### Problem: Installation Fails

**Symptoms**: Helm installation reports errors

**Solution**: Check logs and retry:
```bash
# View detailed logs
cat $HOME/.kalypso/bootstrap-*.log | tail -100

# Clean up and retry
./scripts/bootstrap/bootstrap.sh --clean
./scripts/bootstrap/bootstrap.sh --cluster-name my-kalypso
```

---

## What's Next?

After successful bootstrap, you can:

### 1. Explore Your Control-Plane Repository

```bash
# Clone the repository
gh repo clone $(./scripts/bootstrap/bootstrap.sh --status | grep "Control-Plane Repo" | awk '{print $3}')

# View the structure
cd *-control-plane
tree .
```

You'll find minimal configuration with two branches:

**Main branch** (promotion source):
- **.environments/dev.yaml**: Development environment definition
- **.github/workflows/**: CI/CD workflows (ci.yaml, cd.yaml, check-promote.yaml)
- **templates/default-template.yaml**: Default Kalypso template for workload transformation
- **workloads/sample-workload-registration.yaml**: Example workload registration

**Dev branch** (environment-specific):
- **cluster-types/default.yaml**: Default cluster type
- **configs/default-config.yaml**: Environment configuration
- **scheduling-policies/default-policy.yaml**: Default scheduling policy
- **base-repo.yaml**: Reference to main branch commit
- **gitops-repo.yaml**: Reference to GitOps repository

### 2. Deploy Your First Workload

Create a `WorkloadRegistration`:

```yaml
# workload-registration.yaml
apiVersion: scheduler.kalypso.io/v1alpha1
kind: WorkloadRegistration
metadata:
  name: hello-world
  namespace: kalypso-system
spec:
  url: https://github.com/myorg/hello-world-manifests
  branch: main
  path: manifests/
```

Apply it:
```bash
kubectl apply -f workload-registration.yaml
```

Watch Kalypso create `DeploymentTargets`:
```bash
kubectl get deploymenttargets -n kalypso-system --watch
```

### 3. Customize Scheduling Policies

Edit your control-plane repository to add more sophisticated policies:

```bash
cd *-control-plane
# Edit scheduling-policies/default.yaml
# Commit and push changes
git add scheduling-policies/
git commit -m "Update scheduling policy"
git push
```

Flux will sync the changes to your cluster automatically.

### 4. Add More Environments

Create additional environments (staging, production):

```yaml
# environments/staging.yaml
apiVersion: scheduler.kalypso.io/v1alpha1
kind: Environment
metadata:
  name: staging
  namespace: kalypso-system
spec:
  displayName: Staging
  description: Pre-production environment
```

### 5. Monitor Kalypso Operations

```bash
# Watch controller logs
kubectl logs -n kalypso-system deployment/kalypso-scheduler-controller-manager -f

# View all Kalypso resources
kubectl get workloads,deploymenttargets,assignments -n kalypso-system

# Check status of specific workload
kubectl get workload hello-world -n kalypso-system -o yaml
```

### 6. Explore GitOps Repository

```bash
# Clone the gitops repository
gh repo clone $(./scripts/bootstrap/bootstrap.sh --status | grep "GitOps Repo" | awk '{print $3}')

cd *-gitops
```

You'll see two branches:

**Main branch** (minimal):
- **README.md**: Documentation

**Dev branch**:
- **.github/workflows/check-promote.yaml**: Workflow triggered by PRs from control-plane
- **README.md**: Documentation
- Cluster directories (e.g., `small/`, `large/`) will be created by Kalypso during operation

### 7. Set Up CI/CD

The gitops repository includes a GitHub Actions workflow. Configure it:

1. Add secrets to your GitHub repository:
   - `KUBE_CONFIG`: Base64-encoded kubeconfig
   - `AZURE_CREDENTIALS`: Azure service principal credentials

2. Trigger workflow on push:
   ```bash
   # Make a change
   echo "# Update" >> README.md
   git add README.md
   git commit -m "Test workflow"
   git push
   ```

3. Watch workflow run in GitHub Actions tab

---

## Advanced Usage

### Non-Interactive Mode for CI/CD

```bash
# Set credentials
export AZURE_SUBSCRIPTION_ID="..."
export GITHUB_TOKEN="..."

# Run with config file
./scripts/bootstrap/bootstrap.sh \
  --config bootstrap-config.yaml \
  --non-interactive \
  --auto-rollback \
  --no-color
```

### Custom Helm Values

```yaml
# my-values.yaml
controllerManager:
  replicas: 2
  resources:
    limits:
      cpu: 500m
      memory: 256Mi
```

```bash
./scripts/bootstrap/bootstrap.sh \
  --cluster-name my-kalypso \
  --helm-values ./my-values.yaml
```

### Multiple Clusters

Bootstrap multiple clusters (run separately for each):

```bash
# Development cluster
./scripts/bootstrap/bootstrap.sh \
  --cluster-name dev-kalypso \
  --cluster-region eastus \
  --cluster-nodes 3 \
  --namespace kalypso-dev

# Production cluster
./scripts/bootstrap/bootstrap.sh \
  --cluster-name prod-kalypso \
  --cluster-region westus2 \
  --cluster-nodes 10 \
  --cluster-vm-size Standard_D4s_v3 \
  --namespace kalypso-prod
```

---

## Cleanup

To remove all resources created by the bootstrap script:

```bash
# Check what will be deleted
./scripts/bootstrap/bootstrap.sh --status

# Delete all resources
./scripts/bootstrap/bootstrap.sh --clean
```

**Warning**: This will delete:
- AKS cluster (if created by script)
- Kubeconfig context
- Helm release
- **NOT repositories** (manual deletion required for safety)

To delete repositories:
```bash
# Get repository names from status
./scripts/bootstrap/bootstrap.sh --status

# Delete manually
gh repo delete myuser/my-kalypso-control-plane
gh repo delete myuser/my-kalypso-gitops
```

---

## Getting Help

### View Script Help

```bash
./scripts/bootstrap/bootstrap.sh --help
```

### Check Script Version

```bash
./scripts/bootstrap/bootstrap.sh --version
```

### View Logs

```bash
# Latest log
ls -lt $HOME/.kalypso/bootstrap-*.log | head -1 | awk '{print $9}' | xargs cat

# Or specify log file
cat $HOME/.kalypso/bootstrap-1699700345.log
```

### Report Issues

If you encounter issues:

1. Run with verbose logging:
   ```bash
   ./scripts/bootstrap/bootstrap.sh --cluster-name my-kalypso --verbose
   ```

2. Collect logs:
   ```bash
   cp $HOME/.kalypso/bootstrap-*.log ./
   ```

3. Submit issue with logs and error messages

---

## Summary

You now have a fully functional Kalypso Scheduler environment! Here's what you've accomplished:

✅ Installed and configured required tools  
✅ Created or connected AKS cluster  
✅ Set up control-plane Git repository  
✅ Set up gitops Git repository  
✅ Installed Kalypso Scheduler operator  
✅ Verified installation health  

**Next**: Deploy your first workload and explore Kalypso's scheduling capabilities!

For comprehensive documentation, see `docs/bootstrap/BOOTSTRAP.md`.
