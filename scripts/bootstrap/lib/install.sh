#!/usr/bin/env bash
# Installation functions for Kalypso Scheduler
# Handles Helm chart installation and verification

#######################################
# Install Kalypso Scheduler using Helm
# Globals:
#   CONTROL_PLANE_REPO, GITOPS_REPO
# Arguments:
#   None
# Returns:
#   0 on success, 1 on error
#######################################
install_kalypso() {
    local namespace="kalypso-system"
    local release_name="kalypso-scheduler"
    
    log_info "Installing Kalypso Scheduler..." "install"
    
    # Add Kalypso Helm repository (stub - actual repo TBD)
    log_info "Configuring Helm repository..." "install"
    
    # For now, assume we're installing from local chart in the repo
    local chart_path
    chart_path="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../helm/kalypso-scheduler" && pwd)"
    
    if [[ ! -d "$chart_path" ]]; then
        log_error "Helm chart not found at: $chart_path" "install"
        return 1
    fi
    
    # Install or upgrade Kalypso
    log_info "Installing Helm chart..." "install"
    
    if ! helm upgrade --install "$release_name" "$chart_path" \
        --namespace "$namespace" \
        --create-namespace \
        --wait \
        --timeout 5m \
        --set controlPlaneRepo="$CONTROL_PLANE_REPO" \
        --set gitopsRepo="$GITOPS_REPO"; then
        log_error "Helm installation failed" "install"
        return 1
    fi
    
    log_success "Kalypso Scheduler installed successfully"
    return 0
}

#######################################
# Verify Kalypso installation
# Arguments:
#   None
# Returns:
#   0 if verification passes, 1 otherwise
#######################################
verify_installation() {
    local namespace="kalypso-system"
    
    log_info "Verifying Kalypso installation..." "install"
    
    # Check if namespace exists
    if ! kubectl get namespace "$namespace" &> /dev/null; then
        log_error "Namespace $namespace not found" "install"
        return 1
    fi
    
    # Check if deployment exists
    local deployment_name="kalypso-scheduler-controller-manager"
    if ! kubectl get deployment "$deployment_name" -n "$namespace" &> /dev/null; then
        log_warning "Deployment $deployment_name not found" "install"
        return 1
    fi
    
    # Check if pods are running
    local ready_replicas
    ready_replicas=$(kubectl get deployment "$deployment_name" -n "$namespace" \
        -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [[ "$ready_replicas" -lt 1 ]]; then
        log_warning "No ready replicas for $deployment_name" "install"
        return 1
    fi
    
    log_success "Kalypso Scheduler is running ($ready_replicas replicas ready)"
    
    # Check CRDs
    log_info "Checking CRDs..." "install"
    local crds_found=0
    local expected_crds=(
        "workloads.scheduler.kalypso.io"
        "deploymenttargets.scheduler.kalypso.io"
        "schedulingpolicies.scheduler.kalypso.io"
    )
    
    for crd in "${expected_crds[@]}"; do
        if kubectl get crd "$crd" &> /dev/null; then
            log_success "CRD found: $crd"
            crds_found=$((crds_found + 1))
        else
            log_warning "CRD not found: $crd" "install"
        fi
    done
    
    if [[ $crds_found -eq 0 ]]; then
        log_error "No Kalypso CRDs found" "install"
        return 1
    fi
    
    return 0
}

#######################################
# Rollback Kalypso installation
# Arguments:
#   None
# Returns:
#   None
#######################################
rollback_kalypso() {
    local namespace="kalypso-system"
    local release_name="kalypso-scheduler"
    
    log_warning "Rolling back Kalypso installation..." "install"
    
    # Uninstall Helm release
    if helm list -n "$namespace" | grep -q "$release_name"; then
        log_info "Uninstalling Helm release: $release_name" "install"
        helm uninstall "$release_name" -n "$namespace" || true
    fi
    
    # Delete namespace
    if kubectl get namespace "$namespace" &> /dev/null; then
        log_info "Deleting namespace: $namespace" "install"
        kubectl delete namespace "$namespace" --timeout=60s || true
    fi
}

#######################################
# Main rollback function for all resources
# Globals:
#   CREATED_RESOURCES
# Arguments:
#   None
# Returns:
#   None
#######################################
rollback_resources() {
    log_warning "Starting rollback of created resources..." "rollback"
    
    # Rollback in reverse order: install -> repos -> cluster
    rollback_kalypso
    
    if declare -f rollback_repositories &> /dev/null; then
        rollback_repositories
    fi
    
    if declare -f rollback_cluster &> /dev/null; then
        rollback_cluster
    fi
    
    log_info "Rollback completed" "rollback"
}

# Export functions
export -f install_kalypso verify_installation rollback_kalypso rollback_resources
