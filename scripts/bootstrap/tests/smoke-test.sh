#!/usr/bin/env bash
# Basic smoke tests for bootstrap script
# These tests verify basic functionality without requiring Azure/GitHub credentials

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_FAILED=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

#######################################
# Print test result
# Arguments:
#   $1 - Test name
#   $2 - Result (0=pass, 1=fail)
#######################################
print_result() {
    local test_name="$1"
    local result="$2"
    
    if [[ $result -eq 0 ]]; then
        echo -e "${GREEN}✓${NC} $test_name"
    else
        echo -e "${RED}✗${NC} $test_name"
        TEST_FAILED=1
    fi
}

#######################################
# Test: Script files exist
#######################################
test_files_exist() {
    local result=0
    
    [[ -f "$SCRIPT_DIR/bootstrap.sh" ]] || result=1
    [[ -f "$SCRIPT_DIR/lib/utils.sh" ]] || result=1
    [[ -f "$SCRIPT_DIR/lib/prerequisites.sh" ]] || result=1
    [[ -f "$SCRIPT_DIR/lib/config.sh" ]] || result=1
    [[ -f "$SCRIPT_DIR/lib/cluster.sh" ]] || result=1
    [[ -f "$SCRIPT_DIR/lib/repositories.sh" ]] || result=1
    [[ -f "$SCRIPT_DIR/lib/install.sh" ]] || result=1
    
    print_result "All required script files exist" "$result"
}

#######################################
# Test: Scripts are executable
#######################################
test_executable() {
    local result=0
    
    [[ -x "$SCRIPT_DIR/bootstrap.sh" ]] || result=1
    
    print_result "Main script is executable" "$result"
}

#######################################
# Test: Scripts have valid syntax
#######################################
test_syntax() {
    local result=0
    
    bash -n "$SCRIPT_DIR/bootstrap.sh" 2>/dev/null || result=1
    bash -n "$SCRIPT_DIR/lib/utils.sh" 2>/dev/null || result=1
    bash -n "$SCRIPT_DIR/lib/prerequisites.sh" 2>/dev/null || result=1
    bash -n "$SCRIPT_DIR/lib/config.sh" 2>/dev/null || result=1
    bash -n "$SCRIPT_DIR/lib/cluster.sh" 2>/dev/null || result=1
    bash -n "$SCRIPT_DIR/lib/repositories.sh" 2>/dev/null || result=1
    bash -n "$SCRIPT_DIR/lib/install.sh" 2>/dev/null || result=1
    
    print_result "All scripts have valid bash syntax" "$result"
}

#######################################
# Test: Help message works
#######################################
test_help() {
    local result=0
    
    if "$SCRIPT_DIR/bootstrap.sh" --help 2>&1 | grep -q "USAGE:"; then
        result=0
    else
        result=1
    fi
    
    print_result "Help message displays correctly" "$result"
}

#######################################
# Test: Library functions can be sourced
#######################################
test_source_libs() {
    local result=0
    
    # Try sourcing each library
    if ! (source "$SCRIPT_DIR/lib/utils.sh" 2>/dev/null); then
        result=1
    fi
    
    if ! (
        source "$SCRIPT_DIR/lib/utils.sh" 2>/dev/null
        source "$SCRIPT_DIR/lib/prerequisites.sh" 2>/dev/null
    ); then
        result=1
    fi
    
    if ! (
        source "$SCRIPT_DIR/lib/utils.sh" 2>/dev/null
        source "$SCRIPT_DIR/lib/config.sh" 2>/dev/null
    ); then
        result=1
    fi
    
    print_result "Library files can be sourced" "$result"
}

#######################################
# Test: Utility functions work
#######################################
test_utils() {
    local result=0
    
    # Source utils
    source "$SCRIPT_DIR/lib/utils.sh" 2>/dev/null || result=1
    
    # Test is_empty function
    if ! is_empty ""; then
        result=1
    fi
    
    if is_empty "not empty"; then
        result=1
    fi
    
    # Test trim function
    local trimmed
    trimmed=$(trim "  test  ")
    if [[ "$trimmed" != "test" ]]; then
        result=1
    fi
    
    # Test to_lower function
    local lower
    lower=$(to_lower "TEST")
    if [[ "$lower" != "test" ]]; then
        result=1
    fi
    
    print_result "Utility functions work correctly" "$result"
}

#######################################
# Test: Documentation files exist
#######################################
test_docs() {
    local result=0
    local doc_dir
    doc_dir="$(cd "$SCRIPT_DIR/../../docs/bootstrap" && pwd)"
    
    [[ -f "$doc_dir/README.md" ]] || result=1
    [[ -f "$doc_dir/prerequisites.md" ]] || result=1
    [[ -f "$doc_dir/troubleshooting.md" ]] || result=1
    [[ -f "$doc_dir/quickstart.md" ]] || result=1
    
    print_result "Documentation files exist" "$result"
}

#######################################
# Test: Configuration template exists
#######################################
test_template() {
    local result=0
    
    [[ -f "$SCRIPT_DIR/templates/config.yaml" ]] || result=1
    
    print_result "Configuration template exists" "$result"
}

#######################################
# Test: ShellCheck config exists
#######################################
test_shellcheck_config() {
    local result=0
    
    [[ -f "$SCRIPT_DIR/.shellcheckrc" ]] || result=1
    
    print_result "ShellCheck configuration exists" "$result"
}

#######################################
# Main test runner
#######################################
main() {
    echo "Running bootstrap script smoke tests..."
    echo ""
    
    test_files_exist
    test_executable
    test_syntax
    test_help
    test_source_libs
    test_utils
    test_docs
    test_template
    test_shellcheck_config
    
    echo ""
    if [[ $TEST_FAILED -eq 0 ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
        return 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        return 1
    fi
}

# Run tests
main
