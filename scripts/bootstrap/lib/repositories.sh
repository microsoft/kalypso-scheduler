#!/usr/bin/env bash
# Repository management for Kalypso bootstrapping script
# Handles GitHub repository creation and configuration

#######################################
# Create control-plane repository
# Globals:
#   GITHUB_TOKEN, GITHUB_ORG, GITHUB_USER
# Arguments:
#   None
# Returns:
#   0 on success, 1 on error
#######################################
create_control_plane_repo() {
    local repo_name="${DEFAULT_CONTROL_PLANE_REPO_NAME}"
    local owner="${GITHUB_ORG:-$GITHUB_USER}"
    
    log_info "Creating control-plane repository: $owner/$repo_name" "repo"
    
    # Create repository via GitHub API
    local api_endpoint="https://api.github.com/user/repos"
    if [[ -n "$GITHUB_ORG" ]]; then
        api_endpoint="https://api.github.com/orgs/$GITHUB_ORG/repos"
    fi
    
    local response
    response=$(curl -s -X POST "$api_endpoint" \
        -H "Authorization: token $GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        -d "{\"name\":\"$repo_name\",\"description\":\"Kalypso control-plane configuration\",\"private\":false}")
    
    if echo "$response" | grep -q "\"full_name\""; then
        local repo_url
        repo_url=$(echo "$response" | json_get_value "html_url")
        CONTROL_PLANE_REPO="$repo_url"
        track_created_resource "github-repo:$owner/$repo_name"
        log_success "Control-plane repository created: $repo_url"
    else
        log_error "Failed to create repository: $response" "repo"
        return 1
    fi
    
    # Initialize repository with minimal structure
    if ! initialize_control_plane_repo "$owner" "$repo_name"; then
        log_error "Failed to initialize control-plane repository" "repo"
        return 1
    fi
    
    return 0
}

#######################################
# Create gitops repository
# Globals:
#   GITHUB_TOKEN, GITHUB_ORG, GITHUB_USER
# Arguments:
#   None
# Returns:
#   0 on success, 1 on error
#######################################
create_gitops_repo() {
    local repo_name="${DEFAULT_GITOPS_REPO_NAME}"
    local owner="${GITHUB_ORG:-$GITHUB_USER}"
    
    log_info "Creating gitops repository: $owner/$repo_name" "repo"
    
    # Create repository via GitHub API
    local api_endpoint="https://api.github.com/user/repos"
    if [[ -n "$GITHUB_ORG" ]]; then
        api_endpoint="https://api.github.com/orgs/$GITHUB_ORG/repos"
    fi
    
    local response
    response=$(curl -s -X POST "$api_endpoint" \
        -H "Authorization: token $GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        -d "{\"name\":\"$repo_name\",\"description\":\"Kalypso GitOps repository\",\"private\":false}")
    
    if echo "$response" | grep -q "\"full_name\""; then
        local repo_url
        repo_url=$(echo "$response" | json_get_value "html_url")
        GITOPS_REPO="$repo_url"
        track_created_resource "github-repo:$owner/$repo_name"
        log_success "GitOps repository created: $repo_url"
    else
        log_error "Failed to create repository: $response" "repo"
        return 1
    fi
    
    # Initialize repository with minimal structure
    if ! initialize_gitops_repo "$owner" "$repo_name"; then
        log_error "Failed to initialize gitops repository" "repo"
        return 1
    fi
    
    return 0
}

#######################################
# Initialize control-plane repository with minimal structure
# Arguments:
#   $1 - Repository owner
#   $2 - Repository name
# Returns:
#   0 on success, 1 on error
#######################################
initialize_control_plane_repo() {
    local owner="$1"
    local repo_name="$2"
    local temp_dir
    
    temp_dir=$(mktemp -d)
    trap 'rm -rf "$temp_dir"' RETURN
    
    log_info "Initializing control-plane repository structure..." "repo"
    
    cd "$temp_dir" || return 1
    
    # Initialize git repo
    git init &> /dev/null
    git config user.name "Kalypso Bootstrap" &> /dev/null
    git config user.email "bootstrap@kalypso.local" &> /dev/null
    
    # Create minimal structure (stub for now)
    mkdir -p environments/dev cluster-types scheduling-policies config-maps
    
    # Create README
    cat > README.md <<EOF
# Kalypso Control Plane

This repository contains Kalypso Scheduler configuration resources.

Created by Kalypso bootstrapping script.

## Structure

- \`environments/\` - Environment definitions
- \`cluster-types/\` - Cluster type configurations
- \`scheduling-policies/\` - Scheduling policy definitions
- \`config-maps/\` - Configuration maps

For more information, see: https://github.com/microsoft/kalypso-scheduler
EOF
    
    # Create initial environment (minimal placeholder)
    cat > environments/dev/environment.yaml <<EOF
apiVersion: scheduler.kalypso.io/v1alpha1
kind: Environment
metadata:
  name: dev
  namespace: kalypso-system
spec:
  description: Development environment
EOF
    
    # Commit and push
    git add . &> /dev/null
    git commit -m "Initial commit - Kalypso control plane structure" &> /dev/null
    git branch -M main &> /dev/null
    git remote add origin "https://${GITHUB_TOKEN}@github.com/${owner}/${repo_name}.git" &> /dev/null
    
    if ! git push -u origin main &> /dev/null; then
        log_error "Failed to push to control-plane repository" "repo"
        return 1
    fi
    
    log_success "Control-plane repository initialized"
    return 0
}

#######################################
# Initialize gitops repository with minimal structure
# Arguments:
#   $1 - Repository owner
#   $2 - Repository name
# Returns:
#   0 on success, 1 on error
#######################################
initialize_gitops_repo() {
    local owner="$1"
    local repo_name="$2"
    local temp_dir
    
    temp_dir=$(mktemp -d)
    trap 'rm -rf "$temp_dir"' RETURN
    
    log_info "Initializing gitops repository structure..." "repo"
    
    cd "$temp_dir" || return 1
    
    # Initialize git repo
    git init &> /dev/null
    git config user.name "Kalypso Bootstrap" &> /dev/null
    git config user.email "bootstrap@kalypso.local" &> /dev/null
    
    # Create minimal structure
    mkdir -p .github/workflows clusters
    
    # Create README
    cat > README.md <<EOF
# Kalypso GitOps

This repository contains Kalypso deployment manifests and GitHub workflows.

Created by Kalypso bootstrapping script.

## Structure

- \`.github/workflows/\` - GitHub Actions workflows
- \`clusters/\` - Cluster-specific deployment manifests

For more information, see: https://github.com/microsoft/kalypso-scheduler
EOF
    
    # Create placeholder workflow
    cat > .github/workflows/sync.yaml <<EOF
name: GitOps Sync
on:
  push:
    branches: [main]
jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Sync manifests
        run: echo "GitOps sync placeholder"
EOF
    
    # Commit and push
    git add . &> /dev/null
    git commit -m "Initial commit - Kalypso gitops structure" &> /dev/null
    git branch -M main &> /dev/null
    git remote add origin "https://${GITHUB_TOKEN}@github.com/${owner}/${repo_name}.git" &> /dev/null
    
    if ! git push -u origin main &> /dev/null; then
        log_error "Failed to push to gitops repository" "repo"
        return 1
    fi
    
    log_success "GitOps repository initialized"
    return 0
}

#######################################
# Validate existing control-plane repository
# Globals:
#   CONTROL_PLANE_REPO
# Arguments:
#   None
# Returns:
#   0 if valid, 1 otherwise
#######################################
validate_control_plane_repo() {
    log_info "Validating control-plane repository: $CONTROL_PLANE_REPO" "repo"
    
    # Basic validation - check if repo is accessible
    local repo_path
    repo_path="${CONTROL_PLANE_REPO#https://github.com/}"
    
    local response
    response=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
        "https://api.github.com/repos/$repo_path")
    
    if echo "$response" | grep -q "\"full_name\""; then
        log_success "Control-plane repository validated"
        return 0
    else
        log_error "Cannot access control-plane repository" "repo"
        return 1
    fi
}

#######################################
# Validate existing gitops repository
# Globals:
#   GITOPS_REPO
# Arguments:
#   None
# Returns:
#   0 if valid, 1 otherwise
#######################################
validate_gitops_repo() {
    log_info "Validating gitops repository: $GITOPS_REPO" "repo"
    
    # Basic validation - check if repo is accessible
    local repo_path
    repo_path="${GITOPS_REPO#https://github.com/}"
    
    local response
    response=$(curl -s -H "Authorization: token $GITHUB_TOKEN" \
        "https://api.github.com/repos/$repo_path")
    
    if echo "$response" | grep -q "\"full_name\""; then
        log_success "GitOps repository validated"
        return 0
    else
        log_error "Cannot access gitops repository" "repo"
        return 1
    fi
}

#######################################
# Setup repositories (create or validate existing)
# Globals:
#   CREATE_REPOS
# Arguments:
#   None
# Returns:
#   0 on success, 1 on error
#######################################
setup_repositories() {
    if [[ "$CREATE_REPOS" == "true" ]]; then
        # Create new repositories
        if ! create_control_plane_repo; then
            return 1
        fi
        
        if ! create_gitops_repo; then
            return 1
        fi
    else
        # Validate existing repositories
        if ! validate_control_plane_repo; then
            return 1
        fi
        
        if ! validate_gitops_repo; then
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Rollback repository creation
# Globals:
#   CREATED_RESOURCES
# Arguments:
#   None
# Returns:
#   None
#######################################
rollback_repositories() {
    log_warning "Rolling back repository resources..." "repo"
    
    for resource in "${CREATED_RESOURCES[@]}"; do
        if [[ "$resource" == github-repo:* ]]; then
            local repo="${resource#github-repo:}"
            
            log_info "Deleting GitHub repository: $repo" "repo"
            curl -s -X DELETE "https://api.github.com/repos/$repo" \
                -H "Authorization: token $GITHUB_TOKEN" \
                -H "Accept: application/vnd.github.v3+json" > /dev/null || true
        fi
    done
}

# Export functions
export -f create_control_plane_repo create_gitops_repo
export -f validate_control_plane_repo validate_gitops_repo
export -f setup_repositories rollback_repositories
