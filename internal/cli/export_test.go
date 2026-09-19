package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
)

func TestNewExportCmd(t *testing.T) {
	cmd := NewExportCmd()
	if cmd == nil {
		t.Fatal("NewExportCmd() returned nil")
	}
	if !strings.HasPrefix(cmd.Use, "export") {
		t.Errorf("Expected Use to start with 'export', got '%s'", cmd.Use)
	}
}

func TestExportProject_ValidProject(t *testing.T) {
	// Use unique project key to avoid conflicts
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
		os.Remove(projectKey + ".json")
	}()

	// Create project first
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})
	rootCmd.SetOut(new(bytes.Buffer))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Create an issue
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"issue", "create", "--project", projectKey, "--title", "Test Issue"})
	rootCmd2.SetOut(new(bytes.Buffer))
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("Failed to create issue: %v", err)
	}

	// Export project
	exportFile := projectKey + ".json"
	rootCmd3 := NewRootCmd()
	rootCmd3.SetArgs([]string{"export", projectKey, "--output", exportFile})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd3.SetOut(buf)
	rootCmd3.SetErr(errBuf)

	err := rootCmd3.Execute()
	if err != nil {
		t.Fatalf("export command failed: %v\nStderr: %s", err, errBuf.String())
	}

	output := buf.String()
	if !strings.Contains(output, "Exported project") {
		t.Errorf("Expected output to contain 'Exported project', got: %s", output)
	}

	// Verify export file exists
	if _, err := os.Stat(exportFile); os.IsNotExist(err) {
		t.Fatal("Export file was not created")
	}

	// Verify export file content
	var exportData ExportData
	data, err := os.ReadFile(exportFile)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	if err := json.Unmarshal(data, &exportData); err != nil {
		t.Fatalf("Failed to parse export file: %v", err)
	}

	if exportData.Version != "1.0" {
		t.Errorf("Export Version = %q, want '1.0'", exportData.Version)
	}

	if exportData.Project == nil {
		t.Fatal("Export Project is nil")
	}

	if exportData.Project.ProjectKey != projectKey {
		t.Errorf("Export Project.ProjectKey = %q, want %q", exportData.Project.ProjectKey, projectKey)
	}

	if len(exportData.Issues) != 1 {
		t.Errorf("Export Issues count = %d, want 1", len(exportData.Issues))
	}

	if exportData.Issues[0].Title != "Test Issue" {
		t.Errorf("Export Issue Title = %q, want 'Test Issue'", exportData.Issues[0].Title)
	}
}

func TestExportProject_WithEpics(t *testing.T) {
	// Use unique project key to avoid conflicts
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
		os.Remove(projectKey + ".json")
	}()

	// Create project first
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})
	rootCmd.SetOut(new(bytes.Buffer))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Create an epic manually (since there's no epic command yet)
	epic := &models.Epic{
		ID:        "E-1",
		Title:     "Test Epic",
		Status:    models.StatusTODO,
		CreatedAt: "2024-01-01T00:00:00Z",
	}

	epicPath, err := storage.EpicPath(projectKey, epic.ID)
	if err != nil {
		t.Fatalf("Failed to resolve epic path: %v", err)
	}

	if err := storage.WriteJSONAtomic(epicPath, epic); err != nil {
		t.Fatalf("Failed to write epic: %v", err)
	}

	// Export project
	exportFile := projectKey + ".json"
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"export", projectKey, "--output", exportFile})

	buf := new(bytes.Buffer)
	rootCmd2.SetOut(buf)

	err = rootCmd2.Execute()
	if err != nil {
		t.Fatalf("export command failed: %v", err)
	}

	// Verify export file content
	var exportData ExportData
	data, err := os.ReadFile(exportFile)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	if err := json.Unmarshal(data, &exportData); err != nil {
		t.Fatalf("Failed to parse export file: %v", err)
	}

	if len(exportData.Epics) != 1 {
		t.Errorf("Export Epics count = %d, want 1", len(exportData.Epics))
	}

	if exportData.Epics[0].ID != "E-1" {
		t.Errorf("Export Epic ID = %q, want 'E-1'", exportData.Epics[0].ID)
	}
}

func TestExportProject_CustomOutputPath(t *testing.T) {
	// Use unique project key to avoid conflicts
	projectKey := sanitizeTestName("TEST" + t.Name())
	customPath := filepath.Join(t.TempDir(), "custom-export.json")
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

	// Export project with custom output path
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"export", projectKey, "--output", customPath})

	buf := new(bytes.Buffer)
	rootCmd2.SetOut(buf)

	err := rootCmd2.Execute()
	if err != nil {
		t.Fatalf("export command failed: %v", err)
	}

	// Verify export file exists at custom path
	if _, err := os.Stat(customPath); os.IsNotExist(err) {
		t.Fatal("Export file was not created at custom path")
	}
}

func TestExportProject_ProjectNotFound(t *testing.T) {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"export", "NONEXISTENT"})

	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("export should fail for non-existent project")
	}

	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected error about project not existing, got: %v", err)
	}
}

func TestExportProject_MultipleIssues(t *testing.T) {
	// Use unique project key to avoid conflicts
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
		os.Remove(projectKey + ".json")
	}()

	// Create project first
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})
	rootCmd.SetOut(new(bytes.Buffer))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Create multiple issues
	for i := 1; i <= 3; i++ {
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

	buf := new(bytes.Buffer)
	rootCmd3.SetOut(buf)

	err := rootCmd3.Execute()
	if err != nil {
		t.Fatalf("export command failed: %v", err)
	}

	// Verify export file content
	var exportData ExportData
	data, err := os.ReadFile(exportFile)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	if err := json.Unmarshal(data, &exportData); err != nil {
		t.Fatalf("Failed to parse export file: %v", err)
	}

	if len(exportData.Issues) != 3 {
		t.Errorf("Export Issues count = %d, want 3", len(exportData.Issues))
	}
}

func TestValidateExportData(t *testing.T) {
	tests := []struct {
		name    string
		data    *ExportData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid data",
			data: &ExportData{
				Version: "1.0",
				Project: &models.ProjectIndex{
					ProjectKey: "TEST",
					Issues:     []models.IndexEntry{},
				},
				Issues: []*models.Issue{},
				Epics:  []*models.Epic{},
			},
			wantErr: false,
		},
		{
			name: "missing version",
			data: &ExportData{
				Version: "",
				Project: &models.ProjectIndex{
					ProjectKey: "TEST",
					Issues:     []models.IndexEntry{},
				},
				Issues: []*models.Issue{},
				Epics:  []*models.Epic{},
			},
			wantErr: true,
			errMsg:  "missing version",
		},
		{
			name: "missing project",
			data: &ExportData{
				Version: "1.0",
				Project: nil,
				Issues:  []*models.Issue{},
				Epics:   []*models.Epic{},
			},
			wantErr: true,
			errMsg:  "missing project data",
		},
		{
			name: "invalid issue (allowed - validated during import)",
			data: &ExportData{
				Version: "1.0",
				Project: &models.ProjectIndex{
					ProjectKey: "TEST",
					Issues: []models.IndexEntry{
						{ID: "TEST-1", Title: "Issue 1", Status: models.StatusTODO},
					},
				},
				Issues: []*models.Issue{
					{ID: "TEST-1", Title: ""}, // Missing title - will be skipped during import
				},
				Epics: []*models.Epic{},
			},
			wantErr: false, // Individual issues are validated during import, not here
		},
		{
			name: "index inconsistency (allowed - validated during import)",
			data: &ExportData{
				Version: "1.0",
				Project: &models.ProjectIndex{
					ProjectKey: "TEST",
					Issues: []models.IndexEntry{
						{ID: "TEST-1", Title: "Issue 1", Status: models.StatusTODO},
					},
				},
				Issues: []*models.Issue{}, // Missing issue - will be skipped during import
				Epics:  []*models.Epic{},
			},
			wantErr: false, // Consistency is checked during import, not here
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExportData(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Fatal("validateExportData() should return error")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateExportData() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateExportData() error = %v, want nil", err)
				}
			}
		})
	}
}
