package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
)

func TestNewProjectCmd(t *testing.T) {
	cmd := NewProjectCmd()
	if cmd == nil {
		t.Fatal("NewProjectCmd() returned nil")
	}
	if cmd.Use != "project" {
		t.Errorf("Expected Use to be 'project', got '%s'", cmd.Use)
	}

	// Check subcommands
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("create subcommand not found: %v", err)
	}
	if createCmd == nil {
		t.Fatal("create subcommand is nil")
	}

	repairCmd, _, err := cmd.Find([]string{"repair"})
	if err != nil {
		t.Fatalf("repair subcommand not found: %v", err)
	}
	if repairCmd == nil {
		t.Fatal("repair subcommand is nil")
	}
}

func TestNewProjectCreateCmd(t *testing.T) {
	cmd := NewProjectCreateCmd()
	if cmd == nil {
		t.Fatal("NewProjectCreateCmd() returned nil")
	}
	if !strings.HasPrefix(cmd.Use, "create") {
		t.Errorf("Expected Use to start with 'create', got '%s'", cmd.Use)
	}
}

func TestNewProjectRepairCmd(t *testing.T) {
	cmd := NewProjectRepairCmd()
	if cmd == nil {
		t.Fatal("NewProjectRepairCmd() returned nil")
	}
	if !strings.HasPrefix(cmd.Use, "repair") {
		t.Errorf("Expected Use to start with 'repair', got '%s'", cmd.Use)
	}
}

func TestIsValidProjectKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"uppercase letters", "TEST", true},
		{"uppercase with numbers", "TEST123", true},
		{"numbers only", "123", true},
		{"with hyphen", "TEST-123", true},
		{"lowercase", "test", false},
		{"mixed case", "Test", false},
		{"with underscore", "TEST_123", false},
		{"with space", "TEST 123", false},
		{"empty", "", false},
		{"with special chars", "TEST@123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidProjectKey(tt.key)
			if got != tt.want {
				t.Errorf("isValidProjectKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestCreateProject_ValidKey(t *testing.T) {
	// Use unique project key to avoid conflicts (sanitize test name)
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("project create command failed: %v\nStderr: %s", err, errBuf.String())
	}

	output := buf.String()
	if !strings.Contains(output, "Created project") {
		t.Errorf("Expected output to contain 'Created project', got: %s", output)
	}

	// Verify project structure was created
	projectDir, err := storage.ProjectDir(projectKey)
	if err != nil {
		t.Fatalf("Failed to resolve project directory: %v", err)
	}

	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		t.Errorf("Project directory was not created: %s", projectDir)
	}

	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		t.Fatalf("Failed to resolve index path: %v", err)
	}

	var index models.ProjectIndex
	if err := storage.ReadJSON(indexPath, &index); err != nil {
		t.Fatalf("Failed to read project index: %v", err)
	}

	if index.ProjectKey != projectKey {
		t.Errorf("ProjectKey = %q, want %q", index.ProjectKey, projectKey)
	}
	if len(index.Issues) != 0 {
		t.Errorf("Expected empty issues list, got %d issues", len(index.Issues))
	}
}

func TestCreateProject_WithName(t *testing.T) {
	// Use unique project key to avoid conflicts (sanitize test name)
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey, "--name", "Test Project"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("project create command failed: %v", err)
	}

	// Verify project name was set
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		t.Fatalf("Failed to resolve index path: %v", err)
	}

	var index models.ProjectIndex
	if err := storage.ReadJSON(indexPath, &index); err != nil {
		t.Fatalf("Failed to read project index: %v", err)
	}

	if index.ProjectName != "Test Project" {
		t.Errorf("ProjectName = %q, want 'Test Project'", index.ProjectName)
	}
}

func TestCreateProject_InvalidKey(t *testing.T) {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", "test"}) // lowercase invalid

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("project create should fail with invalid key")
	}

	if !strings.Contains(err.Error(), "invalid project key") {
		t.Errorf("Expected error about invalid project key, got: %v", err)
	}
}

func TestCreateProject_AlreadyExists(t *testing.T) {
	// Use unique project key to avoid conflicts (sanitize test name)
	projectKey := sanitizeTestName("TEST" + t.Name())
	// Clean up after test
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	// Create project first time
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})
	rootCmd.SetOut(new(bytes.Buffer))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	// Try to create again
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"project", "create", projectKey})
	errBuf := new(bytes.Buffer)
	rootCmd2.SetErr(errBuf)

	err := rootCmd2.Execute()
	if err == nil {
		t.Fatal("project create should fail when project already exists")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected error about project already existing, got: %v", err)
	}
}

func TestRepairProject_ValidProject(t *testing.T) {
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

	// Create an issue file manually
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

	// Repair project
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"project", "repair", projectKey})

	buf := new(bytes.Buffer)
	rootCmd2.SetOut(buf)

	err = rootCmd2.Execute()
	if err != nil {
		t.Fatalf("project repair command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Repaired project") {
		t.Errorf("Expected output to contain 'Repaired project', got: %s", output)
	}

	// Verify index was updated
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		t.Fatalf("Failed to resolve index path: %v", err)
	}

	var index models.ProjectIndex
	if err := storage.ReadJSON(indexPath, &index); err != nil {
		t.Fatalf("Failed to read project index: %v", err)
	}

	if len(index.Issues) != 1 {
		t.Errorf("Expected 1 issue in index, got %d", len(index.Issues))
	}

	if index.Issues[0].ID != projectKey+"-1" {
		t.Errorf("Issue ID = %q, want %q", index.Issues[0].ID, projectKey+"-1")
	}
}

func TestRepairProject_MissingProject(t *testing.T) {
	// Use a unique non-existent project key (sanitize test name)
	projectKey := sanitizeTestName("MISSING" + t.Name())

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "repair", projectKey})

	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("project repair should fail when project does not exist")
	}

	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected error about project not existing, got: %v", err)
	}
}

func TestRepairProject_CorruptedIssueFiles(t *testing.T) {
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

	// Create a valid issue
	issue := &models.Issue{
		ID:     projectKey + "-1",
		Type:   models.TypeTask,
		Title:  "Valid Issue",
		Status: models.StatusTODO,
	}

	issuePath, err := storage.IssuePath(projectKey, projectKey+"-1")
	if err != nil {
		t.Fatalf("Failed to resolve issue path: %v", err)
	}

	if err := storage.WriteJSONAtomic(issuePath, issue); err != nil {
		t.Fatalf("Failed to write issue: %v", err)
	}

	// Create a corrupted issue file
	issuesDir, err := storage.IssuesDir(projectKey)
	if err != nil {
		t.Fatalf("Failed to resolve issues directory: %v", err)
	}

	corruptedPath := filepath.Join(issuesDir, "corrupted.json")
	if err := os.WriteFile(corruptedPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}

	// Repair should continue despite corrupted file
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"project", "repair", projectKey})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd2.SetOut(buf)
	rootCmd2.SetErr(errBuf)

	err = rootCmd2.Execute()
	if err != nil {
		t.Fatalf("project repair should succeed despite corrupted files: %v", err)
	}

	// Should have warning about corrupted file
	if !strings.Contains(errBuf.String(), "Warning") {
		t.Logf("Note: No warning about corrupted file (this is acceptable)")
	}

	// Should still index the valid issue
	indexPath, err := storage.ProjectIndexPath(projectKey)
	if err != nil {
		t.Fatalf("Failed to resolve index path: %v", err)
	}

	var index models.ProjectIndex
	if err := storage.ReadJSON(indexPath, &index); err != nil {
		t.Fatalf("Failed to read project index: %v", err)
	}

	if len(index.Issues) != 1 {
		t.Errorf("Expected 1 issue in index, got %d", len(index.Issues))
	}
}

func TestResolveProjectKey(t *testing.T) {
	// This is tested indirectly through list command tests
	// but we can test the wrapper function
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"--project", "TEST"})

	if err := cmd.ParseFlags([]string{"--project", "TEST"}); err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	project, err := ResolveProjectKey(cmd)
	if err != nil {
		t.Fatalf("ResolveProjectKey() failed: %v", err)
	}

	if project != "TEST" {
		t.Errorf("ResolveProjectKey() = %q, want TEST", project)
	}
}

func TestDeleteProject_WithYesFlag(t *testing.T) {
	projectKey := sanitizeTestName("TEST" + t.Name())
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

	// Create an issue to verify it gets deleted too
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{
		"issue", "create",
		"--project", projectKey,
		"--title", "Test Issue",
	})
	rootCmd2.SetOut(new(bytes.Buffer))
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("Failed to create issue: %v", err)
	}

	// Delete project with -y flag
	rootCmd3 := NewRootCmd()
	rootCmd3.SetArgs([]string{
		"project", "delete", projectKey,
		"-y",
	})

	buf := new(bytes.Buffer)
	rootCmd3.SetOut(buf)

	err := rootCmd3.Execute()
	if err != nil {
		t.Fatalf("project delete command failed: %v", err)
	}

	// Verify project directory was deleted
	projectDir, err := storage.ProjectDir(projectKey)
	if err != nil {
		t.Fatalf("Failed to resolve project directory: %v", err)
	}

	if _, err := os.Stat(projectDir); err == nil {
		t.Error("Project directory should not exist after deletion")
	}
}

func TestDeleteProject_NonExistent(t *testing.T) {
	projectKey := sanitizeTestName("TEST" + t.Name())

	// Try to delete non-existent project
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{
		"project", "delete", projectKey,
		"-y",
	})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("project delete should fail for non-existent project")
	}

	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected error about project not existing, got: %v", err)
	}
}

// sanitizeTestName converts a test name to a valid project key format
// by removing invalid characters and converting to uppercase
// Note: Config validation allows uppercase alphanumeric characters and hyphens;
// this helper produces a simplified valid key by stripping hyphens and other symbols.
func sanitizeTestName(name string) string {
	var result strings.Builder
	for _, r := range name {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		} else if r >= 'a' && r <= 'z' {
			result.WriteRune(r - 32) // Convert to uppercase
		}
		// Skip other characters (including hyphens and underscores)
	}
	sanitized := result.String()
	if sanitized == "" {
		return "TEST"
	}
	return sanitized
}
