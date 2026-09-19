package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/spf13/cobra"
)

// TestNewRenderer tests format selection
func TestNewRenderer(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{"modern format", "modern", false},
		{"json format", "json", false},
		{"lson format", "lson", false},
		{"invalid format", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer, err := NewRenderer(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRenderer(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
				return
			}
			if !tt.wantErr && renderer == nil {
				t.Errorf("NewRenderer(%q) returned nil renderer", tt.format)
			}
		})
	}
}

// TestGetRenderer tests getting renderer from command
func TestGetRenderer(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("format", "", "Output format")

	// Test with flag set
	cmd.Flags().Set("format", "json")
	renderer, err := GetRenderer(cmd)
	if err != nil {
		t.Fatalf("GetRenderer() failed: %v", err)
	}
	if renderer == nil {
		t.Fatal("GetRenderer() returned nil renderer")
	}

	// Verify it's a JSON renderer
	if _, ok := renderer.(*JSONRenderer); !ok {
		t.Error("GetRenderer() did not return JSONRenderer when format=json")
	}
}

// TestModernRenderer_RenderIssueList tests modern format issue list rendering
func TestModernRenderer_RenderIssueList(t *testing.T) {
	renderer := NewModernRenderer()
	issues := []*models.Issue{
		{
			ID:       "CORE-1",
			Title:    "Test Issue 1",
			Status:   models.StatusTODO,
			Priority: models.PriorityHIGH,
			Type:     models.TypeTask,
		},
		{
			ID:       "CORE-2",
			Title:    "Test Issue 2",
			Status:   models.StatusDONE,
			Priority: models.PriorityLOW,
			Type:     models.TypeBug,
		},
	}

	var buf bytes.Buffer
	err := renderer.RenderIssueList(issues, &buf)
	if err != nil {
		t.Fatalf("RenderIssueList() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "CORE-1") {
		t.Error("RenderIssueList() output missing issue ID")
	}
	if !strings.Contains(output, "Test Issue 1") {
		t.Error("RenderIssueList() output missing issue title")
	}
}

// TestModernRenderer_RenderIssue tests modern format issue detail rendering
func TestModernRenderer_RenderIssue(t *testing.T) {
	renderer := NewModernRenderer()
	issue := &models.Issue{
		ID:          "CORE-12",
		Title:       "Test Issue",
		Status:      models.StatusDOING,
		Priority:    models.PriorityMEDIUM,
		Type:        models.TypeTask,
		Description: "This is a test description",
		BlockedBy:   []string{"CORE-10"},
		PRs:         []string{"https://github.com/example/pr/1"},
	}

	var buf bytes.Buffer
	err := renderer.RenderIssue(issue, &buf)
	if err != nil {
		t.Fatalf("RenderIssue() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "CORE-12") {
		t.Error("RenderIssue() output missing issue ID")
	}
	if !strings.Contains(output, "Test Issue") {
		t.Error("RenderIssue() output missing issue title")
	}
	if !strings.Contains(output, "DOING") {
		t.Error("RenderIssue() output missing status")
	}
}

// TestModernRenderer_RenderIssue_EmptyFields tests rendering issue with empty optional fields
func TestModernRenderer_RenderIssue_EmptyFields(t *testing.T) {
	renderer := NewModernRenderer()
	issue := &models.Issue{
		ID:     "CORE-1",
		Title:  "Minimal Issue",
		Status: models.StatusTODO,
		Type:   models.TypeTask,
	}

	var buf bytes.Buffer
	err := renderer.RenderIssue(issue, &buf)
	if err != nil {
		t.Fatalf("RenderIssue() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "CORE-1") {
		t.Error("RenderIssue() output missing issue ID")
	}
}

// TestModernRenderer_RenderEpic tests modern format epic rendering
func TestModernRenderer_RenderEpic(t *testing.T) {
	renderer := NewModernRenderer()
	epic := &models.Epic{
		ID:          "E-1",
		Title:       "Test Epic",
		Status:      models.StatusDOING,
		Description: "Epic description",
	}

	var buf bytes.Buffer
	err := renderer.RenderEpic(epic, &buf)
	if err != nil {
		t.Fatalf("RenderEpic() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "E-1") {
		t.Error("RenderEpic() output missing epic ID")
	}
	if !strings.Contains(output, "Test Epic") {
		t.Error("RenderEpic() output missing epic title")
	}
}

// TestModernRenderer_RenderProjectIndex tests modern format project index rendering
func TestModernRenderer_RenderProjectIndex(t *testing.T) {
	renderer := NewModernRenderer()
	index := &models.ProjectIndex{
		ProjectKey:  "CORE",
		ProjectName: "Core Project",
		Issues: []models.IndexEntry{
			{
				ID:     "CORE-1",
				Title:  "Issue 1",
				Status: models.StatusTODO,
				Type:   models.TypeTask,
			},
		},
	}

	var buf bytes.Buffer
	err := renderer.RenderProjectIndex(index, &buf)
	if err != nil {
		t.Fatalf("RenderProjectIndex() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "CORE") {
		t.Error("RenderProjectIndex() output missing project key")
	}
}

// TestModernRenderer_RenderProjectIndex_Empty tests rendering empty project index
func TestModernRenderer_RenderProjectIndex_Empty(t *testing.T) {
	renderer := NewModernRenderer()
	index := &models.ProjectIndex{
		ProjectKey: "CORE",
		Issues:     []models.IndexEntry{},
	}

	var buf bytes.Buffer
	err := renderer.RenderProjectIndex(index, &buf)
	if err != nil {
		t.Fatalf("RenderProjectIndex() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "CORE") {
		t.Error("RenderProjectIndex() output missing project key")
	}
	if !strings.Contains(output, "No issues") {
		t.Error("RenderProjectIndex() should show message for empty index")
	}
}

// TestJSONRenderer_RenderIssue tests JSON format issue rendering
func TestJSONRenderer_RenderIssue(t *testing.T) {
	renderer := NewJSONRenderer()
	issue := &models.Issue{
		ID:     "CORE-1",
		Title:  "Test Issue",
		Status: models.StatusTODO,
		Type:   models.TypeTask,
	}

	var buf bytes.Buffer
	err := renderer.RenderIssue(issue, &buf)
	if err != nil {
		t.Fatalf("RenderIssue() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "CORE-1") {
		t.Error("RenderIssue() JSON output missing issue ID")
	}
	if !strings.Contains(output, "Test Issue") {
		t.Error("RenderIssue() JSON output missing issue title")
	}
	if !strings.Contains(output, `"id"`) {
		t.Error("RenderIssue() JSON output missing JSON structure")
	}
}

// TestJSONRenderer_RenderIssueList tests JSON format issue list rendering
func TestJSONRenderer_RenderIssueList(t *testing.T) {
	renderer := NewJSONRenderer()
	issues := []*models.Issue{
		{
			ID:     "CORE-1",
			Title:  "Issue 1",
			Status: models.StatusTODO,
			Type:   models.TypeTask,
		},
		{
			ID:     "CORE-2",
			Title:  "Issue 2",
			Status: models.StatusDONE,
			Type:   models.TypeBug,
		},
	}

	var buf bytes.Buffer
	err := renderer.RenderIssueList(issues, &buf)
	if err != nil {
		t.Fatalf("RenderIssueList() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[") {
		t.Error("RenderIssueList() JSON output missing array bracket")
	}
	if !strings.Contains(output, "CORE-1") {
		t.Error("RenderIssueList() JSON output missing first issue")
	}
}

// TestJSONRenderer_RenderEpic tests JSON format epic rendering
func TestJSONRenderer_RenderEpic(t *testing.T) {
	renderer := NewJSONRenderer()
	epic := &models.Epic{
		ID:    "E-1",
		Title: "Test Epic",
	}

	var buf bytes.Buffer
	err := renderer.RenderEpic(epic, &buf)
	if err != nil {
		t.Fatalf("RenderEpic() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "E-1") {
		t.Error("RenderEpic() JSON output missing epic ID")
	}
}

// TestJSONRenderer_RenderProjectIndex tests JSON format project index rendering
func TestJSONRenderer_RenderProjectIndex(t *testing.T) {
	renderer := NewJSONRenderer()
	index := &models.ProjectIndex{
		ProjectKey: "CORE",
		Issues:     []models.IndexEntry{},
	}

	var buf bytes.Buffer
	err := renderer.RenderProjectIndex(index, &buf)
	if err != nil {
		t.Fatalf("RenderProjectIndex() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "CORE") {
		t.Error("RenderProjectIndex() JSON output missing project key")
	}
}

// TestLSONRenderer_RenderIssue tests L-SON format issue rendering
func TestLSONRenderer_RenderIssue(t *testing.T) {
	renderer := NewLSONRenderer()
	issue := &models.Issue{
		ID:       "CORE-1",
		Title:    "Test Issue",
		Status:   models.StatusTODO,
		Priority: models.PriorityHIGH,
		Type:     models.TypeTask,
	}

	var buf bytes.Buffer
	err := renderer.RenderIssue(issue, &buf)
	if err != nil {
		t.Fatalf("RenderIssue() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "@ID: CORE-1") {
		t.Error("RenderIssue() L-SON output missing @ID")
	}
	if !strings.Contains(output, "@TITLE: Test Issue") {
		t.Error("RenderIssue() L-SON output missing @TITLE")
	}
	if !strings.Contains(output, "@STATUS: TODO") {
		t.Error("RenderIssue() L-SON output missing @STATUS")
	}
}

// TestLSONRenderer_RenderIssueList tests L-SON format issue list rendering
func TestLSONRenderer_RenderIssueList(t *testing.T) {
	renderer := NewLSONRenderer()
	issues := []*models.Issue{
		{
			ID:     "CORE-1",
			Title:  "Issue 1",
			Status: models.StatusTODO,
			Type:   models.TypeTask,
		},
		{
			ID:     "CORE-2",
			Title:  "Issue 2",
			Status: models.StatusDONE,
			Type:   models.TypeBug,
		},
	}

	var buf bytes.Buffer
	err := renderer.RenderIssueList(issues, &buf)
	if err != nil {
		t.Fatalf("RenderIssueList() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "@ID: CORE-1") {
		t.Error("RenderIssueList() L-SON output missing first issue")
	}
	if !strings.Contains(output, "@ID: CORE-2") {
		t.Error("RenderIssueList() L-SON output missing second issue")
	}
}

// TestLSONRenderer_RenderEpic tests L-SON format epic rendering
func TestLSONRenderer_RenderEpic(t *testing.T) {
	renderer := NewLSONRenderer()
	epic := &models.Epic{
		ID:    "E-1",
		Title: "Test Epic",
	}

	var buf bytes.Buffer
	err := renderer.RenderEpic(epic, &buf)
	if err != nil {
		t.Fatalf("RenderEpic() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "@ID: E-1") {
		t.Error("RenderEpic() L-SON output missing @ID")
	}
}

// TestLSONRenderer_RenderProjectIndex tests L-SON format project index rendering
func TestLSONRenderer_RenderProjectIndex(t *testing.T) {
	renderer := NewLSONRenderer()
	index := &models.ProjectIndex{
		ProjectKey: "CORE",
		Issues: []models.IndexEntry{
			{
				ID:     "CORE-1",
				Title:  "Issue 1",
				Status: models.StatusTODO,
				Type:   models.TypeTask,
			},
		},
	}

	var buf bytes.Buffer
	err := renderer.RenderProjectIndex(index, &buf)
	if err != nil {
		t.Fatalf("RenderProjectIndex() failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "@PROJECT: CORE") {
		t.Error("RenderProjectIndex() L-SON output missing @PROJECT")
	}
}

// TestRenderMarkdown tests markdown rendering
func TestRenderMarkdown(t *testing.T) {
	text := "# Heading\n\nThis is a paragraph."
	rendered, err := RenderMarkdown(text)
	if err != nil {
		t.Fatalf("RenderMarkdown() failed: %v", err)
	}

	if rendered == "" {
		t.Error("RenderMarkdown() returned empty string")
	}
}

// TestGetTerminalWidth tests terminal width detection
func TestGetTerminalWidth(t *testing.T) {
	// Save original env var
	originalWidth := os.Getenv("BUYRUK_TERM_WIDTH")
	defer func() {
		if originalWidth != "" {
			os.Setenv("BUYRUK_TERM_WIDTH", originalWidth)
		} else {
			os.Unsetenv("BUYRUK_TERM_WIDTH")
		}
	}()

	// Test environment variable override
	os.Setenv("BUYRUK_TERM_WIDTH", "120")
	width := getTerminalWidth()
	if width != 120 {
		t.Errorf("getTerminalWidth() with env var = %d, want 120", width)
	}

	// Test clamping minimum
	os.Setenv("BUYRUK_TERM_WIDTH", "20")
	width = getTerminalWidth()
	if width != 40 {
		t.Errorf("getTerminalWidth() with small env var = %d, want 40 (clamped)", width)
	}

	// Test clamping maximum
	os.Setenv("BUYRUK_TERM_WIDTH", "300")
	width = getTerminalWidth()
	if width != 200 {
		t.Errorf("getTerminalWidth() with large env var = %d, want 200 (clamped)", width)
	}

	// Test invalid env var falls back to detection/default
	os.Setenv("BUYRUK_TERM_WIDTH", "invalid")
	width = getTerminalWidth()
	// Should get either detected width or default 80
	if width < 40 || width > 200 {
		t.Errorf("getTerminalWidth() with invalid env var = %d, want between 40-200", width)
	}

	// Test unset env var uses detection/default
	os.Unsetenv("BUYRUK_TERM_WIDTH")
	width = getTerminalWidth()
	// Should get either detected width or default 80
	if width < 40 || width > 200 {
		t.Errorf("getTerminalWidth() without env var = %d, want between 40-200", width)
	}
}

// TestRenderMarkdownToWriter tests markdown rendering to writer
func TestRenderMarkdownToWriter(t *testing.T) {
	text := "# Heading\n\nThis is a paragraph."
	var buf bytes.Buffer
	err := RenderMarkdownToWriter(text, &buf)
	if err != nil {
		t.Fatalf("RenderMarkdownToWriter() failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("RenderMarkdownToWriter() wrote nothing")
	}
}

// TestStyles_ID tests ID styling
func TestStyles_ID(t *testing.T) {
	styles := NewStyles()
	result := styles.ID("CORE-1")
	if result == "" {
		t.Error("Styles.ID() returned empty string")
	}
	// Note: In non-terminal environments or when NO_COLOR is set,
	// lipgloss may return the original string, which is acceptable behavior
	_ = result // Result is valid even if it equals input in some environments
}

// TestStyles_Title tests title styling
func TestStyles_Title(t *testing.T) {
	styles := NewStyles()
	result := styles.Title("Test Title")
	if result == "" {
		t.Error("Styles.Title() returned empty string")
	}
}

// TestStyles_Label tests label styling
func TestStyles_Label(t *testing.T) {
	styles := NewStyles()
	result := styles.Label("Status")
	if result == "" {
		t.Error("Styles.Label() returned empty string")
	}
}

// TestStyles_StatusColor tests status color function
func TestStyles_StatusColor(t *testing.T) {
	styles := NewStyles()

	tests := []struct {
		status string
	}{
		{models.StatusTODO},
		{models.StatusDOING},
		{models.StatusDONE},
		{"UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			colorFunc := styles.StatusColor(tt.status)
			result := colorFunc(tt.status)
			if result == "" {
				t.Errorf("StatusColor(%q) returned empty string", tt.status)
			}
		})
	}
}

// TestStyles_PriorityColor tests priority color function
func TestStyles_PriorityColor(t *testing.T) {
	styles := NewStyles()

	tests := []struct {
		priority string
	}{
		{models.PriorityLOW},
		{models.PriorityMEDIUM},
		{models.PriorityHIGH},
		{models.PriorityCRITICAL},
		{"UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			colorFunc := styles.PriorityColor(tt.priority)
			result := colorFunc(tt.priority)
			if result == "" {
				t.Errorf("PriorityColor(%q) returned empty string", tt.priority)
			}
		})
	}
}

// TestStyles_Error tests error styling
func TestStyles_Error(t *testing.T) {
	styles := NewStyles()
	result := styles.Error("Error message")
	if result == "" {
		t.Error("Styles.Error() returned empty string")
	}
}

// TestStyles_Success tests success styling
func TestStyles_Success(t *testing.T) {
	styles := NewStyles()
	result := styles.Success("Success message")
	if result == "" {
		t.Error("Styles.Success() returned empty string")
	}
}

// TestNO_COLOR tests NO_COLOR environment variable handling
func TestNO_COLOR(t *testing.T) {
	// Save original value
	original := os.Getenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", original)

	// Set NO_COLOR
	os.Setenv("NO_COLOR", "1")

	// Styles should still work (lipgloss handles NO_COLOR automatically)
	styles := NewStyles()
	result := styles.ID("TEST")
	if result == "" {
		t.Error("Styles should work even with NO_COLOR set")
	}

	// Clear NO_COLOR
	os.Unsetenv("NO_COLOR")
}

// TestModernRenderer_RenderIssueList_Empty tests rendering empty issue list
func TestModernRenderer_RenderIssueList_Empty(t *testing.T) {
	renderer := NewModernRenderer()
	issues := []*models.Issue{}

	var buf bytes.Buffer
	err := renderer.RenderIssueList(issues, &buf)
	if err != nil {
		t.Fatalf("RenderIssueList() failed with empty list: %v", err)
	}

	// Should render table header even with empty list
	output := buf.String()
	if !strings.Contains(output, "ID") {
		t.Error("RenderIssueList() should render header even with empty list")
	}
}

// TestLSONRenderer_RenderIssueList_Empty tests L-SON rendering of empty list
func TestLSONRenderer_RenderIssueList_Empty(t *testing.T) {
	renderer := NewLSONRenderer()
	issues := []*models.Issue{}

	var buf bytes.Buffer
	err := renderer.RenderIssueList(issues, &buf)
	if err != nil {
		t.Fatalf("RenderIssueList() failed with empty list: %v", err)
	}

	// Empty list should produce empty output (or minimal output)
	output := buf.String()
	if len(output) > 0 && !strings.Contains(output, "@ID") {
		t.Error("RenderIssueList() with empty list should produce minimal output")
	}
}
