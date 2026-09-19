package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
)

func TestNewEpicCmd(t *testing.T) {
	cmd := NewEpicCmd()
	if cmd == nil {
		t.Fatal("NewEpicCmd() returned nil")
	}
	if cmd.Use != "epic" {
		t.Errorf("Expected Use to be 'epic', got '%s'", cmd.Use)
	}

	// Check subcommands
	createCmd, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("create subcommand not found: %v", err)
	}
	if createCmd == nil {
		t.Fatal("create subcommand is nil")
	}
}

func TestCreateEpic_Minimal(t *testing.T) {
	projectKey := sanitizeTestName("TEST" + t.Name())
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

	// Create epic
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{
		"epic", "create",
		"--project", projectKey,
		"--title", "Test Epic",
	})

	buf := new(bytes.Buffer)
	rootCmd2.SetOut(buf)

	err := rootCmd2.Execute()
	if err != nil {
		t.Fatalf("epic create command failed: %v", err)
	}

	// Verify epic was created
	epicPath, err := storage.EpicPath(projectKey, "E-1")
	if err != nil {
		t.Fatalf("Failed to resolve epic path: %v", err)
	}

	var epic models.Epic
	if err := storage.ReadJSON(epicPath, &epic); err != nil {
		t.Fatalf("Failed to read epic: %v", err)
	}

	if epic.ID != "E-1" {
		t.Errorf("Epic ID = %q, want E-1", epic.ID)
	}
	if epic.Title != "Test Epic" {
		t.Errorf("Epic Title = %q, want 'Test Epic'", epic.Title)
	}
	if epic.Status != models.StatusTODO {
		t.Errorf("Epic Status = %q, want %q", epic.Status, models.StatusTODO)
	}
}

func TestCreateEpic_WithCustomID(t *testing.T) {
	projectKey := sanitizeTestName("TEST" + t.Name())
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

	// Create epic with custom ID
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{
		"epic", "create",
		"--project", projectKey,
		"--id", "CUSTOM-1",
		"--title", "Custom Epic",
		"--status", "DOING",
	})

	buf := new(bytes.Buffer)
	rootCmd2.SetOut(buf)

	err := rootCmd2.Execute()
	if err != nil {
		t.Fatalf("epic create command failed: %v", err)
	}

	// Verify epic was created with custom ID
	epicPath, err := storage.EpicPath(projectKey, "CUSTOM-1")
	if err != nil {
		t.Fatalf("Failed to resolve epic path: %v", err)
	}

	var epic models.Epic
	if err := storage.ReadJSON(epicPath, &epic); err != nil {
		t.Fatalf("Failed to read epic: %v", err)
	}

	if epic.ID != "CUSTOM-1" {
		t.Errorf("Epic ID = %q, want CUSTOM-1", epic.ID)
	}
	if epic.Status != models.StatusDOING {
		t.Errorf("Epic Status = %q, want %q", epic.Status, models.StatusDOING)
	}
}

func TestViewEpic(t *testing.T) {
	projectKey := sanitizeTestName("TEST" + t.Name())
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	// Create project and epic
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})
	rootCmd.SetOut(new(bytes.Buffer))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{
		"epic", "create",
		"--project", projectKey,
		"--title", "View Test Epic",
	})
	rootCmd2.SetOut(new(bytes.Buffer))
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// View epic
	rootCmd3 := NewRootCmd()
	rootCmd3.SetArgs([]string{
		"epic", "view", "E-1",
		"--project", projectKey,
	})

	buf := new(bytes.Buffer)
	rootCmd3.SetOut(buf)

	err := rootCmd3.Execute()
	if err != nil {
		t.Fatalf("epic view command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "E-1") {
		t.Error("epic view output missing epic ID")
	}
	if !strings.Contains(output, "View Test Epic") {
		t.Error("epic view output missing epic title")
	}
}

func TestDeleteEpic_WithYesFlag(t *testing.T) {
	projectKey := sanitizeTestName("TEST" + t.Name())
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	// Create project and epic
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"project", "create", projectKey})
	rootCmd.SetOut(new(bytes.Buffer))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{
		"epic", "create",
		"--project", projectKey,
		"--title", "Epic to Delete",
	})
	rootCmd2.SetOut(new(bytes.Buffer))
	if err := rootCmd2.Execute(); err != nil {
		t.Fatalf("Failed to create epic: %v", err)
	}

	// Delete epic with -y flag
	rootCmd3 := NewRootCmd()
	rootCmd3.SetArgs([]string{
		"epic", "delete", "E-1",
		"--project", projectKey,
		"-y",
	})

	buf := new(bytes.Buffer)
	rootCmd3.SetOut(buf)

	err := rootCmd3.Execute()
	if err != nil {
		t.Fatalf("epic delete command failed: %v", err)
	}

	// Verify epic was deleted
	epicPath, err := storage.EpicPath(projectKey, "E-1")
	if err != nil {
		t.Fatalf("Failed to resolve epic path: %v", err)
	}

	if _, err := os.Stat(epicPath); err == nil {
		t.Error("Epic file should not exist after deletion")
	}
}

func TestDeleteEpic_NonExistent(t *testing.T) {
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

	// Try to delete non-existent epic
	rootCmd2 := NewRootCmd()
	rootCmd2.SetArgs([]string{
		"epic", "delete", "E-999",
		"--project", projectKey,
		"-y",
	})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd2.SetOut(buf)
	rootCmd2.SetErr(errBuf)

	err := rootCmd2.Execute()
	if err == nil {
		t.Fatal("epic delete should fail for non-existent epic")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error about epic not found, got: %v", err)
	}
}
