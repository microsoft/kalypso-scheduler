#!/usr/bin/env bash
# Kalypso Scheduler Bootstrapping Script
# Main entry point for bootstrapping Kalypso Scheduler infrastructure
#
# Usage: ./bootstrap.sh [OPTIONS]
#
# This script helps platform engineers set up Kalypso Scheduler by:
# - Creating or using existing AKS clusters
# - Creating or using existing control-plane repositories
# - Creating or using existing gitops repositories
# - Installing Kalypso Scheduler on the target cluster
#
# For detailed usage information, run: ./bootstrap.sh --help

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source library files
# shellcheck source=lib/utils.sh
source "${SCRIPT_DIR}/lib/utils.sh"

# shellcheck source=lib/prerequisites.sh
source "${SCRIPT_DIR}/lib/prerequisites.sh"

# shellcheck source=lib/config.sh
source "${SCRIPT_DIR}/lib/config.sh"

# shellcheck source=lib/cluster.sh
source "${SCRIPT_DIR}/lib/cluster.sh"

# shellcheck source=lib/repositories.sh
source "${SCRIPT_DIR}/lib/repositories.sh"

# shellcheck source=lib/install.sh
source "${SCRIPT_DIR}/lib/install.sh"

#######################################
# Display usage information
# Globals:
#   None
# Arguments:
#   None
# Returns:
#   None
#######################################
show_usage() {
    cat <<EOF
Kalypso Scheduler Bootstrapping Script

USAGE:
    $(basename "$0") [OPTIONS]

DESCRIPTION:
    Bootstrap Kalypso Scheduler infrastructure by creating or using existing:
    - AKS cluster where Kalypso Scheduler will be installed
    - Control-plane repository with configuration resources
    - GitOps repository for continuous delivery

OPTIONS:
    -h, --help              Show this help message
    -v, --verbose           Enable verbose logging
    -q, --quiet             Suppress non-error output
    --config FILE           Load configuration from FILE
    --non-interactive       Run in non-interactive mode (requires config file or env vars)
    
CLUSTER OPTIONS:
    --create-cluster        Create a new AKS cluster
    --use-cluster NAME      Use existing AKS cluster NAME
    --cluster-name NAME     Name for new cluster (default: kalypso-cluster)
    --resource-group NAME   Azure resource group (default: kalypso-rg)
    --location LOCATION     Azure location (default: eastus)
    --node-count COUNT      Number of nodes (default: 3)
    --node-size SIZE        VM size (default: Standard_DS2_v2)
    
REPOSITORY OPTIONS:
    --create-repos          Create new control-plane and gitops repositories
    --control-plane-repo URL    Use existing control-plane repository
    --gitops-repo URL       Use existing gitops repository
    --github-org ORG        GitHub organization (default: user's account)
    
AUTHENTICATION:
    Environment variables:
        AZURE_SUBSCRIPTION_ID   Azure subscription ID
        GITHUB_TOKEN           GitHub personal access token
        
EXAMPLES:
    # Interactive mode (recommended for first-time users)
    ./$(basename "$0")
    
    # Create everything new
    ./$(basename "$0") --create-cluster --create-repos --non-interactive
    
    # Use existing cluster, create repositories
    ./$(basename "$0") --use-cluster my-cluster --create-repos --non-interactive
    
    # Use configuration file
    ./$(basename "$0") --config bootstrap-config.yaml

For detailed documentation, see: docs/bootstrap/README.md
EOF
}

#######################################
# Main execution flow
# Globals:
#   LOG_LEVEL
# Arguments:
#   Command line arguments
# Returns:
#   0 on success, 1 on failure
#######################################
main() {
    # Initialize logging
    init_logging
    
    log_info "=== Kalypso Scheduler Bootstrap ===" "main"
    log_info "Starting bootstrap process..." "main"
    
    # Parse command line arguments
    if ! parse_arguments "$@"; then
        log_error "Failed to parse arguments" "main"
        show_usage
        return 1
    fi
    
    # Check if help was requested
    if [[ "${SHOW_HELP:-false}" == "true" ]]; then
        show_usage
        return 0
    fi
    
    # Step 1: Validate prerequisites
    log_step "Checking prerequisites"
    if ! check_all_prerequisites; then
        log_error "Prerequisites check failed" "main"
        log_error "Please install required tools and try again" "main"
        return 1
    fi
    log_success "All prerequisites satisfied"
    
    # Step 2: Load and validate configuration
    log_step "Loading configuration"
    if ! load_configuration; then
        log_error "Configuration loading failed" "main"
        return 1
    fi
    
    if ! validate_configuration; then
        log_error "Configuration validation failed" "main"
        return 1
    fi
    log_success "Configuration validated"
    
    # Step 3: Show configuration and get confirmation (if interactive)
    if [[ "${INTERACTIVE_MODE:-true}" == "true" ]]; then
        display_configuration
        if ! confirm_proceed; then
            log_info "Bootstrap cancelled by user" "main"
            return 0
        fi
    fi
    
    # Step 4: Authenticate with Azure and GitHub
    log_step "Validating authentication"
    if ! validate_authentication; then
        log_error "Authentication validation failed" "main"
        return 1
    fi
    log_success "Authentication validated"
    
    # Step 5: Cluster setup
    log_step "Setting up Kubernetes cluster"
    if ! setup_cluster; then
        log_error "Cluster setup failed" "main"
        handle_error "cluster_setup"
        return 1
    fi
    log_success "Cluster ready"
    
    # Step 6: Repository setup
    log_step "Setting up repositories"
    if ! setup_repositories; then
        log_error "Repository setup failed" "main"
        handle_error "repository_setup"
        return 1
    fi
    log_success "Repositories ready"
    
    # Step 7: Install Kalypso Scheduler
    log_step "Installing Kalypso Scheduler"
    if ! install_kalypso; then
        log_error "Kalypso installation failed" "main"
        handle_error "kalypso_install"
        return 1
    fi
    log_success "Kalypso Scheduler installed"
    
    # Step 8: Verify installation
    log_step "Verifying installation"
    if ! verify_installation; then
        log_warning "Installation verification had warnings" "main"
        log_info "Kalypso may still be starting up. Check status with: kubectl get pods -n kalypso-system" "main"
    else
        log_success "Installation verified"
    fi
    
    # Step 9: Display success message and next steps
    display_success_message
    
    log_success "Bootstrap completed successfully!"
    return 0
}

#######################################
# Handle errors and optionally trigger rollback
# Globals:
#   CREATED_RESOURCES
# Arguments:
#   $1 - Error context
# Returns:
#   None
#######################################
handle_error() {
    local context="$1"
    
    log_error "Error during: $context" "error_handler"
    
    # Check if we should attempt rollback
    if [[ "${AUTO_ROLLBACK:-false}" == "true" ]]; then
        log_warning "Attempting rollback of created resources..." "error_handler"
        rollback_resources
    else
        log_info "To clean up manually, review created resources:" "error_handler"
        display_created_resources
        log_info "Run with --auto-rollback to automatically clean up on failure" "error_handler"
    fi
}

#######################################
# Display success message and next steps
# Globals:
#   CLUSTER_NAME
#   CONTROL_PLANE_REPO
#   GITOPS_REPO
# Arguments:
#   None
# Returns:
#   None
#######################################
display_success_message() {
    cat <<EOF

╔════════════════════════════════════════════════════════════════════════════╗
║                   KALYPSO SCHEDULER BOOTSTRAP SUCCESS                      ║
╚════════════════════════════════════════════════════════════════════════════╝

Your Kalypso Scheduler environment is ready!

CLUSTER:
  Name: ${CLUSTER_NAME:-N/A}
  Resource Group: ${RESOURCE_GROUP:-N/A}
  
REPOSITORIES:
  Control Plane: ${CONTROL_PLANE_REPO:-N/A}
  GitOps: ${GITOPS_REPO:-N/A}

NEXT STEPS:
  1. Review the generated documentation in docs/bootstrap/
  2. Configure kubectl context:
     $ az aks get-credentials --resource-group ${RESOURCE_GROUP} --name ${CLUSTER_NAME}
  
  3. Verify Kalypso is running:
     $ kubectl get pods -n kalypso-system
  
  4. Deploy your first workload:
     - See example workloads in: ${CONTROL_PLANE_REPO:-your-control-plane-repo}
     - Follow the quickstart guide: docs/bootstrap/quickstart.md

For troubleshooting and additional configuration:
  - Documentation: docs/bootstrap/README.md
  - Support: ${SUPPORT_URL:-https://github.com/microsoft/kalypso-scheduler/issues}

EOF
}

# Run main function if script is executed (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
