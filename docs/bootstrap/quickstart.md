# Kalypso Bootstrap Quickstart Guide

Get up and running with Kalypso Scheduler in under 15 minutes.

## Prerequisites Check

Before starting, verify you have:

- [x] kubectl (>= 1.20.0)
- [x] Azure CLI (>= 2.30.0)
- [x] git (>= 2.0.0)
- [x] Helm (>= 3.0.0)
- [x] Azure account with subscription access
- [x] GitHub personal access token

See [prerequisites.md](prerequisites.md) for detailed installation instructions.

## Quick Setup (5 Minutes)

### 1. Authenticate

Login to Azure:
```bash
az login
az account set --subscription "your-subscription-id"
```

Set GitHub token:
```bash
export GITHUB_TOKEN="your-github-personal-access-token"
```

### 2. Run Bootstrap Script

Create everything new (interactive mode):
```bash
cd scripts/bootstrap
./bootstrap.sh
```

The script will:
1. ✓ Check prerequisites
2. ✓ Prompt for configuration
3. ✓ Create AKS cluster (~10 minutes)
4. ✓ Create GitHub repositories
5. ✓ Install Kalypso Scheduler
6. ✓ Verify installation

### 3. Verify Installation

Check that everything is running:
```bash
# Get cluster credentials (if not already done)
az aks get-credentials \
  --resource-group kalypso-rg \
  --name kalypso-cluster

# Check Kalypso pods
kubectl get pods -n kalypso-system

# Expected output:
# NAME                                                READY   STATUS    RESTARTS   AGE
# kalypso-scheduler-controller-manager-xxxxx-xxxxx    2/2     Running   0          2m

# Check CRDs
kubectl get crd | grep kalypso

# Expected output:
# deploymenttargets.scheduler.kalypso.io
# workloads.scheduler.kalypso.io
# schedulingpolicies.scheduler.kalypso.io
# ... (and more)
```

## Deploy Your First Workload

### 1. Clone Control Plane Repository

```bash
cd ~
git clone https://github.com/YOUR_USER/kalypso-control-plane
cd kalypso-control-plane
```

### 2. Create a Simple Workload

Create a file `workloads/hello-world.yaml`:

```yaml
apiVersion: scheduler.kalypso.io/v1alpha1
kind: WorkloadRegistration
metadata:
  name: hello-world
  namespace: kalypso-system
spec:
  workloadName: hello-world-app
  sourceRepoURL: https://github.com/YOUR_USER/hello-world-app
  sourceBranch: main
  sourcePath: manifests
  targets:
    - name: dev
      cluster: ""  # Will be scheduled by policy
```

### 3. Commit and Push

```bash
git add workloads/hello-world.yaml
git commit -m "Add hello-world workload"
git push origin main
```

### 4. Watch Kalypso Schedule the Workload

```bash
# Watch for Workload creation
kubectl get workloads -n kalypso-system --watch

# Watch for DeploymentTarget creation
kubectl get deploymenttargets -n kalypso-system --watch

# Watch for Assignment creation
kubectl get assignments -n kalypso-system --watch
```

Kalypso will:
1. Detect the WorkloadRegistration
2. Create a Workload resource
3. Create DeploymentTargets based on targets
4. Use SchedulingPolicies to create Assignments
5. Generate manifests in the GitOps repository

### 5. Check GitOps Repository

```bash
cd ~/kalypso-gitops
git pull origin main

# You should see new directories for cluster types and deployment targets
ls -R clusters/
```

## Common Workflows

### Using Existing AKS Cluster

```bash
./bootstrap.sh \
  --use-cluster my-existing-cluster \
  --resource-group my-existing-rg \
  --create-repos
```

### Using Existing Repositories

```bash
./bootstrap.sh \
  --create-cluster \
  --control-plane-repo https://github.com/myorg/control-plane \
  --gitops-repo https://github.com/myorg/gitops
```

### Automated Setup (CI/CD)

Create a config file `kalypso-config.yaml`:

```yaml
cluster:
  name: kalypso-prod
  resourceGroup: kalypso-prod-rg
  location: westus2
  nodeCount: 5
  nodeSize: Standard_DS3_v2

repositories:
  controlPlane: ""
  gitops: ""

github:
  org: my-organization
```

Run non-interactively:

```bash
export AZURE_SUBSCRIPTION_ID="xxx"
export GITHUB_TOKEN="xxx"

./bootstrap.sh --config kalypso-config.yaml --non-interactive
```

## Next Steps

### Learn More

- **Scheduling Policies**: Configure how workloads are assigned to clusters
- **Cluster Types**: Define different types of target clusters
- **Templates**: Customize manifest generation
- **Environments**: Set up multiple environments (dev, staging, prod)

### Explore Examples

Check the `example/` directory in the Kalypso repository for:
- Sample workload registrations
- Scheduling policy configurations
- Cluster type definitions
- Template examples

### Customize Your Setup

1. **Modify Scheduling Policies**: Edit `scheduling-policies/` in control-plane repo
2. **Add Cluster Types**: Create new cluster type definitions
3. **Configure Templates**: Customize manifest templates
4. **Set Up Environments**: Add staging, production environments

### Advanced Topics

- **Multi-Environment Setup**: Configure dev, staging, and production
- **Custom Templates**: Create specialized manifest templates
- **Policy Composition**: Combine multiple scheduling policies
- **GitOps Integration**: Configure CI/CD pipelines

## Troubleshooting

### Bootstrap fails during cluster creation

Check Azure quotas:
```bash
az vm list-usage --location eastus --output table
```

### Kalypso pods not starting

Check pod logs:
```bash
kubectl logs -n kalypso-system -l app=kalypso-scheduler
```

### Workload not being scheduled

Check Kalypso logs and scheduling policies:
```bash
kubectl logs -n kalypso-system deployment/kalypso-scheduler-controller-manager
kubectl get schedulingpolicies -n kalypso-system
```

For more troubleshooting, see [troubleshooting.md](troubleshooting.md).

## Cleanup

To remove everything created by the bootstrap script:

```bash
# Delete Kalypso
helm uninstall kalypso-scheduler -n kalypso-system

# Delete AKS cluster
az aks delete \
  --resource-group kalypso-rg \
  --name kalypso-cluster \
  --yes

# Delete resource group (if created by script)
az group delete --name kalypso-rg --yes

# Delete GitHub repositories (via UI or gh CLI)
gh repo delete YOUR_USER/kalypso-control-plane --yes
gh repo delete YOUR_USER/kalypso-gitops --yes
```

## Getting Help

- **Documentation**: [README.md](README.md)
- **Issues**: https://github.com/microsoft/kalypso-scheduler/issues
- **Discussions**: https://github.com/microsoft/kalypso-scheduler/discussions

## Success Metrics

After completing this quickstart, you should be able to:

- [x] Bootstrap Kalypso infrastructure in under 15 minutes
- [x] Deploy a workload using WorkloadRegistration
- [x] See Kalypso schedule workloads to clusters
- [x] View generated manifests in GitOps repository
- [x] Understand the basic Kalypso workflow

Congratulations! You now have a working Kalypso Scheduler environment. 🎉
