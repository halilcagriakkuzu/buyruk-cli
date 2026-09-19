package ui

import (
	"encoding/json"
	"io"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
)

// JSONRenderer renders output in JSON format
type JSONRenderer struct{}

// NewJSONRenderer creates a new JSONRenderer
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

// RenderIssue renders a single issue as JSON
func (r *JSONRenderer) RenderIssue(issue *models.Issue, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(issue)
}

// RenderIssueList renders a list of issues as JSON
func (r *JSONRenderer) RenderIssueList(issues []*models.Issue, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(issues)
}

// RenderEpic renders an epic as JSON
func (r *JSONRenderer) RenderEpic(epic *models.Epic, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(epic)
}

// RenderProjectIndex renders a project index as JSON
func (r *JSONRenderer) RenderProjectIndex(index *models.ProjectIndex, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(index)
}
