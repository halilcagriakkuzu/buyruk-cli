package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
)

// Styles provides styling utilities for rendering
type Styles struct{}

// NewStyles creates a new Styles instance
func NewStyles() *Styles {
	return &Styles{}
}

// ID styles an issue ID
func (s *Styles) ID(id string) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("6"))
	return style.Render(id)
}

// Title styles a title
func (s *Styles) Title(title string) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("7"))
	return style.Render(title)
}

// Label styles a label
func (s *Styles) Label(label string) string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("4"))
	return style.Render(label)
}

// StatusColor returns a function that styles text with the appropriate color for a status
func (s *Styles) StatusColor(status string) func(string) string {
	colors := map[string]lipgloss.Color{
		models.StatusTODO:  lipgloss.Color("3"), // Yellow
		models.StatusDOING: lipgloss.Color("6"), // Cyan
		models.StatusDONE:  lipgloss.Color("2"), // Green
	}

	color := colors[status]
	if color == "" {
		color = lipgloss.Color("7") // Default white
	}

	return func(text string) string {
		return lipgloss.NewStyle().Foreground(color).Render(text)
	}
}

// PriorityColor returns a function that styles text with the appropriate color for a priority
func (s *Styles) PriorityColor(priority string) func(string) string {
	colors := map[string]lipgloss.Color{
		models.PriorityLOW:      lipgloss.Color("2"), // Green
		models.PriorityMEDIUM:   lipgloss.Color("3"), // Yellow
		models.PriorityHIGH:     lipgloss.Color("1"), // Red
		models.PriorityCRITICAL: lipgloss.Color("1"), // Red
	}

	color := colors[priority]
	if color == "" {
		color = lipgloss.Color("7") // Default white
	}

	style := lipgloss.NewStyle().Foreground(color)
	if priority == models.PriorityCRITICAL {
		style = style.Bold(true)
	}

	return func(text string) string {
		return style.Render(text)
	}
}

// Error styles error text
func (s *Styles) Error(text string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("1")).
		Bold(true)
	return style.Render(text)
}

// Success styles success text
func (s *Styles) Success(text string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("2")).
		Bold(true)
	return style.Render(text)
}

func init() {
	// Respect NO_COLOR environment variable
	// lipgloss automatically handles NO_COLOR, but we ensure it's checked
	if os.Getenv("NO_COLOR") != "" {
		// lipgloss will automatically disable colors when NO_COLOR is set
		// This is handled by the library itself
	}
}
