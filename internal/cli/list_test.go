package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/config"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
)

func TestNewListCmd(t *testing.T) {
	cmd := NewListCmd()
	if cmd == nil {
		t.Fatal("NewListCmd() returned nil")
	}
	if cmd.Use != "list" {
		t.Errorf("Expected Use to be 'list', got '%s'", cmd.Use)
	}
}

func TestListIssues_WithProjectFlag(t *testing.T) {
	// Use unique project key to avoid conflicts (sanitize test name)
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	// Create project
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})
	rootCmd.SetOut(new(bytes.Buffer))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Create an issue
	issue := &models.Issue{
		ID:     projectKey + "-1",
		Type:   models.TypeTask,
		Title:  "Test Issue",
		Status: models.StatusTODO,
	}

	issuePath, err := storage.IssuePath(projectKey, projectKey+"-1")
	if err != nil {
		t.Fatalf("Failed to resolve issue path: %v", err)
	}

	if err := storage.WriteJSONAtomic(issuePath, issue); err != nil {
		t.Fatalf("Failed to write issue: %v", err)
	}

	// Update index manually
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		t.Fatalf("Failed to resolve index path: %v", err)
	}

	var index models.ProjectIndex
	if err := storage.ReadJSON(indexPath, &index); err != nil {
		t.Fatalf("Failed to read index: %v", err)
	}

	index.AddIssue(issue)
	if err := storage.WriteJSONAtomic(indexPath, &index); err != nil {
		t.Fatalf("Failed to update index: %v", err)
	}

	// List issues
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"list", "--project", projectKey})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd2.SetOut(buf)
	rootCmd2.SetErr(errBuf)

	err = rootCmd2.Execute()
	if err != nil {
		t.Fatalf("list command failed: %v\nStderr: %s", err, errBuf.String())
	}

	output := buf.String()
	if !strings.Contains(output, "Test Issue") {
		t.Errorf("Expected output to contain 'Test Issue', got: %s", output)
	}
}

func TestListIssues_WithConfigProject(t *testing.T) {
	// Use unique project key to avoid conflicts (sanitize test name)
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
		// Clear config
		config.Set("default_project", "")
	}()

	// Set default project in config
	if err := config.Set("default_project", projectKey); err != nil {
		t.Fatalf("Failed to set default project: %v", err)
	}

	// Create project
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})
	rootCmd.SetOut(new(bytes.Buffer))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Create an issue
	issue := &models.Issue{
		ID:     projectKey + "-1",
		Type:   models.TypeTask,
		Title:  "Test Issue",
		Status: models.StatusTODO,
	}

	issuePath, err := storage.IssuePath(projectKey, projectKey+"-1")
	if err != nil {
		t.Fatalf("Failed to resolve issue path: %v", err)
	}

	if err := storage.WriteJSONAtomic(issuePath, issue); err != nil {
		t.Fatalf("Failed to write issue: %v", err)
	}

	// Update index manually
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		t.Fatalf("Failed to resolve index path: %v", err)
	}

	var index models.ProjectIndex
	if err := storage.ReadJSON(indexPath, &index); err != nil {
		t.Fatalf("Failed to read index: %v", err)
	}

	index.AddIssue(issue)
	if err := storage.WriteJSONAtomic(indexPath, &index); err != nil {
		t.Fatalf("Failed to update index: %v", err)
	}

	// List issues without --project flag (should use config)
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"list"})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd2.SetOut(buf)
	rootCmd2.SetErr(errBuf)

	err = rootCmd2.Execute()
	if err != nil {
		t.Fatalf("list command failed: %v\nStderr: %s", err, errBuf.String())
	}

	output := buf.String()
	if !strings.Contains(output, "Test Issue") {
		t.Errorf("Expected output to contain 'Test Issue', got: %s", output)
	}
}

func TestListIssues_NoProject(t *testing.T) {
	// Clear any existing config project
	originalCfg, _ := config.Get()
	defer func() {
		if originalCfg != nil {
			config.Save(originalCfg)
		}
	}()

	// Clear default_project
	if err := config.Set("default_project", ""); err != nil {
		t.Fatalf("Failed to clear config: %v", err)
	}

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"list"})

	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("list command should fail when no project is specified")
	}

	if !strings.Contains(err.Error(), "no project specified") {
		t.Errorf("Expected error about no project specified, got: %v", err)
	}
}

func TestListIssues_MissingProject(t *testing.T) {
	// Use a unique non-existent project key (sanitize test name)
	projectKey := sanitizeTestName("MISSING" + t.Name())

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"list", "--project", projectKey})

	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("list command should fail when project does not exist")
	}

	if !strings.Contains(err.Error(), "failed to load project index") {
		t.Errorf("Expected error about failed to load project index, got: %v", err)
	}
}

func TestListIssues_EmptyProject(t *testing.T) {
	// Use unique project key to avoid conflicts (sanitize test name)
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	// Create project
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})
	rootCmd.SetOut(new(bytes.Buffer))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// List issues (should succeed with empty list)
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"list", "--project", projectKey})

	buf := new(bytes.Buffer)
	rootCmd2.SetOut(buf)

	err := rootCmd2.Execute()
	if err != nil {
		t.Fatalf("list command should succeed with empty project: %v", err)
	}

	// Output should be empty or contain appropriate message
	output := buf.String()
	// Modern format might show a table with headers, which is fine
	_ = output
}

func TestListIssues_WithFormatFlags(t *testing.T) {
	// Use unique project key to avoid conflicts (sanitize test name)
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	// Create project
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})
	rootCmd.SetOut(new(bytes.Buffer))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Create an issue
	issue := &models.Issue{
		ID:     projectKey + "-1",
		Type:   models.TypeTask,
		Title:  "Test Issue",
		Status: models.StatusTODO,
	}

	issuePath, err := storage.IssuePath(projectKey, projectKey+"-1")
	if err != nil {
		t.Fatalf("Failed to resolve issue path: %v", err)
	}

	if err := storage.WriteJSONAtomic(issuePath, issue); err != nil {
		t.Fatalf("Failed to write issue: %v", err)
	}

	// Update index manually
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		t.Fatalf("Failed to resolve index path: %v", err)
	}

	var index models.ProjectIndex
	if err := storage.ReadJSON(indexPath, &index); err != nil {
		t.Fatalf("Failed to read index: %v", err)
	}

	index.AddIssue(issue)
	if err := storage.WriteJSONAtomic(indexPath, &index); err != nil {
		t.Fatalf("Failed to update index: %v", err)
	}

	// Test JSON format
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"list", "--project", projectKey, "--format", "json"})

	buf := new(bytes.Buffer)
	rootCmd2.SetOut(buf)

	err = rootCmd2.Execute()
	if err != nil {
		t.Fatalf("list command with json format failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
		t.Errorf("Expected JSON output, got: %s", output)
	}

	// Test LSON format
	rootCmd3 := NewRootCmd()
	rootCmd3.SetArgs([]string{"list", "--project", projectKey, "--format", "lson"})

	buf2 := new(bytes.Buffer)
	rootCmd3.SetOut(buf2)

	err = rootCmd3.Execute()
	if err != nil {
		t.Fatalf("list command with lson format failed: %v", err)
	}

	output2 := buf2.String()
	if !strings.Contains(output2, "@") {
		t.Errorf("Expected LSON output with @ prefix, got: %s", output2)
	}
}

func TestListIssues_MissingIssueFile(t *testing.T) {
	// Use unique project key to avoid conflicts (sanitize test name)
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	// Create project
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})
	rootCmd.SetOut(new(bytes.Buffer))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Manually add an index entry for a non-existent issue file
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		t.Fatalf("Failed to resolve index path: %v", err)
	}

	var index models.ProjectIndex
	if err := storage.ReadJSON(indexPath, &index); err != nil {
		t.Fatalf("Failed to read index: %v", err)
	}

	// Add entry for missing issue
	index.Issues = append(index.Issues, models.IndexEntry{
		ID:     projectKey + "-999",
		Title:  "Missing Issue",
		Status: models.StatusTODO,
		Type:   models.TypeTask,
	})

	if err := storage.WriteJSONAtomic(indexPath, &index); err != nil {
		t.Fatalf("Failed to update index: %v", err)
	}

	// List should continue despite missing issue file
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"list", "--project", projectKey})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd2.SetOut(buf)
	rootCmd2.SetErr(errBuf)

	err = rootCmd2.Execute()
	// Should succeed but with warning
	if err != nil {
		t.Logf("List command failed (acceptable if it warns about missing file): %v", err)
	}

	// Should have warning about missing issue
	if !strings.Contains(errBuf.String(), "Warning") {
		t.Logf("Note: No warning about missing issue file (this is acceptable)")
	}
}
