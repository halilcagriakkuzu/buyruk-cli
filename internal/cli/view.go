package cli

import (
	"fmt"
	"os"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/ui"
	"github.com/spf13/cobra"
)

// NewViewCmd creates and returns the view command.
func NewViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "View issue details",
		Long:  "View detailed information about an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			return viewIssue(issueID, cmd)
		},
	}

	return cmd
}

// viewIssue views a single issue by ID.
func viewIssue(issueID string, cmd *cobra.Command) error {
	// Parse issue ID to get project key
	projectKey, _, err := models.ParseIssueID(issueID)
	if err != nil {
		return fmt.Errorf("cli: invalid issue ID %q: %w", issueID, err)
	}

	// Load issue
	issuePath, err := storage.IssuePath(projectKey, issueID)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve issue path: %w", err)
	}

	var issue models.Issue
	if err := storage.ReadJSON(issuePath, &issue); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cli: issue %q not found", issueID)
		}
		return fmt.Errorf("cli: failed to load issue: %w", err)
	}

	// Render using UI layer
	renderer, err := ui.GetRenderer(cmd)
	if err != nil {
		return fmt.Errorf("cli: failed to get renderer: %w", err)
	}

	out := cmd.OutOrStdout()
	if err := renderer.RenderIssue(&issue, out); err != nil {
		return fmt.Errorf("cli: failed to render issue: %w", err)
	}

	return nil
}
