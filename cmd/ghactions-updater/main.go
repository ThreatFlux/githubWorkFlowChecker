package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// CLI flags
	repoPath := flag.String("repo", ".", "Path to the repository to check")
	token := flag.String("token", "", "GitHub token for API access")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Show version info if requested
	if *showVersion {
		fmt.Printf("ghactions-updater version %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("built at: %s\n", date)
		os.Exit(0)
	}

	// Validate GitHub token
	if *token == "" {
		token = nil
		// Try to get token from environment
		if envToken := os.Getenv("GITHUB_TOKEN"); envToken != "" {
			token = &envToken
		}
	}

	if token == nil {
		fmt.Fprintln(os.Stderr, "Error: GitHub token is required. Provide via -token flag or GITHUB_TOKEN environment variable")
		os.Exit(1)
	}

	// TODO: Initialize and run the updater
	fmt.Printf("Checking repository at: %s\n", *repoPath)
}
