package updater

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// DefaultPRCreator implements the PRCreator interface
type DefaultPRCreator struct {
	client *github.Client
	owner  string
	repo   string
}

// NewPRCreator creates a new instance of DefaultPRCreator
func NewPRCreator(token, owner, repo string) *DefaultPRCreator {
	client := github.NewClient(nil)
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		client = github.NewClient(oauth2.NewClient(context.Background(), ts))
	}

	return &DefaultPRCreator{
		client: client,
		owner:  owner,
		repo:   repo,
	}
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
		// Get current file content
		content, _, _, err := c.client.Repositories.GetContents(ctx, c.owner, c.repo, file,
			&github.RepositoryContentGetOptions{Ref: branch})
		if err != nil {
			return fmt.Errorf("error getting file contents: %w", err)
		}

		// Apply updates to content
		fileContent, err := content.GetContent()
		if err != nil {
			return fmt.Errorf("error decoding content: %w", err)
		}

		for _, update := range fileUpdates {
			oldRef := fmt.Sprintf("%s/%s@%s", update.Action.Owner, update.Action.Name, update.OldVersion)
			newRef := fmt.Sprintf("%s/%s@%s", update.Action.Owner, update.Action.Name, update.NewVersion)
			fileContent = strings.Replace(fileContent, oldRef, newRef, 1)
		}

		// Create blob for updated content
		blob, _, err := c.client.Git.CreateBlob(ctx, c.owner, c.repo, &github.Blob{
			Content:  github.String(fileContent),
			Encoding: github.String("utf-8"),
		})
		if err != nil {
			return fmt.Errorf("error creating blob: %w", err)
		}

		// Add tree entry
		entries = append(entries, &github.TreeEntry{
			Path: github.String(file),
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
		sb.WriteString(fmt.Sprintf("* `%s/%s` from %s to %s\n",
			update.Action.Owner, update.Action.Name, update.OldVersion, update.NewVersion))
	}

	sb.WriteString("\n---\n")
	sb.WriteString("🤖 This PR was created automatically by the GitHub Actions workflow updater.")
	return sb.String()
}
