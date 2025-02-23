#!/bin/bash
set -e

# Setup signal handlers
trap 'kill -TERM $PID' TERM INT

# Function to validate environment variables
validate_env() {
    local required_vars=("GITHUB_TOKEN")
    local missing_vars=()

    for var in "${required_vars[@]}"; do
        if [[ -z "${!var}" ]]; then
            missing_vars+=("$var")
        fi
    done

    if [[ ${#missing_vars[@]} -ne 0 ]]; then
        echo "Error: Required environment variables are not set: ${missing_vars[*]}"
        exit 1
    fi
}

# Function to check GitHub API access
check_github_access() {
    if ! curl -s -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user > /dev/null; then
        echo "Error: Unable to authenticate with GitHub. Please check your token."
        exit 1
    fi
}

# Main execution
main() {
    echo "Starting ThreatFlux GitHub Workflow Checker..."
    
    # Validate environment variables
    validate_env
    
    # Check GitHub API access
    check_github_access
    
    # Run the main application
    exec /app/githubWorkFlowChecker "$@" &
    
    # Store PID for signal handling
    PID=$!
    
    # Wait for the process to complete
    wait $PID
    
    # Capture exit code
    exit_code=$?
    
    # Exit with the same code as the main process
    exit $exit_code
}

# Run main function with all arguments passed to the script
main "$@"