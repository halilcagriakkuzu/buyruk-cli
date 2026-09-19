package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
	"github.com/spf13/cobra"
)

// ExportData represents the structure of an exported project
type ExportData struct {
	Version    string               `json:"version"`     // Export format version
	ExportedAt string               `json:"exported_at"` // ISO 8601 timestamp
	Project    *models.ProjectIndex `json:"project"`     // Project index
	Issues     []*models.Issue      `json:"issues"`      // All issues
	Epics      []*models.Epic       `json:"epics"`       // All epics (if any)
}

// NewExportCmd creates and returns the export command.
func NewExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <project>",
		Short: "Export a project",
		Long:  "Export a project to a portable JSON file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectKey := args[0]
			return exportProject(projectKey, cmd)
		},
	}

	cmd.Flags().String("output", "", "Output file path (default: <project>.json)")

	return cmd
}

// exportProject exports a project to a JSON file.
func exportProject(projectKey string, cmd *cobra.Command) error {
	// Validate project exists
	projectDir, err := storage.ProjectDir(projectKey)
	if err != nil {
		return fmt.Errorf("cli: failed to resolve project directory: %w", err)
	}

	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return fmt.Errorf("cli: project %q does not exist", projectKey)
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

	// Load all issues
	issues := []*models.Issue{}
	for _, entry := range index.Issues {
		issuePath, err := storage.IssuePath(projectKey, entry.ID)
		if err != nil {
			errOut := cmd.ErrOrStderr()
			fmt.Fprintf(errOut, "Warning: failed to resolve path for issue %s: %v\n", entry.ID, err)
			continue
		}

		var issue models.Issue
		if err := storage.ReadJSON(issuePath, &issue); err != nil {
			errOut := cmd.ErrOrStderr()
			fmt.Fprintf(errOut, "Warning: failed to load issue %s: %v\n", entry.ID, err)
			continue
		}

		issues = append(issues, &issue)
	}

	// Load all epics (if epic directory exists and has files)
	epics := []*models.Epic{}
	epicsDir, err := storage.EpicsDir(projectKey)
	if err == nil {
		if entries, err := os.ReadDir(epicsDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
					continue
				}

				epicPath := filepath.Join(epicsDir, entry.Name())
				var epic models.Epic
				if err := storage.ReadJSON(epicPath, &epic); err != nil {
					errOut := cmd.ErrOrStderr()
					fmt.Fprintf(errOut, "Warning: failed to load epic %s: %v\n", entry.Name(), err)
					continue
				}

				epics = append(epics, &epic)
			}
		}
	}

	// Create export data
	exportData := ExportData{
		Version:    "1.0",
		ExportedAt: time.Now().Format(time.RFC3339),
		Project:    &index,
		Issues:     issues,
		Epics:      epics,
	}

	// Determine output path
	outputPath, _ := cmd.Flags().GetString("output")
	if outputPath == "" {
		outputPath = fmt.Sprintf("%s.json", projectKey)
	}

	// Write export file
	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("cli: failed to marshal export data: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("cli: failed to write export file: %w", err)
	}

	// Success message
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Exported project %q to %s (%d issues, %d epics)\n",
		projectKey, outputPath, len(issues), len(epics))

	return nil
}

// validateExportData validates the export data structure.
// Individual issues and epics are validated during import, not here.
func validateExportData(data *ExportData) error {
	if data.Version == "" {
		return fmt.Errorf("export: missing version")
	}

	if data.Project == nil {
		return fmt.Errorf("export: missing project data")
	}

	if err := data.Project.Validate(); err != nil {
		return fmt.Errorf("export: invalid project data: %w", err)
	}

	// Note: Individual issue and epic validation happens during import
	// to allow skipping invalid items with warnings rather than failing completely.

	return nil
}
