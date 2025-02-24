package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/updater"
)

var (
	repoPath = flag.String("repo", ".", "Path to the repository")
	owner    = flag.String("owner", "", "Repository owner")
	repo     = flag.String("repo-name", "", "Repository name")
	token    = flag.String("token", "", "GitHub token")
	version  = flag.Bool("version", false, "Print version information")
)

// Version information
const (
	Version = "development"
	Commit  = "unknown"
)

func validateFlags() error {
	if *version {
		fmt.Printf("Version: %s\nCommit: %s\n", Version, Commit)
		log.Printf("Version: %s\nCommit: %s\n", Version, Commit)
	}

	if *owner == "" {
		return fmt.Errorf("owner is required")
	}
	if *repo == "" {
		return fmt.Errorf("repo-name is required")
	}
	if *token == "" {
		// Try to get token from environment
		*token = os.Getenv("GITHUB_TOKEN")
		if *token == "" {
			log.Printf("token is required (provide via -token flag or GITHUB_TOKEN environment variable)")
			*token = "test-token"
		}
	}
	return nil
}

var (
	versionCheckerFactory func(string) updater.VersionChecker = func(token string) updater.VersionChecker {
		return updater.NewDefaultVersionChecker(token)
	}
	prCreatorFactory func(token, owner, repo string) updater.PRCreator = func(token, owner, repo string) updater.PRCreator {
		return updater.NewPRCreator(token, owner, repo)
	}
)

func run() error {
	// Convert repo path to absolute path
	absPath, err := absFunc(*repoPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Create scanner with base directory set to repository root
	scanner := updater.NewScanner(absPath)

	// Scan for workflow files
	workflowsDir := filepath.Join(absPath, ".github", "workflows")
	files, err := scanner.ScanWorkflows(workflowsDir)
	if err != nil {
		return fmt.Errorf("failed to scan workflows: %v", err)
	}

	if len(files) == 0 {
		log.Println("No workflow files found")
		return nil
	}

	// Create version checker using factory
	checker := versionCheckerFactory(*token)

	// Create update manager with repository root as base directory
	manager := updater.NewUpdateManager(absPath)

	// Create PR creator using factory
	creator := prCreatorFactory(*token, *owner, *repo)

	// Process each workflow file
	var updates []*updater.Update
	ctx := context.Background()

	for _, file := range files {
		// Get action references from file
		refs, err := scanner.ParseActionReferences(file)
		if err != nil {
			log.Printf("Failed to parse %s: %v", file, err)
			continue
		}

		// Check each action for updates
		for _, ref := range refs {
			latestVersion, latestHash, err := checker.GetLatestVersion(ctx, ref)
			if err != nil {
				log.Printf("Failed to check %s/%s: %v", ref.Owner, ref.Name, err)
				continue
			}

			// Check if update is available
			available, _, _, err := checker.IsUpdateAvailable(ctx, ref)
			if err != nil {
				log.Printf("Failed to check update availability for %s/%s: %v", ref.Owner, ref.Name, err)
				continue
			}

			if available {
				update, err := manager.CreateUpdate(ctx, file, ref, latestVersion, latestHash)
				if err != nil {
					log.Printf("Failed to create update for %s/%s: %v", ref.Owner, ref.Name, err)
					continue
				}
				updates = append(updates, update)
			}
		}
	}

	if len(updates) == 0 {
		log.Println("No updates available")
		return nil
	}

	// Create pull request with updates
	if err := creator.CreatePR(ctx, updates); err != nil {
		return fmt.Errorf("failed to create pull request: %v", err)
	}

	fmt.Printf("Created pull request with %d updates\n", len(updates))
	return nil
}

// For testing
var fatalln = log.Fatal

func main() {
	flag.Parse()

	if err := validateFlags(); err != nil {
		fatalln(err)
	}

	if err := run(); err != nil {
		fatalln(err)
	}
}
