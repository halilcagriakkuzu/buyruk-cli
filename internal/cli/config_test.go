package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/config"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
)

func TestNewConfigCmd(t *testing.T) {
	cmd := NewConfigCmd()
	if cmd == nil {
		t.Fatal("NewConfigCmd() returned nil")
	}
	if cmd.Use != "config" {
		t.Errorf("Expected Use to be 'config', got '%s'", cmd.Use)
	}

	// Check subcommands
	getCmd, _, err := cmd.Find([]string{"get"})
	if err != nil {
		t.Fatalf("get subcommand not found: %v", err)
	}
	if getCmd == nil {
		t.Fatal("get subcommand is nil")
	}

	setCmd, _, err := cmd.Find([]string{"set"})
	if err != nil {
		t.Fatalf("set subcommand not found: %v", err)
	}
	if setCmd == nil {
		t.Fatal("set subcommand is nil")
	}

	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatalf("list subcommand not found: %v", err)
	}
	if listCmd == nil {
		t.Fatal("list subcommand is nil")
	}
}

func TestConfigGet_ValidKey(t *testing.T) {
	// Save original config
	originalCfg, _ := config.Get()
	defer func() {
		if originalCfg != nil {
			config.Save(originalCfg)
		}
	}()

	// Set a test value
	if err := config.Set("default_format", "json"); err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"config", "get", "default_format"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config get command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "json") {
		t.Errorf("Expected output to contain 'json', got: %s", output)
	}
}

func TestConfigGet_InvalidKey(t *testing.T) {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"config", "get", "invalid_key"})

	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("config get should fail with invalid key")
	}

	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("Expected error about unknown config key, got: %v", err)
	}
}

func TestConfigGet_UnsetValue(t *testing.T) {
	// Save original config
	originalCfg, _ := config.Get()
	defer func() {
		if originalCfg != nil {
			config.Save(originalCfg)
		}
	}()

	// Clear default_project
	cfg, _ := config.Get()
	cfg.DefaultProject = ""
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Failed to clear config: %v", err)
	}

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"config", "get", "default_project"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config get command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "(not set)") {
		t.Errorf("Expected output to contain '(not set)', got: %s", output)
	}
}

func TestConfigGet_JSONFormat(t *testing.T) {
	// Save original config
	originalCfg, _ := config.Get()
	defer func() {
		if originalCfg != nil {
			config.Save(originalCfg)
		}
	}()

	// Set a test value
	if err := config.Set("default_format", "json"); err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"config", "get", "default_format", "--format", "json"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config get command failed: %v", err)
	}

	output := buf.String()
	var result map[string]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if result["default_format"] != "json" {
		t.Errorf("Expected default_format to be 'json', got: %s", result["default_format"])
	}
}

func TestConfigGet_LSONFormat(t *testing.T) {
	// Save original config
	originalCfg, _ := config.Get()
	defer func() {
		if originalCfg != nil {
			config.Save(originalCfg)
		}
	}()

	// Set a test value
	if err := config.Set("default_format", "lson"); err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"config", "get", "default_format", "--format", "lson"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config get command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "@DEFAULT_FORMAT:") {
		t.Errorf("Expected output to contain '@DEFAULT_FORMAT:', got: %s", output)
	}
	if !strings.Contains(output, "lson") {
		t.Errorf("Expected output to contain 'lson', got: %s", output)
	}
}

func TestConfigSet_ValidKeyValue(t *testing.T) {
	// Save original config
	originalCfg, _ := config.Get()
	defer func() {
		if originalCfg != nil {
			config.Save(originalCfg)
		}
	}()

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"config", "set", "default_format", "json"})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config set command failed: %v\nStderr: %s", err, errBuf.String())
	}

	output := buf.String()
	if !strings.Contains(output, "Set default_format = json") {
		t.Errorf("Expected output to contain 'Set default_format = json', got: %s", output)
	}

	// Verify value was set
	value, err := config.GetValue("default_format")
	if err != nil {
		t.Fatalf("Failed to get config value: %v", err)
	}
	if value != "json" {
		t.Errorf("Expected default_format to be 'json', got: %s", value)
	}
}

func TestConfigSet_InvalidKey(t *testing.T) {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"config", "set", "invalid_key", "value"})

	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("config set should fail with invalid key")
	}

	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("Expected error about unknown config key, got: %v", err)
	}
}

func TestConfigSet_InvalidFormatValue(t *testing.T) {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"config", "set", "default_format", "invalid"})

	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("config set should fail with invalid format value")
	}

	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("Expected error about invalid format, got: %v", err)
	}
}

func TestConfigSet_InvalidProjectKeyFormat(t *testing.T) {
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"config", "set", "default_project", "invalid-key"})

	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("config set should fail with invalid project key format")
	}

	if !strings.Contains(err.Error(), "invalid project key") {
		t.Errorf("Expected error about invalid project key, got: %v", err)
	}
}

func TestConfigSet_NonExistentProject(t *testing.T) {
	// Save original config
	originalCfg, _ := config.Get()
	defer func() {
		if originalCfg != nil {
			config.Save(originalCfg)
		}
	}()

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"config", "set", "default_project", "NONEXISTENT"})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config set should succeed even with non-existent project (with warning): %v", err)
	}

	// Check for warning
	warning := errBuf.String()
	if !strings.Contains(warning, "Warning: project") {
		t.Errorf("Expected warning about non-existent project, got: %s", warning)
	}

	// Verify value was still set
	value, err := config.GetValue("default_project")
	if err != nil {
		t.Fatalf("Failed to get config value: %v", err)
	}
	if value != "NONEXISTENT" {
		t.Errorf("Expected default_project to be 'NONEXISTENT', got: %s", value)
	}
}

func TestConfigSet_ValidProject(t *testing.T) {
	// Save original config
	originalCfg, _ := config.Get()
	defer func() {
		if originalCfg != nil {
			config.Save(originalCfg)
		}
	}()

	// Create a test project
	projectKey := sanitizeTestName("TEST" + t.Name())
	defer func() {
		projectDir, _ := storage.ProjectDir(projectKey)
		os.RemoveAll(projectDir)
	}()

	// Create project
	createCmd := NewRootCmd()
	createCmd.SetArgs([]string{"project", "create", projectKey})
	createCmd.SetOut(new(bytes.Buffer))
	if err := createCmd.Execute(); err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Set default_project to the created project
	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"config", "set", "default_project", projectKey})

	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(errBuf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config set command failed: %v\nStderr: %s", err, errBuf.String())
	}

	// Verify value was set
	value, err := config.GetValue("default_project")
	if err != nil {
		t.Fatalf("Failed to get config value: %v", err)
	}
	if value != projectKey {
		t.Errorf("Expected default_project to be %q, got: %s", projectKey, value)
	}
}

func TestConfigList_ModernFormat(t *testing.T) {
	// Save original config
	originalCfg, _ := config.Get()
	defer func() {
		if originalCfg != nil {
			config.Save(originalCfg)
		}
	}()

	// Set some test values
	if err := config.Set("default_format", "json"); err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"config", "list"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config list command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "default_format") {
		t.Errorf("Expected output to contain 'default_format', got: %s", output)
	}
	if !strings.Contains(output, "default_project") {
		t.Errorf("Expected output to contain 'default_project', got: %s", output)
	}
}

func TestConfigList_JSONFormat(t *testing.T) {
	// Save original config
	originalCfg, _ := config.Get()
	defer func() {
		if originalCfg != nil {
			config.Save(originalCfg)
		}
	}()

	// Set some test values
	if err := config.Set("default_format", "json"); err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"config", "list", "--format", "json"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config list command failed: %v", err)
	}

	output := buf.String()
	var result config.Config
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if result.DefaultFormat != "json" {
		t.Errorf("Expected DefaultFormat to be 'json', got: %s", result.DefaultFormat)
	}
}

func TestConfigList_LSONFormat(t *testing.T) {
	// Save original config
	originalCfg, _ := config.Get()
	defer func() {
		if originalCfg != nil {
			config.Save(originalCfg)
		}
	}()

	// Set some test values
	if err := config.Set("default_format", "lson"); err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}
	if err := config.Set("default_project", "TEST"); err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	rootCmd := NewRootCmd()
	rootCmd.SetArgs([]string{"config", "list", "--format", "lson"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("config list command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "@DEFAULT_FORMAT:") {
		t.Errorf("Expected output to contain '@DEFAULT_FORMAT:', got: %s", output)
	}
	if !strings.Contains(output, "@DEFAULT_PROJECT:") {
		t.Errorf("Expected output to contain '@DEFAULT_PROJECT:', got: %s", output)
	}
}
