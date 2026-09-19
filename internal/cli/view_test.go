package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
)

func TestNewViewCmd(t *testing.T) {
	cmd := NewViewCmd()
	if cmd == nil {
		t.Fatal("NewViewCmd() returned nil")
	}
	if !strings.HasPrefix(cmd.Use, "view") {
		t.Errorf("Expected Use to start with 'view', got '%s'", cmd.Use)
	}
}

func TestViewIssue_Success(t *testing.T) {
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

	// Create an issue
	issueID := projectKey + "-1"
	issue := &models.Issue{
		ID:          issueID,
		Type:        models.TypeTask,
		Title:       "Test Issue",
		Status:      models.StatusTODO,
		Priority:    models.PriorityHIGH,
		Description: "Test description",
		CreatedAt:   "2024-01-01T00:00:00Z",
		UpdatedAt:   "2024-01-01T00:00:00Z",
	}

	issuePath, err := storage.IssuePath(projectKey, issueID)
	if err != nil {
		t.Fatalf("Failed to resolve issue path: %v", err)
	}

	if err := storage.WriteJSONAtomic(issuePath, issue); err != nil {
		t.Fatalf("Failed to write issue: %v", err)
	}

	// View the issue
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"view", issueID})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd2.SetOut(buf)
	rootCmd2.SetErr(errBuf)

	err = rootCmd2.Execute()
	if err != nil {
		t.Fatalf("view command failed: %v\nStderr: %s", err, errBuf.String())
	}

	output := buf.String()
	if !strings.Contains(output, issueID) {
		t.Errorf("Expected output to contain issue ID %q, got: %s", issueID, output)
	}
	if !strings.Contains(output, "Test Issue") {
		t.Errorf("Expected output to contain issue title, got: %s", output)
	}
}

func TestViewIssue_NotFound(t *testing.T) {
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

	// Try to view non-existent issue
	issueID := projectKey + "-999"
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{"view", issueID})

	errBuf := new(bytes.Buffer)
	rootCmd2.SetErr(errBuf)

	err := rootCmd2.Execute()
	if err == nil {
		t.Fatal("view command should fail for non-existent issue")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error about issue not found, got: %v", err)
	}
}

func TestViewIssue_InvalidID(t *testing.T) {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"view", "INVALID-ID"})

	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("view command should fail with invalid ID")
	}

	if !strings.Contains(err.Error(), "invalid issue ID") {
		t.Errorf("Expected error about invalid ID, got: %v", err)
	}
}
