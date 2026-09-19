package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
	"github.com/olekukonko/tablewriter"
)

// ModernRenderer renders output in a modern, human-readable format with tables and colors
type ModernRenderer struct {
	styles *Styles
}

// NewModernRenderer creates a new ModernRenderer
func NewModernRenderer() *ModernRenderer {
	return &ModernRenderer{
		styles: NewStyles(),
	}
}

// RenderIssueList renders a list of issues as a table
func (r *ModernRenderer) RenderIssueList(issues []*models.Issue, w io.Writer) error {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"ID", "Title", "Status", "Priority", "Type"})
	table.SetBorder(false)
	table.SetColumnSeparator(" ")
	table.SetRowSeparator("")
	table.SetCenterSeparator("")

	for _, issue := range issues {
		statusColor := r.styles.StatusColor(issue.Status)
		priorityColor := r.styles.PriorityColor(issue.Priority)

		row := []string{
			r.styles.ID(issue.ID),
			issue.Title,
			statusColor(issue.Status),
			priorityColor(issue.Priority),
			issue.Type,
		}
		table.Append(row)
	}

	table.Render()
	return nil
}

// RenderIssue renders a single issue in detail
func (r *ModernRenderer) RenderIssue(issue *models.Issue, w io.Writer) error {
	styles := r.styles

	// Header
	fmt.Fprintf(w, "%s %s\n\n", styles.ID(issue.ID), styles.Title(issue.Title))

	// Metadata
	fmt.Fprintf(w, "%s: %s\n", styles.Label("Status"), styles.StatusColor(issue.Status)(issue.Status))
	if issue.Priority != "" {
		fmt.Fprintf(w, "%s: %s\n", styles.Label("Priority"), styles.PriorityColor(issue.Priority)(issue.Priority))
	}
	if issue.Type != "" {
		fmt.Fprintf(w, "%s: %s\n", styles.Label("Type"), issue.Type)
	}
	if issue.EpicID != "" {
		fmt.Fprintf(w, "%s: %s\n", styles.Label("Epic"), issue.EpicID)
	}
	fmt.Fprintf(w, "\n")

	// Description
	if issue.Description != "" {
		fmt.Fprintf(w, "%s\n", styles.Label("Description"))
		rendered, err := RenderMarkdown(issue.Description)
		if err != nil {
			return fmt.Errorf("ui: failed to render markdown: %w", err)
		}
		fmt.Fprintf(w, "%s\n\n", rendered)
	}

	// Dependencies
	if len(issue.BlockedBy) > 0 {
		fmt.Fprintf(w, "%s: %s\n", styles.Label("Blocked By"), strings.Join(issue.BlockedBy, ", "))
	}

	// PRs
	if len(issue.PRs) > 0 {
		fmt.Fprintf(w, "%s:\n", styles.Label("Pull Requests"))
		for _, pr := range issue.PRs {
			fmt.Fprintf(w, "  - %s\n", pr)
		}
	}

	return nil
}

// RenderEpic renders an epic in detail
func (r *ModernRenderer) RenderEpic(epic *models.Epic, w io.Writer) error {
	styles := r.styles

	// Header
	fmt.Fprintf(w, "%s %s\n\n", styles.ID(epic.ID), styles.Title(epic.Title))

	// Status
	if epic.Status != "" {
		fmt.Fprintf(w, "%s: %s\n", styles.Label("Status"), styles.StatusColor(epic.Status)(epic.Status))
	}
	fmt.Fprintf(w, "\n")

	// Description
	if epic.Description != "" {
		fmt.Fprintf(w, "%s\n", styles.Label("Description"))
		rendered, err := RenderMarkdown(epic.Description)
		if err != nil {
			return fmt.Errorf("ui: failed to render markdown: %w", err)
		}
		fmt.Fprintf(w, "%s\n\n", rendered)
	}

	return nil
}

// RenderProjectIndex renders a project index
func (r *ModernRenderer) RenderProjectIndex(index *models.ProjectIndex, w io.Writer) error {
	styles := r.styles

	// Header
	fmt.Fprintf(w, "%s", styles.Title(index.ProjectKey))
	if index.ProjectName != "" {
		fmt.Fprintf(w, " - %s", index.ProjectName)
	}
	fmt.Fprintf(w, "\n\n")

	// Convert index entries to issues for table rendering
	if len(index.Issues) > 0 {
		table := tablewriter.NewWriter(w)
		table.SetHeader([]string{"ID", "Title", "Status", "Type"})
		table.SetBorder(false)
		table.SetColumnSeparator(" ")
		table.SetRowSeparator("")
		table.SetCenterSeparator("")

		for _, entry := range index.Issues {
			statusColor := r.styles.StatusColor(entry.Status)

			row := []string{
				r.styles.ID(entry.ID),
				entry.Title,
				statusColor(entry.Status),
				entry.Type,
			}
			table.Append(row)
		}

		table.Render()
	} else {
		fmt.Fprintf(w, "No issues found.\n")
	}

	return nil
}
