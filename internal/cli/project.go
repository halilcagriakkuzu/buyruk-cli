package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/config"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
	"github.com/spf13/cobra"
)

// NewProjectCmd creates and returns the project command.
func NewProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
		Long:  "Create and manage buyruk projects",
	}

	cmd.AddCommand(NewProjectCreateCmd())
	cmd.AddCommand(NewProjectRepairCmd())
	cmd.AddCommand(NewProjectDeleteCmd())

	return cmd
}

// NewProjectCreateCmd creates and returns the project create command.
func NewProjectCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <key>",
		Short: "Create a new project",
		Long:  "Create a new buyruk project with the specified key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectKey := args[0]
			return createProject(projectKey, cmd)
		},
	}

	cmd.Flags().String("name", "", "Project name (optional)")

	return cmd
}

// NewProjectRepairCmd creates and returns the project repair command.
func NewProjectRepairCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repair <key>",
		Short: "Repair project index",
		Long:  "Rebuild project.json index from issues directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectKey := args[0]
			return repairProject(projectKey, cmd)
		},
	}

	return cmd
}

// createProject creates a new project with the given key.
func createProject(projectKey string, cmd *cobra.Command) error {
	// Validate project key format
	if !isValidProjectKey(projectKey) {
		return fmt.Errorf("cli: invalid project key %q (must contain only uppercase letters, numbers, and hyphens)", projectKey)
	}

	// Get project name from flag or use key
	projectName, _ := cmd.Flags().GetString("name")
	if projectName == "" {
		projectName = projectKey
	}

	// Resolve paths
	projectDir, err := storage.ProjectDir(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve project directory: %w", err)
	}

	issuesDir, err := storage.IssuesDir(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve issues directory: %w", err)
	}

	epicsDir, err := storage.EpicsDir(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve epics directory: %w", err)
	}

	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve index path: %w", err)
	}

	// Create initial project index atomically (fails if project already exists)
	// This is the atomic check - if index file exists, project exists
	index := &models.ProjectIndex{
		ProjectKey:  projectKey,
		ProjectName: projectName,
		Issues:      []models.IndexEntry{},
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}

	if err := storage.WriteJSONAtomicCreate(indexPath, index); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("cli: project %q already exists", projectKey)
		}
		return fmt.Errorf("cli: failed to create project index: %w", err)
	}

	// Create project structure directories (idempotent, safe to call multiple times)
	// These are created after the atomic index creation to ensure project is registered first
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("cli: failed to create project directory: %w", err)
	}

	if err := os.MkdirAll(issuesDir, 0755); err != nil {
		return fmt.Errorf("cli: failed to create issues directory: %w", err)
	}

	if err := os.MkdirAll(epicsDir, 0755); err != nil {
		return fmt.Errorf("cli: failed to create epics directory: %w", err)
	}

	// Success message
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Created project %q\n", projectKey)

	return nil
}

// repairProject repairs a project index by rebuilding it from the issues directory.
func repairProject(projectKey string, cmd *cobra.Command) error {
	// Check if project exists
	projectDir, err := storage.ProjectDir(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve project directory: %w", err)
	}

	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return fmt.Errorf("cli: project %q does not exist", projectKey)
	}

	// Check for pending transaction
	pendingPath := filepath.Join(projectDir, ".buyruk_pending")
	if _, err := os.Stat(pendingPath); err == nil {
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "Warning: Found pending transaction. This may indicate a previous crash.\n")
	}

	// Read all issue files from issues directory
	issuesDir, err := storage.IssuesDir(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve issues directory: %w", err)
	}

	entries, err := os.ReadDir(issuesDir)
	if err != nil {
		return fmt.Errorf("cli: failed to read issues directory: %w", err)
	}

	// Rebuild index from issue files
	indexEntries := []models.IndexEntry{}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		issuePath := filepath.Join(issuesDir, entry.Name())
		var issue models.Issue

		if err := storage.ReadJSON(issuePath, &issue); err != nil {
			// Log error but continue
			errOut := cmd.ErrOrStderr()
			fmt.Fprintf(errOut, "Warning: failed to read issue file %s: %v\n", entry.Name(), err)
			continue
		}

		// Validate issue
		if err := issue.Validate(); err != nil {
			errOut := cmd.ErrOrStderr()
			fmt.Fprintf(errOut, "Warning: invalid issue in %s: %v\n", entry.Name(), err)
			continue
		}

		// Add to index
		indexEntries = append(indexEntries, models.IndexEntry{
			ID:     issue.ID,
			Title:  issue.Title,
			Status: issue.Status,
			Type:   issue.Type,
			EpicID: issue.EpicID,
		})
	}

	// Update index atomically (read-modify-write with locking)
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve index path: %w", err)
	}

	var index models.ProjectIndex
	if err := storage.UpdateJSONAtomic(indexPath, &index, func(v interface{}) error {
		idx := v.(*models.ProjectIndex)
		// If index doesn't exist, initialize it
		if idx.ProjectKey == "" {
			idx.ProjectKey = projectKey
			idx.Issues = []models.IndexEntry{}
		}
		// Update with rebuilt entries
		idx.Issues = indexEntries
		idx.UpdatedAt = time.Now().Format(time.RFC3339)
		return nil
	}); err != nil {
		return fmt.Errorf("cli: failed to write repaired index: %w", err)
	}

	// Success message
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Repaired project %q: %d issues indexed\n", projectKey, len(indexEntries))

	return nil
}

// isValidProjectKey validates that the project key is uppercase alphanumeric or hyphen.
func isValidProjectKey(key string) bool {
	if len(key) == 0 {
		return false
	}
	for _, r := range key {
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return true
}

// NewProjectDeleteCmd creates and returns the project delete command.
func NewProjectDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <key>",
		Short: "Delete a project",
		Long:  "Delete a project and all its data (issues, epics, etc.)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectKey := args[0]
			return deleteProject(projectKey, cmd)
		},
	}

	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt and override safety checks (force delete)")

	return cmd
}

// deleteProject deletes a project and all its data.
func deleteProject(projectKey string, cmd *cobra.Command) error {
	// Validate project key format
	if !isValidProjectKey(projectKey) {
		return fmt.Errorf("cli: invalid project key %q (must contain only uppercase letters, numbers, and hyphens)", projectKey)
	}

	// Resolve project directory
	projectDir, err := storage.ProjectDir(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve project directory: %w", err)
	}

	// Check if project exists
	if _, err := os.Stat(projectDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cli: project %q does not exist", projectKey)
		}
		return fmt.Errorf("cli: failed to access project directory %q: %w", projectDir, err)
	}

	// Check for pending transaction before acquiring lock
	hasPending, _, err := storage.CheckPendingTransaction(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to check pending transaction: %w", err)
	}
	if hasPending {
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			return fmt.Errorf("cli: project %q has a pending transaction (may indicate a crash). Use -y to force deletion", projectKey)
		}
		errOut := cmd.ErrOrStderr()
		fmt.Fprintf(errOut, "Warning: project %q has a pending transaction. Proceeding with deletion anyway.\n", projectKey)
	}

	// Acquire project lock to prevent concurrent modifications during deletion
	// This ensures no other operations are running while we delete
	cleanup, err := storage.AcquireLock(projectKey)
	if err != nil {
		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			return fmt.Errorf("cli: failed to acquire project lock for %q (another operation may be in progress). Use -y to force deletion: %w", projectKey, err)
		}
		errOut := cmd.ErrOrStderr()
		fmt.Fprintf(errOut, "Warning: failed to acquire lock for project %q. Proceeding with deletion anyway.\n", projectKey)
	} else {
		defer cleanup()
	}

	// Count issues and epics for warning
	issueCount := 0
	epicCount := 0

	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err == nil {
		var index models.ProjectIndex
		if err := storage.ReadJSON(indexPath, &index); err == nil {
			issueCount = len(index.Issues)
		}
	}

	epicsDir, err := storage.EpicsDir(projectKey)
	if err == nil {
		if entries, err := os.ReadDir(epicsDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
					epicCount++
				}
			}
		}
	}

	// Confirmation prompt (unless -y flag is set)
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		errOut := cmd.ErrOrStderr()
		fmt.Fprintf(errOut, "Warning: This will delete project %q and all its data (%d issues, %d epics).\n", projectKey, issueCount, epicCount)
		fmt.Fprintf(errOut, "Are you sure you want to delete project %q? (yes/no): ", projectKey)

		scanner := bufio.NewScanner(cmd.InOrStdin())
		if !scanner.Scan() {
			return fmt.Errorf("cli: failed to read confirmation: %w", scanner.Err())
		}
		response := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if response != "yes" && response != "y" {
			return fmt.Errorf("cli: deletion cancelled")
		}
	}

	// Begin transaction for project deletion
	if err := storage.BeginTransaction(projectKey, "delete_project", map[string]interface{}{
		"project_key": projectKey,
	}); err != nil {
		return fmt.Errorf("cli: failed to begin deletion transaction: %w", err)
	}

	success := false
	defer func() {
		if !success {
			storage.RollbackTransaction(projectKey)
		}
	}()

	// Delete project directory (removes all files including issues, epics, index, etc.)
	// Note: We're deleting the entire directory, so the lock file and transaction log
	// will also be removed. This is safe because we hold the lock.
	if err := os.RemoveAll(projectDir); err != nil {
		return fmt.Errorf("cli: failed to delete project directory: %w", err)
	}

	// Commit transaction (though the directory is already deleted, this cleans up
	// any remaining transaction state if the deletion was partial)
	// Since the directory is gone, this may fail, but that's okay
	_ = storage.CommitTransaction(projectKey)
	success = true

	// Success message
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Deleted project %q\n", projectKey)

	return nil
}

// ResolveProjectKey resolves the project key from the command.
// This is a convenience wrapper around config.ResolveProject.
func ResolveProjectKey(cmd *cobra.Command) (string, error) {
	return config.ResolveProject(cmd)
}
