package cli

import (
	"fmt"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/config"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/ui"
	"github.com/spf13/cobra"
)

// NewListCmd creates and returns the list command.
func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List project issues",
		Long:  "List all issues in a project using the project index",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listIssues(cmd)
		},
	}

	return cmd
}

// listIssues lists all issues in the current project.
func listIssues(cmd *cobra.Command) error {
	// Resolve project
	projectKey, err := config.ResolveProject(cmd)
	if err != nil {
		return err
	}

	// Load project index
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve index path: %w", err)
	}

	var index models.ProjectIndex
	if err := storage.ReadJSON(indexPath, &index); err != nil {
		return fmt.Errorf("cli: failed to load project index: %w", err)
	}

	// Convert index entries to issues (load full issue data)
	issues := []*models.Issue{}

	for _, entry := range index.Issues {
		issuePath, err := storage.IssuePath(projectKey, entry.ID)
		if err != nil {
			return fmt.Errorf("cli: failed to resolve issue path: %w", err)
		}

		var issue models.Issue
		if err := storage.ReadJSON(issuePath, &issue); err != nil {
			// Log warning but continue
			errOut := cmd.ErrOrStderr()
			fmt.Fprintf(errOut, "Warning: failed to load issue %s: %v\n", entry.ID, err)
			continue
		}

		issues = append(issues, &issue)
	}

	// Render using UI layer
	renderer, err := ui.GetRenderer(cmd)
	if err != nil {
		return fmt.Errorf("cli: failed to get renderer: %w", err)
	}

	out := cmd.OutOrStdout()
	if err := renderer.RenderIssueList(issues, out); err != nil {
		return fmt.Errorf("cli: failed to render issue list: %w", err)
	}

	return nil
}
