package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
)

func TestNewImportCmd(t *testing.T) {
	cmd := NewImportCmd()
	if cmd == nil {
		t.Fatal("NewImportCmd() returned nil")
	}
	if !strings.HasPrefix(cmd.Use, "import") {
		t.Errorf("Expected Use to start with 'import', got '%s'", cmd.Use)
	}
}

func TestImportProject_ValidExport(t *testing.T) {
	// Use unique project key to avoid conflicts
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	// Create export file
	exportFile := filepath.Join(t.TempDir(), "export.json")
	exportData := ExportData{
		Version:    "1.0",
		ExportedAt: time.Now().Format(time.RFC3339),
		Project: &models.ProjectIndex{
			ProjectKey:  projectKey,
			ProjectName: "Test Project",
			Issues:      []models.IndexEntry{},
			CreatedAt:   time.Now().Format(time.RFC3339),
			UpdatedAt:   time.Now().Format(time.RFC3339),
		},
		Issues: []*models.Issue{},
		Epics:  []*models.Epic{},
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal export data: %v", err)
	}

	if err := os.WriteFile(exportFile, data, 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	// Import project
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"import", exportFile})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("import command failed: %v\nStderr: %s", err, errBuf.String())
	}

	output := buf.String()
	if !strings.Contains(output, "Imported project") {
		t.Errorf("Expected output to contain 'Imported project', got: %s", output)
	}

	// Verify project was created
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		t.Fatalf("Failed to resolve index path: %v", err)
	}

	var index models.ProjectIndex
	if err := storage.ReadJSON(indexPath, &index); err != nil {
		t.Fatalf("Failed to read project index: %v", err)
	}

	if index.ProjectKey != projectKey {
		t.Errorf("Project Key = %q, want %q", index.ProjectKey, projectKey)
	}
}

func TestImportProject_WithIssues(t *testing.T) {
	// Use unique project key to avoid conflicts
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	// Create export file with issues
	exportFile := filepath.Join(t.TempDir(), "export.json")
	exportData := ExportData{
		Version:    "1.0",
		ExportedAt: time.Now().Format(time.RFC3339),
		Project: &models.ProjectIndex{
			ProjectKey: projectKey,
			Issues: []models.IndexEntry{
				{ID: projectKey + "-1", Title: "Issue 1", Status: models.StatusTODO, Type: models.TypeTask},
				{ID: projectKey + "-2", Title: "Issue 2", Status: models.StatusDOING, Type: models.TypeBug},
			},
		},
		Issues: []*models.Issue{
			{
				ID:     projectKey + "-1",
				Title:  "Issue 1",
				Status: models.StatusTODO,
				Type:   models.TypeTask,
			},
			{
				ID:     projectKey + "-2",
				Title:  "Issue 2",
				Status: models.StatusDOING,
				Type:   models.TypeBug,
			},
		},
		Epics: []*models.Epic{},
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal export data: %v", err)
	}

	if err := os.WriteFile(exportFile, data, 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	// Import project
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"import", exportFile})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("import command failed: %v", err)
	}

	// Verify issues were imported
	issue1Path, err := storage.IssuePath(projectKey, projectKey+"-1")
	if err != nil {
		t.Fatalf("Failed to resolve issue path: %v", err)
	}

	var issue1 models.Issue
	if err := storage.ReadJSON(issue1Path, &issue1); err != nil {
		t.Fatalf("Failed to read issue 1: %v", err)
	}

	if issue1.Title != "Issue 1" {
		t.Errorf("Issue 1 Title = %q, want 'Issue 1'", issue1.Title)
	}

	issue2Path, err := storage.IssuePath(projectKey, projectKey+"-2")
	if err != nil {
		t.Fatalf("Failed to resolve issue path: %v", err)
	}

	var issue2 models.Issue
	if err := storage.ReadJSON(issue2Path, &issue2); err != nil {
		t.Fatalf("Failed to read issue 2: %v", err)
	}

	if issue2.Title != "Issue 2" {
		t.Errorf("Issue 2 Title = %q, want 'Issue 2'", issue2.Title)
	}
}

func TestImportProject_WithEpics(t *testing.T) {
	// Use unique project key to avoid conflicts
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	// Create export file with epics
	exportFile := filepath.Join(t.TempDir(), "export.json")
	exportData := ExportData{
		Version:    "1.0",
		ExportedAt: time.Now().Format(time.RFC3339),
		Project: &models.ProjectIndex{
			ProjectKey: projectKey,
			Issues:     []models.IndexEntry{},
		},
		Issues: []*models.Issue{},
		Epics: []*models.Epic{
			{
				ID:        "E-1",
				Title:     "Epic 1",
				Status:    models.StatusTODO,
				CreatedAt: time.Now().Format(time.RFC3339),
			},
		},
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal export data: %v", err)
	}

	if err := os.WriteFile(exportFile, data, 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	// Import project
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"import", exportFile})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("import command failed: %v", err)
	}

	// Verify epic was imported
	epicPath, err := storage.EpicPath(projectKey, "E-1")
	if err != nil {
		t.Fatalf("Failed to resolve epic path: %v", err)
	}

	var epic models.Epic
	if err := storage.ReadJSON(epicPath, &epic); err != nil {
		t.Fatalf("Failed to read epic: %v", err)
	}

	if epic.Title != "Epic 1" {
		t.Errorf("Epic Title = %q, want 'Epic 1'", epic.Title)
	}
}

func TestImportProject_OverwriteExisting(t *testing.T) {
	// Use unique project key to avoid conflicts
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	// Create project first
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})
	rootCmd.SetOut(new(bytes.Buffer))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Create export file
	exportFile := filepath.Join(t.TempDir(), "export.json")
	exportData := ExportData{
		Version:    "1.0",
		ExportedAt: time.Now().Format(time.RFC3339),
		Project: &models.ProjectIndex{
			ProjectKey: projectKey,
			Issues:     []models.IndexEntry{},
		},
		Issues: []*models.Issue{},
		Epics:  []*models.Epic{},
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal export data: %v", err)
	}

	if err := os.WriteFile(exportFile, data, 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	// Import with overwrite flag
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"import", exportFile, "--overwrite"})

	buf := new(bytes.Buffer)
	rootCmd2.SetOut(buf)

	err = rootCmd2.Execute()
	if err != nil {
		t.Fatalf("import command failed: %v", err)
	}

	// Verify project still exists
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		t.Fatalf("Failed to resolve index path: %v", err)
	}

	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatal("Project index was not created after overwrite")
	}
}

func TestImportProject_ProjectExistsWithoutOverwrite(t *testing.T) {
	// Use unique project key to avoid conflicts
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	// Create project first
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})
	rootCmd.SetOut(new(bytes.Buffer))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Create export file
	exportFile := filepath.Join(t.TempDir(), "export.json")
	exportData := ExportData{
		Version:    "1.0",
		ExportedAt: time.Now().Format(time.RFC3339),
		Project: &models.ProjectIndex{
			ProjectKey: projectKey,
			Issues:     []models.IndexEntry{},
		},
		Issues: []*models.Issue{},
		Epics:  []*models.Epic{},
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal export data: %v", err)
	}

	if err := os.WriteFile(exportFile, data, 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	// Try to import without overwrite flag
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"import", exportFile})

	errBuf := new(bytes.Buffer)
	rootCmd2.SetErr(errBuf)

	err = rootCmd2.Execute()
	if err == nil {
		t.Fatal("import should fail when project exists without overwrite flag")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected error about project already existing, got: %v", err)
	}
}

func TestImportProject_InvalidExportFormat(t *testing.T) {
	// Create invalid export file
	exportFile := filepath.Join(t.TempDir(), "export.json")
	invalidJSON := "{ invalid json }"
	if err := os.WriteFile(exportFile, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	// Try to import
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"import", exportFile})

	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("import should fail for invalid JSON")
	}

	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("Expected error about parsing failure, got: %v", err)
	}
}

func TestImportProject_MissingVersion(t *testing.T) {
	// Create export file with missing version
	exportFile := filepath.Join(t.TempDir(), "export.json")
	exportData := ExportData{
		Version: "", // Missing version
		Project: &models.ProjectIndex{
			ProjectKey: "TEST",
			Issues:     []models.IndexEntry{},
		},
		Issues: []*models.Issue{},
		Epics:  []*models.Epic{},
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal export data: %v", err)
	}

	if err := os.WriteFile(exportFile, data, 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	// Try to import
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"import", exportFile})

	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)

	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("import should fail for invalid export data")
	}

	if !strings.Contains(err.Error(), "invalid export file") {
		t.Errorf("Expected error about invalid export file, got: %v", err)
	}
}

func TestImportProject_RoundTrip(t *testing.T) {
	// Use unique project key to avoid conflicts
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
		os.Remove(projectKey + ".json")
	}()

	// Create project and issues
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})
	rootCmd.SetOut(new(bytes.Buffer))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Create multiple issues
	for i := 1; i <= 2; i++ {
		rootCmd2 := NewRootCmd()
		rootCmd2.SetArgs([]string{"issue", "create", "--project", projectKey, "--title", fmt.Sprintf("Issue %d", i)})
		rootCmd2.SetOut(new(bytes.Buffer))
		if err := rootCmd2.Execute(); err != nil {
			t.Fatalf("Failed to create issue %d: %v", i, err)
		}
	}

	// Export project
	exportFile := projectKey + ".json"
	rootCmd3 := NewRootCmd()
	rootCmd3.SetArgs([]string{"export", projectKey, "--output", exportFile})
	rootCmd3.SetOut(new(bytes.Buffer))

	if err := rootCmd3.Execute(); err != nil {
		t.Fatalf("Failed to export project: %v", err)
	}

	// Remove original project
	projectDir, err := storage.ProjectDir(projectKey)
	if err != nil {
		t.Fatalf("Failed to resolve project directory: %v", err)
	}
	if err := os.RemoveAll(projectDir); err != nil {
		t.Fatalf("Failed to remove project: %v", err)
	}

	// Import project
	rootCmd4 := NewRootCmd()
	rootCmd4.SetArgs([]string{"import", exportFile})

	buf := new(bytes.Buffer)
	rootCmd4.SetOut(buf)

	err = rootCmd4.Execute()
	if err != nil {
		t.Fatalf("import command failed: %v", err)
	}

	// Verify issues were restored
	issue1Path, err := storage.IssuePath(projectKey, projectKey+"-1")
	if err != nil {
		t.Fatalf("Failed to resolve issue path: %v", err)
	}

	var issue1 models.Issue
	if err := storage.ReadJSON(issue1Path, &issue1); err != nil {
		t.Fatalf("Failed to read issue 1: %v", err)
	}

	if issue1.Title != "Issue 1" {
		t.Errorf("Issue 1 Title = %q, want 'Issue 1'", issue1.Title)
	}

	issue2Path, err := storage.IssuePath(projectKey, projectKey+"-2")
	if err != nil {
		t.Fatalf("Failed to resolve issue path: %v", err)
	}

	var issue2 models.Issue
	if err := storage.ReadJSON(issue2Path, &issue2); err != nil {
		t.Fatalf("Failed to read issue 2: %v", err)
	}

	if issue2.Title != "Issue 2" {
		t.Errorf("Issue 2 Title = %q, want 'Issue 2'", issue2.Title)
	}
}

func TestImportProject_InvalidIssueSkipped(t *testing.T) {
	// Use unique project key to avoid conflicts
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	// Create export file with invalid issue
	exportFile := filepath.Join(t.TempDir(), "export.json")
	exportData := ExportData{
		Version:    "1.0",
		ExportedAt: time.Now().Format(time.RFC3339),
		Project: &models.ProjectIndex{
			ProjectKey: projectKey,
			Issues: []models.IndexEntry{
				{ID: projectKey + "-1", Title: "Valid Issue", Status: models.StatusTODO, Type: models.TypeTask},
			},
		},
		Issues: []*models.Issue{
			{
				ID:     projectKey + "-1",
				Title:  "Valid Issue",
				Status: models.StatusTODO,
				Type:   models.TypeTask,
			},
			{
				ID:     projectKey + "-2",
				Title:  "", // Invalid: missing title
				Status: models.StatusTODO,
				Type:   models.TypeTask,
			},
		},
		Epics: []*models.Epic{},
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal export data: %v", err)
	}

	if err := os.WriteFile(exportFile, data, 0644); err != nil {
		t.Fatalf("Failed to write export file: %v", err)
	}

	// Import project
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"import", exportFile})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("import command failed: %v", err)
	}

	// Verify warning was printed
	if !strings.Contains(errBuf.String(), "Warning") {
		t.Error("Expected warning about invalid issue")
	}

	// Verify only valid issue was imported
	issue1Path, err := storage.IssuePath(projectKey, projectKey+"-1")
	if err != nil {
		t.Fatalf("Failed to resolve issue path: %v", err)
	}

	if _, err := os.Stat(issue1Path); os.IsNotExist(err) {
		t.Fatal("Valid issue was not imported")
	}

	issue2Path, err := storage.IssuePath(projectKey, projectKey+"-2")
	if err != nil {
		t.Fatalf("Failed to resolve issue path: %v", err)
	}

	if _, err := os.Stat(issue2Path); err == nil {
		t.Error("Invalid issue should not have been imported")
	}
}
