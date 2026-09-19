package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/config"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/ui"
	"github.com/spf13/cobra"
)

// NewEpicCmd creates and returns the epic command.
func NewEpicCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "epic",
		Short: "Manage epics",
		Long:  "Create and manage buyruk epics",
	}

	cmd.AddCommand(NewEpicCreateCmd())
	cmd.AddCommand(NewEpicViewCmd())
	cmd.AddCommand(NewEpicUpdateCmd())
	cmd.AddCommand(NewEpicListCmd())
	cmd.AddCommand(NewEpicDeleteCmd())

	return cmd
}

// NewEpicCreateCmd creates and returns the epic create command.
func NewEpicCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new epic",
		Long:  "Create a new epic in the project. Title is required.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return createEpic(cmd)
		},
	}

	cmd.Flags().String("id", "", "Epic ID (optional, auto-generated if not provided)")
	cmd.Flags().String("title", "", "Epic title (required)")
	cmd.Flags().String("status", "TODO", "Epic status (TODO, DOING, DONE, default: TODO)")
	cmd.Flags().String("description", "", "Epic description (Markdown)")

	return cmd
}

// createEpic creates a new epic in the project.
func createEpic(cmd *cobra.Command) error {
	// Resolve project
	projectKey, err := config.ResolveProject(cmd)
	if err != nil {
		return err
	}

	// Verify project exists
	projectDir, err := storage.ProjectDir(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve project directory: %w", err)
	}

	if _, err := os.Stat(projectDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cli: project %q does not exist", projectKey)
		}
		return fmt.Errorf("cli: failed to access project directory: %w", err)
	}

	// Get title (required)
	title, _ := cmd.Flags().GetString("title")
	if title == "" {
		return fmt.Errorf("cli: title is required")
	}

	// Get ID (optional, auto-generate if not provided)
	epicID, _ := cmd.Flags().GetString("id")
	if epicID == "" {
		nextSeq, err := getNextEpicSequence(projectKey)
		if err != nil {
			return fmt.Errorf("cli: failed to get next epic sequence: %w", err)
		}
		epicID = fmt.Sprintf("E-%d", nextSeq)
	} else {
		// Validate provided ID format
		if err := validateEpicID(epicID); err != nil {
			return fmt.Errorf("cli: invalid epic ID format: %w", err)
		}
	}

	// Get status (default: TODO)
	status, _ := cmd.Flags().GetString("status")
	if status == "" {
		status = models.StatusTODO
	}
	if !models.IsValidStatus(status) {
		return fmt.Errorf("cli: invalid status %q", status)
	}

	// Get optional fields
	description, _ := cmd.Flags().GetString("description")

	// Create epic
	epic := &models.Epic{
		ID:          epicID,
		Title:       title,
		Status:      status,
		Description: description,
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}

	// Validate epic
	if err := epic.Validate(); err != nil {
		return fmt.Errorf("cli: invalid epic: %w", err)
	}

	// Write epic file atomically (fails if file already exists)
	epicPath, err := storage.EpicPath(projectKey, epicID)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve epic path: %w", err)
	}

	if err := storage.WriteJSONAtomicCreate(epicPath, epic); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("cli: epic %q already exists", epicID)
		}
		return fmt.Errorf("cli: failed to create epic file: %w", err)
	}

	// Success message
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Created epic %q\n", epicID)

	return nil
}

// getNextEpicSequence returns the next sequence number for an epic in the project.
// It scans the epics directory to find the highest sequence number and returns the next one.
func getNextEpicSequence(projectKey string) (int, error) {
	epicsDir, err := storage.EpicsDir(projectKey)
	if err != nil {
		return 0, fmt.Errorf("cli: failed to resolve epics directory: %w", err)
	}

	// If epics directory doesn't exist, start from 1
	if _, err := os.Stat(epicsDir); os.IsNotExist(err) {
		return 1, nil
	}

	entries, err := os.ReadDir(epicsDir)
	if err != nil {
		return 0, fmt.Errorf("cli: failed to read epics directory: %w", err)
	}

	// Find the highest sequence number
	maxSeq := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Extract epic ID from filename (remove .json extension)
		epicID := strings.TrimSuffix(entry.Name(), ".json")

		// Parse sequence from epic ID (format: E-1, E-2, etc.)
		// Only consider IDs matching the standard "E-<n>" pattern for auto-increment
		// This prevents unrelated epic IDs (e.g., "CUSTOM-99") from affecting the sequence
		if strings.HasPrefix(epicID, "E-") {
			// Extract the sequence number after "E-"
			seqStr := strings.TrimPrefix(epicID, "E-")
			var seq int
			if _, err := fmt.Sscanf(seqStr, "%d", &seq); err == nil {
				if seq > maxSeq {
					maxSeq = seq
				}
			}
		}
	}

	// Return next sequence number
	return maxSeq + 1, nil
}

// NewEpicViewCmd creates and returns the epic view command.
func NewEpicViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "View epic details",
		Long:  "View detailed information about an epic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			epicID := args[0]
			return viewEpic(epicID, cmd)
		},
	}

	return cmd
}

// viewEpic views a single epic by ID.
func viewEpic(epicID string, cmd *cobra.Command) error {
	// Validate epic ID format
	if err := validateEpicID(epicID); err != nil {
		return fmt.Errorf("cli: invalid epic ID format: %w", err)
	}

	// Resolve project
	projectKey, err := config.ResolveProject(cmd)
	if err != nil {
		return err
	}

	// Load epic
	epicPath, err := storage.EpicPath(projectKey, epicID)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve epic path: %w", err)
	}

	var epic models.Epic
	if err := storage.ReadJSON(epicPath, &epic); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cli: epic %q not found", epicID)
		}
		return fmt.Errorf("cli: failed to load epic: %w", err)
	}

	// Render using UI layer
	renderer, err := ui.GetRenderer(cmd)
	if err != nil {
		return fmt.Errorf("cli: failed to get renderer: %w", err)
	}

	out := cmd.OutOrStdout()
	if err := renderer.RenderEpic(&epic, out); err != nil {
		return fmt.Errorf("cli: failed to render epic: %w", err)
	}

	return nil
}

// NewEpicUpdateCmd creates and returns the epic update command.
func NewEpicUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an epic",
		Long:  "Update fields of an existing epic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			epicID := args[0]
			return updateEpic(epicID, cmd)
		},
	}

	cmd.Flags().String("title", "", "Update title")
	cmd.Flags().String("status", "", "Update status")
	cmd.Flags().String("description", "", "Update description")

	return cmd
}

// updateEpic updates an existing epic.
func updateEpic(epicID string, cmd *cobra.Command) error {
	// Validate epic ID format
	if err := validateEpicID(epicID); err != nil {
		return fmt.Errorf("cli: invalid epic ID format: %w", err)
	}

	// Resolve project
	projectKey, err := config.ResolveProject(cmd)
	if err != nil {
		return err
	}

	// Load and update epic atomically
	epicPath, err := storage.EpicPath(projectKey, epicID)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve epic path: %w", err)
	}

	var epic models.Epic
	if err := storage.UpdateJSONAtomic(epicPath, &epic, func(v interface{}) error {
		ep := v.(*models.Epic)

		// Check if epic exists (ID should match if file existed)
		if ep.ID == "" || ep.ID != epicID {
			return fmt.Errorf("cli: epic %q not found", epicID)
		}

		// Update fields from flags
		if title, _ := cmd.Flags().GetString("title"); title != "" {
			ep.Title = title
		}

		if status, _ := cmd.Flags().GetString("status"); status != "" {
			if !models.IsValidStatus(status) {
				return fmt.Errorf("cli: invalid status %q", status)
			}
			ep.Status = status
		}

		if description, _ := cmd.Flags().GetString("description"); description != "" {
			ep.Description = description
		}

		// Update timestamp
		ep.UpdatedAt = time.Now().Format(time.RFC3339)

		// Validate
		if err := ep.Validate(); err != nil {
			return fmt.Errorf("cli: invalid epic after update: %w", err)
		}

		return nil
	}); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("cli: epic %q not found", epicID)
		}
		return fmt.Errorf("cli: failed to update epic: %w", err)
	}

	// Success message
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Updated %s\n", epicID)

	return nil
}

// NewEpicListCmd creates and returns the epic list command.
func NewEpicListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List project epics",
		Long:  "List all epics in a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listEpics(cmd)
		},
	}

	return cmd
}

// listEpics lists all epics in the current project.
func listEpics(cmd *cobra.Command) error {
	// Resolve project
	projectKey, err := config.ResolveProject(cmd)
	if err != nil {
		return err
	}

	// Read all epic files from epics directory
	epicsDir, err := storage.EpicsDir(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve epics directory: %w", err)
	}

	// Check if epics directory exists
	if _, err := os.Stat(epicsDir); os.IsNotExist(err) {
		// No epics directory means no epics
		epics := []*models.Epic{}
		renderer, err := ui.GetRenderer(cmd)
		if err != nil {
			return fmt.Errorf("cli: failed to get renderer: %w", err)
		}
		out := cmd.OutOrStdout()
		// Render empty list
		return renderEpicList(epics, renderer, cmd, out)
	}

	entries, err := os.ReadDir(epicsDir)
	if err != nil {
		return fmt.Errorf("cli: failed to read epics directory: %w", err)
	}

	// Load all epics
	epics := []*models.Epic{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		epicPath := filepath.Join(epicsDir, entry.Name())
		var epic models.Epic
		if err := storage.ReadJSON(epicPath, &epic); err != nil {
			// Log warning but continue
			errOut := cmd.ErrOrStderr()
			fmt.Fprintf(errOut, "Warning: failed to load epic %s: %v\n", entry.Name(), err)
			continue
		}

		epics = append(epics, &epic)
	}

	// Render using UI layer
	renderer, err := ui.GetRenderer(cmd)
	if err != nil {
		return fmt.Errorf("cli: failed to get renderer: %w", err)
	}

	out := cmd.OutOrStdout()
	return renderEpicList(epics, renderer, cmd, out)
}

// renderEpicList renders a list of epics using the appropriate renderer.
func renderEpicList(epics []*models.Epic, renderer ui.Renderer, cmd *cobra.Command, w interface{ Write([]byte) (int, error) }) error {
	// For JSON format, render as an array
	format := config.ResolveFormat(cmd)
	if format == config.DefaultFormatJSON {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(epics)
	}

	// For modern/LSON, render each epic individually
	for i, epic := range epics {
		if i > 0 {
			// Add separator for multiple epics
			fmt.Fprintf(w, "\n")
		}
		if err := renderer.RenderEpic(epic, w); err != nil {
			return err
		}
	}
	return nil
}

// NewEpicDeleteCmd creates and returns the epic delete command.
func NewEpicDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an epic",
		Long:  "Delete an epic from the project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			epicID := args[0]
			return deleteEpic(epicID, cmd)
		},
	}

	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt and override safety checks (force delete)")

	return cmd
}

// deleteEpic deletes an epic from the project.
func deleteEpic(epicID string, cmd *cobra.Command) error {
	// Validate epic ID format
	if err := validateEpicID(epicID); err != nil {
		return fmt.Errorf("cli: invalid epic ID format: %w", err)
	}

	// Resolve project
	projectKey, err := config.ResolveProject(cmd)
	if err != nil {
		return err
	}

	// Check if epic exists
	epicPath, err := storage.EpicPath(projectKey, epicID)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve epic path: %w", err)
	}

	if _, err := os.Stat(epicPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("cli: epic %q not found", epicID)
		}
		return fmt.Errorf("cli: failed to stat epic: %w", err)
	}

	// Check for issues that reference this epic
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve index path: %w", err)
	}

	var index models.ProjectIndex
	if err := storage.ReadJSON(indexPath, &index); err == nil {
		// Check if any issues reference this epic
		referencedIssues := []string{}
		for _, entry := range index.Issues {
			if entry.EpicID == epicID {
				referencedIssues = append(referencedIssues, entry.ID)
			}
		}
		if len(referencedIssues) > 0 {
			errOut := cmd.ErrOrStderr()
			fmt.Fprintf(errOut, "Warning: %d issue(s) reference this epic: %s\n", len(referencedIssues), strings.Join(referencedIssues, ", "))
		}
	}

	// Confirmation prompt (unless -y flag is set)
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		errOut := cmd.ErrOrStderr()
		fmt.Fprintf(errOut, "Are you sure you want to delete epic %q? (yes/no): ", epicID)

		scanner := bufio.NewScanner(cmd.InOrStdin())
		if !scanner.Scan() {
			return fmt.Errorf("cli: failed to read confirmation: %w", scanner.Err())
		}
		response := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if response != "yes" && response != "y" {
			return fmt.Errorf("cli: deletion cancelled")
		}
	}

	// Delete epic file atomically (with lock and transaction)
	if err := storage.DeleteAtomic(epicPath); err != nil {
		return fmt.Errorf("cli: failed to delete epic: %w", err)
	}

	// Success message
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Deleted epic %q\n", epicID)

	return nil
}
