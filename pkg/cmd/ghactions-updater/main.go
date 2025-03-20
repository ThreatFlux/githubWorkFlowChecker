package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/updater"
)

var (
	repoPath      = flag.String("repo", ".", "Path to the repository")
	owner         = flag.String("owner", "", "Repository owner")
	repo          = flag.String("repo-name", "", "Repository name")
	token         = flag.String("token", "", "GitHub token")
	version       = flag.Bool("version", false, "Print version information")
	workflowsPath = flag.String("workflows-path", ".github/workflows", "Path to workflow files (relative to repository root)")
	dryRun        = flag.Bool("dry-run", false, "Show changes without applying them")
	stage         = flag.Bool("stage", false, "Apply changes locally without creating a PR")
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
		return fmt.Errorf(common.ErrMissingRequiredFlag, "owner")
	}
	if *repo == "" {
		return fmt.Errorf(common.ErrMissingRequiredFlag, "repo-name")
	}
	if *token == "" {
		// Try to get token from environment
		*token = os.Getenv("GITHUB_TOKEN")
		if *token == "" {
			log.Printf(common.ErrNoGithubToken)
			// Allow empty token - the client will use unauthenticated access
		}
	}

	// Check for environment variable override for workflows path
	if envPath := os.Getenv("WORKFLOWS_PATH"); envPath != "" {
		*workflowsPath = envPath
	}

	// Validate that dry-run and stage are not both set
	if *dryRun && *stage {
		return fmt.Errorf(common.ErrInvalidFlagValue, "dry-run/stage", "cannot use both flags simultaneously")
	}

	return nil
}

var (
	versionCheckerFactory = func(token string) updater.VersionChecker {
		return updater.NewDefaultVersionChecker(token)
	}
	prCreatorFactory = func(token, owner, repo string) updater.PRCreator {
		return updater.NewPRCreator(token, owner, repo)
	}
)

func run() error {
	// Convert repo path to absolute path
	absPath, err := absFunc(*repoPath)
	if err != nil {
		return fmt.Errorf(common.ErrCommandExecution, err)
	}

	// Create scanner with base directory set to repository root
	scanner := updater.NewScanner(absPath)

	// Scan for workflow files using configurable path
	workflowsDir := filepath.Join(absPath, *workflowsPath)
	files, err := scanner.ScanWorkflows(workflowsDir)
	if err != nil {
		return fmt.Errorf(common.ErrReadingUpdateFile, err)
	}

	if len(files) == 0 {
		log.Println(common.ErrNoWorkflowsFound)
		return nil
	}

	// Create version checker using factory
	checker := versionCheckerFactory(*token)

	// Create update manager with repository root as base directory
	manager := updater.NewUpdateManager(absPath)

	// Create PR creator using factory and set workflows path
	creator := prCreatorFactory(*token, *owner, *repo)
	if prCreatorWithPath, ok := creator.(*updater.DefaultPRCreator); ok {
		prCreatorWithPath.SetWorkflowsPath(*workflowsPath)
	}

	// Process each workflow file
	var updates []*updater.Update
	ctx := context.Background()

	for _, file := range files {
		// Get action references from file
		refs, err := scanner.ParseActionReferences(file)
		if err != nil {
			log.Printf(common.ErrFailedToParseWorkflow, file, err)
			continue
		}

		// Check each action for updates
		for _, ref := range refs {
			latestVersion, latestHash, err := checker.GetLatestVersion(ctx, ref)
			if err != nil {
				log.Printf(common.ErrFailedToCheckAction, ref.Owner, ref.Name, err)
				continue
			}

			// Check if update is available
			available, _, _, err := checker.IsUpdateAvailable(ctx, ref)
			if err != nil {
				log.Printf(common.ErrFailedToCheckUpdate, ref.Owner, ref.Name, err)
				continue
			}

			if available {
				update, err := manager.CreateUpdate(ctx, file, ref, latestVersion, latestHash)
				if err != nil {
					log.Printf(common.ErrFailedToCreateUpdate, ref.Owner, ref.Name, err)
					continue
				}
				updates = append(updates, update)
			}
		}
	}

	if len(updates) == 0 {
		log.Println(common.ErrNoUpdatesAvailable)
		return nil
	}

	// Handle updates based on mode (dry-run, stage, or normal)
	if *dryRun {
		// Preview changes without applying them
		fmt.Printf("DRY RUN: Would update %d actions in %d files\n", len(updates), countUniqueFiles(updates))
		for _, update := range updates {
			fmt.Printf("- %s: %s/%s from %s to %s\n",
				update.FilePath,
				update.Action.Owner,
				update.Action.Name,
				update.OldVersion,
				update.NewVersion)
		}
	} else if *stage {
		// Apply changes locally without creating a PR
		if err := manager.ApplyUpdates(ctx, updates); err != nil {
			return fmt.Errorf(common.ErrApplyingUpdates, err)
		}
		fmt.Printf("Applied %d updates locally to %d files\n", len(updates), countUniqueFiles(updates))
	} else {
		// Normal mode: Create pull request with updates
		if err := creator.CreatePR(ctx, updates); err != nil {
			return fmt.Errorf(common.ErrCreatingPR, err)
		}
		fmt.Printf("Created pull request with %d updates\n", len(updates))
	}
	return nil
}

// countUniqueFiles counts the number of unique files in the updates slice
func countUniqueFiles(updates []*updater.Update) int {
	uniqueFiles := make(map[string]struct{})
	for _, update := range updates {
		uniqueFiles[update.FilePath] = struct{}{}
	}
	return len(uniqueFiles)
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
