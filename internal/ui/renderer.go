package ui

import (
	"fmt"
	"io"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/config"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/spf13/cobra"
)

// Renderer defines the interface for rendering different data types
type Renderer interface {
	RenderIssue(issue *models.Issue, w io.Writer) error
	RenderIssueList(issues []*models.Issue, w io.Writer) error
	RenderEpic(epic *models.Epic, w io.Writer) error
	RenderProjectIndex(index *models.ProjectIndex, w io.Writer) error
}

// NewRenderer creates a new renderer based on the format string
func NewRenderer(format string) (Renderer, error) {
	switch format {
	case "modern":
		return NewModernRenderer(), nil
	case "json":
		return NewJSONRenderer(), nil
	case "lson":
		return NewLSONRenderer(), nil
	default:
		return nil, fmt.Errorf("ui: unknown format %q", format)
	}
}

// GetRenderer gets a renderer from a cobra command, resolving format from flag > config > default
func GetRenderer(cmd *cobra.Command) (Renderer, error) {
	format := config.ResolveFormat(cmd)
	return NewRenderer(format)
}
