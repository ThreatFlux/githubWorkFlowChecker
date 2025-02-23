#!/bin/bash
set -e

# Setup signal handlers
trap 'kill -TERM $PID' TERM INT

# Main execution
main() {
    echo "Starting ThreatFlux GitHub Workflow Checker..."
    if [[ -z "${GITHUB_OUTPUT}" ]]; then
        export GITHUB_OUTPUT="github_output_${RANDOM}"
        touch "${GITHUB_OUTPUT}"
        chmod 777 "${GITHUB_OUTPUT}"
    fi
    time=$(date)
    echo "time=$time" >> $GITHUB_OUTPUT

    # Run the main application
    exec /app/ghactions-updater "$@" &

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
