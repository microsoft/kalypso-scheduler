# Feature Specification: Bootstrapping Script

**Feature Branch**: `001-bootstrapping-script`  
**Created**: November 11, 2025  
**Status**: Draft  
**Input**: User description: "As a platform engineer I need a bootstrapping script with an md manual so that I can easily: Create or bring my own AKS cluster where Kalypso scheduler will be installed, Bring my own control-plane git repo or create a very minimal control plane repo with a single dev environment, single cluster type, single scheduling policy and a single config map, Bring my own gitops repo or create almost empty gitops repo containing only GitHub workflow and readme, Install Kalypso Scheduler"

## Clarifications

### Session 2025-11-11

- Q: How should the bootstrapping script obtain credentials for Azure and GitHub operations? → A: Environment variables with fallback to interactive prompts
- Q: Which operating systems should the bootstrapping script support? → A: macOS and Linux
- Q: When the script creates new GitHub repositories (control-plane and gitops), where should they be created? → A: User's GitHub account with optional --org flag for organization
- Q: What should be the default node size and count when creating a new AKS cluster? → A: Standard_DS2_v2, 3 nodes
- Q: When the script fails and triggers rollback/cleanup, what should be removed? → A: Delete only resources created in current run

## User Scenarios & Testing *(mandatory)*

### User Story - Bootstrapping Script (Priority: P1)

As a platform engineer I need a bootstrapping script with an md manual so that I can easily:
- Create or bring my own AKS cluster where Kalypso scheduler will be installed
- Bring my own control-plane git repo (e.g. https://github.com/microsoft/kalypso-control-plane) or create a very minimal control plane repo with a single dev environment, single cluster type, single scheduling policy and a single config map
- Bring my own gitops repo (e.g. https://github.com/microsoft/kalypso-gitops) or create almost empty gitops repo containing only GitHub workflow and readme
- Install Kalypso Scheduler

**Why this priority**: This is the foundational capability that enables platform engineers to get started with Kalypso Scheduler quickly, whether they're evaluating it for the first time or integrating it into existing infrastructure. It removes setup friction and provides flexibility for different deployment scenarios.

**Independent Test**: Can be fully tested by running the bootstrapping script in different modes (create new vs. bring your own) and verifying that all components are properly configured and Kalypso Scheduler is operational. Delivers immediate value by providing a working environment.

**Acceptance Scenarios**:

1. **Given** a platform engineer with no existing infrastructure, **When** they run the bootstrapping script in "create new" mode, **Then** the script creates a new AKS cluster, minimal control-plane repo with required components, minimal gitops repo with workflow and readme, and installs Kalypso Scheduler
2. **Given** a platform engineer with an existing AKS cluster, **When** they run the bootstrapping script and specify their cluster, **Then** the script uses the existing cluster and installs Kalypso Scheduler without creating a new cluster
3. **Given** a platform engineer with existing control-plane repository, **When** they provide the repository URL, **Then** the script validates and uses the existing repository instead of creating a new one
4. **Given** a platform engineer with existing gitops repository, **When** they provide the repository URL, **Then** the script validates and uses the existing repository instead of creating a new one
5. **Given** a platform engineer runs the script, **When** any component creation or validation fails, **Then** the script displays clear error messages and guidance for resolution
6. **Given** the script completes successfully, **When** the engineer reviews the generated markdown manual, **Then** they find comprehensive documentation covering prerequisites, configuration options, troubleshooting, and next steps
7. **Given** the minimal control-plane repo is created, **When** the engineer inspects it, **Then** main branch contains .environments, .github/workflows (CI/CD), templates, and sample workload registration; dev branch contains cluster-types, configs, scheduling-policies, base-repo.yaml, and gitops-repo.yaml
8. **Given** the minimal gitops repo is created, **When** the engineer inspects it, **Then** main branch has README; dev branch has .github/workflows/check-promote.yaml workflow

---

### Edge Cases

- What happens when the bootstrapping script is run multiple times on the same machine or with the same repository names?
- How does the system handle network failures during repository creation or cluster provisioning?
- What happens when the user provides invalid credentials or insufficient permissions for cluster creation or repository access?
- How does the script handle partially completed installations (e.g., cluster created but repository creation failed)?
- What happens when the user's machine doesn't have required tools installed (kubectl, git, az cli, etc.)?
- How does the system handle when provided "existing" repositories don't follow the expected structure for Kalypso?
- What happens when the AKS cluster creation fails due to quota limits or regional availability issues?
- How does the script behave when GitHub API rate limits are exceeded during repository operations?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The bootstrapping script MUST provide an option to create a new AKS cluster with Kalypso Scheduler installed
- **FR-002**: The bootstrapping script MUST provide an option to install Kalypso Scheduler on an existing AKS cluster specified by the user
- **FR-003**: The bootstrapping script MUST create a minimal control-plane repository with main and dev branches when no existing repository is provided. Main branch contains: .environments directory, .github/workflows directory (ci.yaml, cd.yaml, check-promote.yaml), templates directory, workloads directory with sample workload registration. Dev branch contains: cluster-types directory, configs directory, scheduling-policies directory, base-repo.yaml, and gitops-repo.yaml
- **FR-004**: The bootstrapping script MUST provide an option to use an existing control-plane git repository URL instead of creating a new one
- **FR-005**: The bootstrapping script MUST create a minimal gitops repository with main and dev branches when no existing repository is provided. Main branch contains minimal README. Dev branch contains .github/workflows/check-promote.yaml workflow
- **FR-006**: The bootstrapping script MUST provide an option to use an existing gitops repository URL instead of creating a new one
- **FR-007**: The bootstrapping script MUST validate that existing repositories (when provided) meet minimum structural requirements for Kalypso Scheduler integration
- **FR-008**: The bootstrapping script MUST install Kalypso Scheduler on the target cluster (new or existing) as the final step
- **FR-009**: The bootstrapping script MUST verify successful installation by checking that all Kalypso components are running
- **FR-010**: The script MUST provide clear progress indicators showing which step is currently executing
- **FR-011**: The script MUST handle errors gracefully by displaying specific error messages and stopping execution at the failed step
- **FR-012**: The script MUST support rollback or cleanup options when installation fails midway, deleting only resources created during the current run (not pre-existing resources)
- **FR-013**: Platform engineers MUST be able to run the script in interactive mode where they are prompted for each configuration choice
- **FR-014**: Platform engineers MUST be able to run the script in non-interactive mode by providing all configuration through command-line arguments or configuration file
- **FR-015**: The script MUST generate comprehensive markdown documentation that explains all setup options, prerequisites, and troubleshooting steps
- **FR-016**: The documentation MUST include examples for common setup scenarios (new everything, bring your own cluster, bring your own repos)
- **FR-017**: The script MUST verify all prerequisite tools are installed before beginning setup (kubectl, git, Azure CLI, etc.)
- **FR-018**: The script MUST obtain credentials via environment variables (AZURE_TOKEN, GITHUB_TOKEN) with fallback to interactive prompts when variables are not set
- **FR-019**: Created repositories MUST include readme files explaining their purpose and structure
- **FR-020**: The bootstrapping script MUST be idempotent where possible, detecting existing resources and skipping their creation
- **FR-021**: The script MUST run on macOS and Linux operating systems
- **FR-022**: When creating new repositories, the script MUST create them in the user's GitHub account by default, with support for an optional --org flag to specify a GitHub organization

### Key Entities

- **AKS Cluster**: The Kubernetes cluster where Kalypso Scheduler will be installed; can be newly created or existing; must have sufficient resources to run Kalypso components; default configuration for new clusters is Standard_DS2_v2 VM size with 3 nodes
- **Control-Plane Repository**: Git repository containing Kalypso configuration resources (environments, cluster types, scheduling policies, config maps); can be newly created with minimal structure or existing with validated structure
- **GitOps Repository**: Git repository containing deployment manifests and GitHub workflows for continuous delivery; can be newly created with basic structure or existing with compatible workflows
- **Kalypso Scheduler Installation**: The deployed Kalypso Scheduler operator and associated components running on the cluster
- **Bootstrap Configuration**: User-provided or interactively collected settings that determine which resources to create vs. use existing (cluster settings, repository URLs, installation parameters)
- **Documentation**: Generated markdown manual explaining setup process, configuration options, prerequisites, and troubleshooting guidance

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Platform engineers can complete a full new installation (cluster + repos + scheduler) in under 15 minutes using the bootstrapping script
- **SC-002**: Platform engineers can install Kalypso Scheduler on an existing cluster in under 5 minutes
- **SC-003**: 95% of script executions either complete successfully or fail with clear, actionable error messages
- **SC-004**: The generated documentation covers all configuration scenarios and enables engineers to troubleshoot 80% of common issues without external support
- **SC-005**: Engineers using the script can deploy their first workload to the bootstrapped environment within 10 minutes of script completion by following the provided documentation
- **SC-006**: The bootstrapping script successfully detects and reports missing prerequisites in 100% of cases before attempting installation
- **SC-007**: Created minimal repositories contain all required components and pass validation checks for Kalypso Scheduler compatibility
