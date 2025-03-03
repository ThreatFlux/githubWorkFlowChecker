package updater

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/ThreatFlux/githubWorkFlowChecker/pkg/common"
	"github.com/google/go-github/v58/github"
)

// DefaultPRCreator implements the PRCreator interface
type DefaultPRCreator struct {
	client        *github.Client
	owner         string
	repo          string
	workflowsPath string // Path to workflow files (relative to repository root)
}

// NewPRCreator creates a new instance of DefaultPRCreator
func NewPRCreator(token, owner, repo string) *DefaultPRCreator {
	client := common.NewGitHubClientWithToken(token)

	return &DefaultPRCreator{
		client:        client,
		owner:         owner,
		repo:          repo,
		workflowsPath: ".github/workflows", // Default path
	}
}

// SetWorkflowsPath sets the path to workflow files
func (c *DefaultPRCreator) SetWorkflowsPath(path string) {
	c.workflowsPath = path
}

// formatRelativePath converts an absolute file path to a repository-relative path
func (c *DefaultPRCreator) formatRelativePath(file string) string {
	relPath := file
	if filepath.IsAbs(relPath) {
		// Extract the workflows path part of the path
		parts := strings.Split(relPath, c.workflowsPath)
		if len(parts) != 2 {
			// If we can't find the workflows path, just use the file name
			relPath = filepath.Base(file)
		} else {
			// Join the workflows path with the file path
			relPath = filepath.Join(c.workflowsPath, strings.TrimPrefix(parts[1], "/"))
		}
	}
	return relPath
}

// CreatePR creates a pull request with the given updates
func (c *DefaultPRCreator) CreatePR(ctx context.Context, updates []*Update) error {
	if len(updates) == 0 {
		return nil
	}

	// Create a new branch for the updates
	branchName := fmt.Sprintf("action-updates-%s", time.Now().Format("20060102-150405"))
	if err := c.createBranch(ctx, branchName); err != nil {
		return fmt.Errorf("error creating branch: %w", err)
	}

	// Create commit with all updates
	if err := c.createCommit(ctx, branchName, updates); err != nil {
		return fmt.Errorf("error creating commit: %w", err)
	}

	// Create pull request
	title := "Update GitHub Actions dependencies"
	body := c.generatePRBody(updates)

	pr, _, err := c.client.PullRequests.Create(ctx, c.owner, c.repo, &github.NewPullRequest{
		Title: &title,
		Body:  &body,
		Head:  &branchName,
		Base:  github.String("main"),
	})

	if err != nil {
		return fmt.Errorf("error creating pull request: %w", err)
	}

	// Add labels if PR was created successfully
	if pr.Number != nil {
		_, _, err = c.client.Issues.AddLabelsToIssue(ctx, c.owner, c.repo, *pr.Number,
			[]string{"dependencies", "automated-pr"})
		if err != nil {
			// Don't fail if we couldn't add labels
			fmt.Printf("Warning: could not add labels to PR: %v\n", err)
		}
	}

	return nil
}

// createBranch creates a new branch from the default branch
func (c *DefaultPRCreator) createBranch(ctx context.Context, branchName string) error {
	// Get the default branch's latest commit
	repo, _, err := c.client.Repositories.Get(ctx, c.owner, c.repo)
	if err != nil {
		return fmt.Errorf("error getting repository: %w", err)
	}

	defaultBranch := repo.GetDefaultBranch()
	ref, _, err := c.client.Git.GetRef(ctx, c.owner, c.repo, "refs/heads/"+defaultBranch)
	if err != nil {
		return fmt.Errorf("error getting default branch ref: %w", err)
	}

	// Create new branch
	newRef := &github.Reference{
		Ref:    github.String("refs/heads/" + branchName),
		Object: ref.Object,
	}

	_, _, err = c.client.Git.CreateRef(ctx, c.owner, c.repo, newRef)
	return err
}

// formatActionReference formats an action reference with version comments
func (c *DefaultPRCreator) formatActionReference(update *Update) string {
	var sb strings.Builder

	// Add the action reference with hash
	// Handle multi-part action names correctly (e.g., github/codeql-action/init)
	actionFullName := update.Action.Owner + "/" + update.Action.Name
	sb.WriteString(fmt.Sprintf("%s@%s", actionFullName, update.NewHash))

	// Add current version comment
	if update.NewVersion != "" {
		sb.WriteString(fmt.Sprintf("  # %s", update.NewVersion))
	}

	return sb.String()
}

// createCommit creates a commit with all updates
func (c *DefaultPRCreator) createCommit(ctx context.Context, branch string, updates []*Update) error {
	// Group updates by file
	fileUpdates := make(map[string][]*Update)
	for _, update := range updates {
		fileUpdates[update.FilePath] = append(fileUpdates[update.FilePath], update)
	}

	// Create tree entries for each file
	var entries []*github.TreeEntry
	for file, fileUpdates := range fileUpdates {
		// Convert absolute path to repository-relative path
		relPath := c.formatRelativePath(file)

		// Get current file content
		content, _, _, err := c.client.Repositories.GetContents(ctx, c.owner, c.repo, relPath,
			&github.RepositoryContentGetOptions{Ref: branch})
		if err != nil {
			// If file doesn't exist in the repository yet, create empty content
			if strings.Contains(err.Error(), "404") {
				content = &github.RepositoryContent{
					Content: github.String(""),
				}
			} else {
				return fmt.Errorf("error getting file contents: %w", err)
			}
		}

		// Apply updates to content
		fileContent, err := content.GetContent()
		if err != nil {
			return fmt.Errorf("error decoding content: %w", err)
		}

		lines := strings.Split(fileContent, "\n")
		for _, update := range fileUpdates {
			// Find the line with the action reference
			lineIdx := update.LineNumber - 1
			if lineIdx >= 0 && lineIdx < len(lines) {
				// Get the line and preserve indentation and structure
				line := lines[update.LineNumber-1]

				// Extract indentation (whitespace at the beginning of the line)
				indentation := ""
				for i, c := range line {
					if !unicode.IsSpace(c) {
						indentation = line[:i]
						break
					}
				}

				// Check if the line starts with "- name:" which indicates it's a step definition
				isStepDefinition := strings.Contains(line, "- name:")

				// Apply the update with improved formatting
				parts := strings.SplitN(line, "#", 2)
				mainPart := strings.TrimSpace(parts[0])

				// Check if the line contains "uses:" to avoid duplication
				usesIdx := strings.Index(mainPart, "uses:")

				// Format the action reference with the new hash
				newRef := c.formatActionReference(update)

				var newLine string

				if usesIdx >= 0 {
					// Case 1: Line contains "uses:" - preserve the format
					beforeUses := mainPart[:usesIdx+5] // +5 to include "uses:"

					// Add version comment (already included in newRef)
					newLine = fmt.Sprintf("%s%s %s", indentation, beforeUses, strings.TrimPrefix(newRef, "uses: "))
				} else if isStepDefinition {
					// Case 2: This is a step definition line, the "uses:" line will be on the next line
					// Just keep it as is
					newLine = line
				} else {
					// Case 3: This is a line that should have "uses:" but doesn't (possibly already processed incorrectly)
					// Add proper indentation and "uses:" prefix
					// Check if this is a step line (should start with "- " or "  - ")
					if strings.Contains(line, "- name:") {
						// This is a step definition line, keep it as is
						newLine = line
					} else if strings.HasPrefix(strings.TrimSpace(line), "-") {
						// This is a step line but not a name line, it should have proper indentation
						newLine = fmt.Sprintf("%s      uses: %s", indentation, strings.TrimPrefix(newRef, "uses: "))
					} else {
						// This is some other line, add standard indentation
						newLine = fmt.Sprintf("%s  %s", indentation, newRef)
					}
				}

				lines[lineIdx] = newLine
			}
		}
		fileContent = strings.Join(lines, "\n")

		// Create blob for updated content
		blob, _, err := c.client.Git.CreateBlob(ctx, c.owner, c.repo, &github.Blob{
			Content:  github.String(fileContent),
			Encoding: github.String("utf-8"),
		})
		if err != nil {
			return fmt.Errorf("error creating blob: %w", err)
		}

		// Ensure path doesn't start with a slash
		relPath = strings.TrimPrefix(relPath, "/")

		entries = append(entries, &github.TreeEntry{
			Path: github.String(relPath),
			Mode: github.String("100644"),
			Type: github.String("blob"),
			SHA:  blob.SHA,
		})
	}

	// Get the branch's latest commit
	ref, _, err := c.client.Git.GetRef(ctx, c.owner, c.repo, "refs/heads/"+branch)
	if err != nil {
		return fmt.Errorf("error getting branch ref: %w", err)
	}

	// Create tree
	tree, _, err := c.client.Git.CreateTree(ctx, c.owner, c.repo, *ref.Object.SHA, entries)
	if err != nil {
		return fmt.Errorf("error creating tree: %w", err)
	}

	// Create commit
	commit, _, err := c.client.Git.CreateCommit(ctx, c.owner, c.repo, &github.Commit{
		Message: github.String(c.generateCommitMessage(updates)),
		Tree:    tree,
		Parents: []*github.Commit{{SHA: ref.Object.SHA}},
	}, &github.CreateCommitOptions{})
	if err != nil {
		return fmt.Errorf("error creating commit: %w", err)
	}

	// Update branch reference
	ref.Object.SHA = commit.SHA
	_, _, err = c.client.Git.UpdateRef(ctx, c.owner, c.repo, ref, false)
	return err
}

// generateCommitMessage generates a commit message for the updates
func (c *DefaultPRCreator) generateCommitMessage(updates []*Update) string {
	var sb strings.Builder
	sb.WriteString("Update GitHub Actions dependencies\n\n")
	for _, update := range updates {
		sb.WriteString(fmt.Sprintf("* %s\n", update.Description))
	}
	return sb.String()
}

// generatePRBody generates the body text for the pull request
func (c *DefaultPRCreator) generatePRBody(updates []*Update) string {
	var sb strings.Builder
	sb.WriteString("This PR updates the following GitHub Actions to their latest versions:\n\n")

	for _, update := range updates {
		// Handle multi-part action names correctly (e.g., github/codeql-action/init)
		actionFullName := update.Action.Owner + "/" + update.Action.Name
		sb.WriteString(fmt.Sprintf("* `%s`\n", actionFullName))
		sb.WriteString(fmt.Sprintf("  * From: %s (%s)\n", update.OldVersion, update.OldHash))
		sb.WriteString(fmt.Sprintf("  * To: %s (%s)\n", update.NewVersion, update.NewHash))
		if update.OriginalVersion != "" && update.OriginalVersion != update.OldVersion {
			sb.WriteString(fmt.Sprintf("  * Original version: %s\n", update.OriginalVersion))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n")
	sb.WriteString("🔒 This PR uses commit hashes for improved security.\n")
	sb.WriteString("🤖 This PR was created automatically by the GitHub Actions workflow updater.")
	return sb.String()
}
