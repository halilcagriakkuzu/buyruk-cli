package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/config"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
	"github.com/spf13/cobra"
)

// NewIssueCmd creates and returns the issue command.
func NewIssueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Manage issues",
		Long:  "Create and manage buyruk issues",
	}

	cmd.AddCommand(NewIssueCreateCmd())
	cmd.AddCommand(NewIssueUpdateCmd())
	cmd.AddCommand(NewIssueLinkCmd())
	cmd.AddCommand(NewIssuePRCmd())
	cmd.AddCommand(NewIssueDeleteCmd())

	return cmd
}

// NewIssueCreateCmd creates and returns the issue create command.
func NewIssueCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		Long:  "Create a new issue in the project. Only title is required.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return createIssue(cmd)
		},
	}

	cmd.Flags().String("id", "", "Issue ID (optional, auto-generated if not provided)")
	cmd.Flags().String("type", "task", "Issue type (task or bug, default: task)")
	cmd.Flags().String("title", "", "Issue title (required)")
	cmd.Flags().String("status", "TODO", "Issue status (TODO, DOING, DONE, default: TODO)")
	cmd.Flags().String("priority", "", "Issue priority (LOW, MEDIUM, HIGH, CRITICAL)")
	cmd.Flags().String("description", "", "Issue description (Markdown)")
	cmd.Flags().String("epic", "", "Link to epic ID")

	return cmd
}

// createIssue creates a new issue in the project.
func createIssue(cmd *cobra.Command) error {
	// Resolve project (required for auto-increment)
	projectKey, err := config.ResolveProject(cmd)
	if err != nil {
		return err
	}

	// Verify project exists
	projectDir, err := storage.ProjectDir(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve project directory: %w", err)
	}

	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return fmt.Errorf("cli: project %q does not exist", projectKey)
	}

	// Get title (required)
	title, _ := cmd.Flags().GetString("title")
	if title == "" {
		return fmt.Errorf("cli: title is required")
	}

	// Get ID (optional, auto-generate if not provided)
	issueID, _ := cmd.Flags().GetString("id")
	if issueID == "" {
		nextSeq, err := getNextIssueSequence(projectKey)
		if err != nil {
			return fmt.Errorf("cli: failed to get next issue sequence: %w", err)
		}
		issueID = models.GenerateIssueID(projectKey, nextSeq)
	} else {
		// Validate provided ID matches project key
		parsedKey, _, err := models.ParseIssueID(issueID)
		if err != nil {
			return fmt.Errorf("cli: invalid issue ID format: %w", err)
		}
		if parsedKey != projectKey {
			return fmt.Errorf("cli: issue ID %q does not match project key %q", issueID, projectKey)
		}
	}

	// Get type (default: task)
	issueType, _ := cmd.Flags().GetString("type")
	if issueType == "" {
		issueType = models.TypeTask
	}

	// Get status (default: TODO)
	status, _ := cmd.Flags().GetString("status")
	if status == "" {
		status = models.StatusTODO
	}

	// Get optional fields
	priority, _ := cmd.Flags().GetString("priority")
	description, _ := cmd.Flags().GetString("description")
	epicID, _ := cmd.Flags().GetString("epic")

	// Validate epic ID format if provided
	if epicID != "" {
		if err := validateEpicID(epicID); err != nil {
			return fmt.Errorf("cli: invalid epic ID format: %w", err)
		}
		// Validate epic exists
		epicPath, err := storage.EpicPath(projectKey, epicID)
		if err != nil {
			return fmt.Errorf("cli: failed to resolve epic path: %w", err)
		}
		if _, err := os.Stat(epicPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("cli: epic %q not found", epicID)
			}
			return fmt.Errorf("cli: failed to stat epic path %q: %w", epicPath, err)
		}
	}

	// Create issue
	issue := &models.Issue{
		ID:          issueID,
		Type:        issueType,
		Title:       title,
		Status:      status,
		Priority:    priority,
		Description: description,
		EpicID:      epicID,
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}

	// Validate issue
	if err := issue.Validate(); err != nil {
		return fmt.Errorf("cli: invalid issue: %w", err)
	}

	// Write issue file atomically (fails if file already exists)
	issuePath, err := storage.IssuePath(projectKey, issueID)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve issue path: %w", err)
	}

	if err := storage.WriteJSONAtomicCreate(issuePath, issue); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("cli: issue %q already exists", issueID)
		}
		return fmt.Errorf("cli: failed to create issue file: %w", err)
	}

	// Update project index atomically (read-modify-write with locking)
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve index path: %w", err)
	}

	var index models.ProjectIndex
	if err := storage.UpdateJSONAtomic(indexPath, &index, func(v interface{}) error {
		idx := v.(*models.ProjectIndex)
		idx.AddIssue(issue)
		idx.UpdatedAt = time.Now().Format(time.RFC3339)
		return nil
	}); err != nil {
		return fmt.Errorf("cli: failed to update project index: %w", err)
	}

	// Success message
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Created issue %q\n", issueID)

	return nil
}

// getNextIssueSequence returns the next sequence number for an issue in the project.
// It parses all existing issue IDs to find the highest sequence number and returns the next one.
func getNextIssueSequence(projectKey string) (int, error) {
	// Load project index
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		return 0, fmt.Errorf("cli: failed to resolve index path: %w", err)
	}

	var index models.ProjectIndex
	if err := storage.ReadJSON(indexPath, &index); err != nil {
		// If index doesn't exist, start from 1
		if os.IsNotExist(err) {
			return 1, nil
		}
		return 0, fmt.Errorf("cli: failed to load project index: %w", err)
	}

	// Find the highest sequence number
	maxSeq := 0
	for _, entry := range index.Issues {
		_, seq, err := models.ParseIssueID(entry.ID)
		if err != nil {
			// Skip invalid IDs
			continue
		}
		if seq > maxSeq {
			maxSeq = seq
		}
	}

	// Return next sequence number
	return maxSeq + 1, nil
}

// NewIssueUpdateCmd creates and returns the issue update command.
func NewIssueUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an issue",
		Long:  "Update fields of an existing issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			return updateIssue(issueID, cmd)
		},
	}

	cmd.Flags().String("title", "", "Update title")
	cmd.Flags().String("type", "", "Update type")
	cmd.Flags().String("status", "", "Update status")
	cmd.Flags().String("priority", "", "Update priority")
	cmd.Flags().String("description", "", "Update description")
	cmd.Flags().String("epic", "", "Update epic link")

	return cmd
}

// updateIssue updates an existing issue.
func updateIssue(issueID string, cmd *cobra.Command) error {
	// Parse issue ID
	projectKey, _, err := models.ParseIssueID(issueID)
	if err != nil {
		return fmt.Errorf("cli: invalid issue ID %q: %w", issueID, err)
	}

	// Load issue atomically (read-modify-write)
	issuePath, err := storage.IssuePath(projectKey, issueID)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve issue path: %w", err)
	}

	var issue models.Issue
	if err := storage.UpdateJSONAtomic(issuePath, &issue, func(v interface{}) error {
		iss := v.(*models.Issue)

		// Check if issue exists (ID should match if file existed)
		if iss.ID == "" || iss.ID != issueID {
			return fmt.Errorf("cli: issue %q not found", issueID)
		}

		// Update fields from flags
		if title, _ := cmd.Flags().GetString("title"); title != "" {
			iss.Title = title
		}

		if issueType, _ := cmd.Flags().GetString("type"); issueType != "" {
			if !models.IsValidType(issueType) {
				return fmt.Errorf("cli: invalid type %q", issueType)
			}
			iss.Type = issueType
		}

		if status, _ := cmd.Flags().GetString("status"); status != "" {
			if !models.IsValidStatus(status) {
				return fmt.Errorf("cli: invalid status %q", status)
			}
			iss.Status = status
		}

		if priority, _ := cmd.Flags().GetString("priority"); priority != "" {
			if !models.IsValidPriority(priority) {
				return fmt.Errorf("cli: invalid priority %q", priority)
			}
			iss.Priority = priority
		}

		if description, _ := cmd.Flags().GetString("description"); description != "" {
			iss.Description = description
		}

		if epicID, _ := cmd.Flags().GetString("epic"); epicID != "" {
			// Validate epic ID format
			if err := validateEpicID(epicID); err != nil {
				return fmt.Errorf("cli: invalid epic ID format: %w", err)
			}
			// Validate epic exists before setting
			epicPath, err := storage.EpicPath(projectKey, epicID)
			if err != nil {
				return fmt.Errorf("cli: failed to resolve epic path: %w", err)
			}
			if _, err := os.Stat(epicPath); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("cli: epic %q not found", epicID)
				}
				return fmt.Errorf("cli: failed to stat epic path %q: %w", epicPath, err)
			}
			iss.EpicID = epicID
		}

		// Update timestamp
		iss.UpdatedAt = time.Now().Format(time.RFC3339)

		// Validate
		if err := iss.Validate(); err != nil {
			return fmt.Errorf("cli: invalid issue after update: %w", err)
		}

		return nil
	}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("cli: issue %q not found", issueID)
		}
		return fmt.Errorf("cli: failed to update issue: %w", err)
	}

	// Update project index atomically
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve index path: %w", err)
	}

	if err := storage.UpdateJSONAtomic(indexPath, &models.ProjectIndex{}, func(v interface{}) error {
		idx := v.(*models.ProjectIndex)
		idx.AddIssue(&issue)
		idx.UpdatedAt = time.Now().Format(time.RFC3339)
		return nil
	}); err != nil {
		return fmt.Errorf("cli: failed to update project index: %w", err)
	}

	// Success message
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Updated %s\n", issueID)

	return nil
}

// NewIssueLinkCmd creates and returns the issue link command.
func NewIssueLinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link <id> <dependency-id>",
		Short: "Link issues with dependencies",
		Long:  "Add a dependency relationship (issue is blocked by dependency)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			dependencyID := args[1]
			return linkIssue(issueID, dependencyID, cmd)
		},
	}

	cmd.Flags().Bool("remove", false, "Remove dependency instead of adding")

	return cmd
}

// linkIssue links an issue with a dependency.
func linkIssue(issueID, dependencyID string, cmd *cobra.Command) error {
	// Parse issue IDs
	projectKey, _, err := models.ParseIssueID(issueID)
	if err != nil {
		return fmt.Errorf("cli: invalid issue ID %q: %w", issueID, err)
	}

	depProjectKey, _, err := models.ParseIssueID(dependencyID)
	if err != nil {
		return fmt.Errorf("cli: invalid dependency ID %q: %w", dependencyID, err)
	}

	// Validate dependency exists
	depPath, err := storage.IssuePath(depProjectKey, dependencyID)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve dependency path: %w", err)
	}

	if _, err := os.Stat(depPath); os.IsNotExist(err) {
		return fmt.Errorf("cli: dependency %q not found", dependencyID)
	}

	// Load and update issue atomically
	issuePath, err := storage.IssuePath(projectKey, issueID)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve issue path: %w", err)
	}

	var issue models.Issue
	remove, _ := cmd.Flags().GetBool("remove")

	if err := storage.UpdateJSONAtomic(issuePath, &issue, func(v interface{}) error {
		iss := v.(*models.Issue)

		// Check if issue exists (ID should match if file existed)
		if iss.ID == "" || iss.ID != issueID {
			return fmt.Errorf("cli: issue %q not found", issueID)
		}

		// Add or remove dependency
		if remove {
			iss.RemoveDependency(dependencyID)
		} else {
			iss.AddDependency(dependencyID)
		}

		// Update timestamp
		iss.UpdatedAt = time.Now().Format(time.RFC3339)

		return nil
	}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("cli: issue %q not found", issueID)
		}
		return fmt.Errorf("cli: failed to update issue: %w", err)
	}

	// Success message
	out := cmd.OutOrStdout()
	if remove {
		fmt.Fprintf(out, "Removed dependency %s from %s\n", dependencyID, issueID)
	} else {
		fmt.Fprintf(out, "Linked %s -> %s (blocked by)\n", issueID, dependencyID)
	}

	return nil
}

// NewIssuePRCmd creates and returns the issue PR command.
func NewIssuePRCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr <id> <pr-url>",
		Short: "Add or remove PR links",
		Long:  "Add or remove pull request URLs from an issue",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			prURL := args[1]
			return manageIssuePR(issueID, prURL, cmd)
		},
	}

	cmd.Flags().Bool("remove", false, "Remove PR instead of adding")

	return cmd
}

// manageIssuePR adds or removes a PR URL from an issue.
func manageIssuePR(issueID, prURL string, cmd *cobra.Command) error {
	// Parse issue ID
	projectKey, _, err := models.ParseIssueID(issueID)
	if err != nil {
		return fmt.Errorf("cli: invalid issue ID %q: %w", issueID, err)
	}

	// Load and update issue atomically
	issuePath, err := storage.IssuePath(projectKey, issueID)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve issue path: %w", err)
	}

	var issue models.Issue
	remove, _ := cmd.Flags().GetBool("remove")

	if err := storage.UpdateJSONAtomic(issuePath, &issue, func(v interface{}) error {
		iss := v.(*models.Issue)

		// Check if issue exists (ID should match if file existed)
		if iss.ID == "" || iss.ID != issueID {
			return fmt.Errorf("cli: issue %q not found", issueID)
		}

		// Add or remove PR
		if remove {
			iss.RemovePR(prURL)
		} else {
			iss.AddPR(prURL)
		}

		// Update timestamp
		iss.UpdatedAt = time.Now().Format(time.RFC3339)

		return nil
	}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("cli: issue %q not found", issueID)
		}
		return fmt.Errorf("cli: failed to update issue: %w", err)
	}

	// Success message
	out := cmd.OutOrStdout()
	if remove {
		fmt.Fprintf(out, "Removed PR %s from %s\n", prURL, issueID)
	} else {
		fmt.Fprintf(out, "Added PR %s to %s\n", prURL, issueID)
	}

	return nil
}

// NewIssueDeleteCmd creates and returns the issue delete command.
func NewIssueDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an issue",
		Long:  "Delete an issue from the project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			return deleteIssue(issueID, cmd)
		},
	}

	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt and override safety checks (force delete)")

	return cmd
}

// deleteIssue deletes an issue from the project.
func deleteIssue(issueID string, cmd *cobra.Command) error {
	// Parse issue ID
	projectKey, _, err := models.ParseIssueID(issueID)
	if err != nil {
		return fmt.Errorf("cli: invalid issue ID %q: %w", issueID, err)
	}

	// Check if issue exists
	issuePath, err := storage.IssuePath(projectKey, issueID)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve issue path: %w", err)
	}

	if _, err := os.Stat(issuePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cli: issue %q not found", issueID)
		}
		return fmt.Errorf("cli: failed to stat issue path %q: %w", issuePath, err)
	}

	// Check for issues that depend on this issue (pre-lock read for warnings only)
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve index path: %w", err)
	}

	var preLockIndex models.ProjectIndex
	if err := storage.ReadJSON(indexPath, &preLockIndex); err == nil {
		// Check if any issues depend on this issue
		dependentIssues := []string{}
		for _, entry := range preLockIndex.Issues {
			// Load issue to check dependencies
			depIssuePath, err := storage.IssuePath(projectKey, entry.ID)
			if err != nil {
				continue
			}
			var depIssue models.Issue
			if err := storage.ReadJSON(depIssuePath, &depIssue); err != nil {
				continue
			}
			for _, blockedBy := range depIssue.BlockedBy {
				if blockedBy == issueID {
					dependentIssues = append(dependentIssues, entry.ID)
					break
				}
			}
		}
		if len(dependentIssues) > 0 {
			errOut := cmd.ErrOrStderr()
			fmt.Fprintf(errOut, "Warning: %d issue(s) depend on this issue: %s\n", len(dependentIssues), strings.Join(dependentIssues, ", "))
		}
	}

	// Confirmation prompt (unless -y flag is set)
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		errOut := cmd.ErrOrStderr()
		fmt.Fprintf(errOut, "Are you sure you want to delete issue %q? (yes/no): ", issueID)

		scanner := bufio.NewScanner(cmd.InOrStdin())
		if !scanner.Scan() {
			return fmt.Errorf("cli: failed to read confirmation: %w", scanner.Err())
		}
		response := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if response != "yes" && response != "y" {
			return fmt.Errorf("cli: deletion cancelled")
		}
	}

	// Delete issue file and update index atomically under one lock/transaction
	// This prevents race conditions where the file is deleted but index update fails
	cleanup, err := storage.AcquireLock(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to acquire lock: %w", err)
	}
	defer cleanup()

	// Begin transaction
	if err := storage.BeginTransaction(projectKey, "delete_issue", map[string]interface{}{
		"issue_id": issueID,
		"file":     issuePath,
	}); err != nil {
		return fmt.Errorf("cli: failed to begin transaction: %w", err)
	}

	success := false
	defer func() {
		if !success {
			storage.RollbackTransaction(projectKey)
		}
	}()

	// Delete issue file
	if err := os.Remove(issuePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cli: issue %q not found", issueID)
		}
		return fmt.Errorf("cli: failed to delete issue file: %w", err)
	}

	// Update project index (remove issue from index)
	// Use a fresh index variable to avoid stale data from pre-lock read
	var index models.ProjectIndex
	if err := storage.ReadJSON(indexPath, &index); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cli: failed to read project index: %w", err)
		}
		// Index doesn't exist, initialize empty index
		index = models.ProjectIndex{
			ProjectKey: projectKey,
			Issues:     []models.IndexEntry{},
		}
	}
	index.RemoveIssue(issueID)
	index.UpdatedAt = time.Now().Format(time.RFC3339)

	// Write updated index
	data, err := json.MarshalIndent(&index, "", "  ")
	if err != nil {
		return fmt.Errorf("cli: failed to marshal index: %w", err)
	}
	if err := storage.WriteAtomic(indexPath, data); err != nil {
		return fmt.Errorf("cli: failed to write index: %w", err)
	}

	// Commit transaction
	if err := storage.CommitTransaction(projectKey); err != nil {
		return fmt.Errorf("cli: failed to commit transaction: %w", err)
	}

	success = true

	// Success message
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Deleted issue %q\n", issueID)

	return nil
}

// validateEpicID validates the format of an epic ID.
// Epic IDs should be non-empty and contain only uppercase alphanumeric characters and hyphens.
// Enforcing uppercase prevents collisions on case-insensitive filesystems (Windows/macOS).
func validateEpicID(epicID string) error {
	if epicID == "" {
		return fmt.Errorf("epic ID cannot be empty")
	}

	// Check for invalid characters (only allow uppercase alphanumeric and hyphens)
	// Enforce uppercase to prevent collisions on case-insensitive filesystems
	for _, r := range epicID {
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
			return fmt.Errorf("epic ID contains invalid character %q (only uppercase alphanumeric and hyphens allowed)", r)
		}
	}

	return nil
}
