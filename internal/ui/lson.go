package ui

import (
	"fmt"
	"io"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/models"
)

// LSONRenderer renders output in L-SON format (token-optimized for LLMs)
type LSONRenderer struct{}

// NewLSONRenderer creates a new LSONRenderer
func NewLSONRenderer() *LSONRenderer {
	return &LSONRenderer{}
}

// RenderIssue renders a single issue in L-SON format
func (r *LSONRenderer) RenderIssue(issue *models.Issue, w io.Writer) error {
	fmt.Fprintf(w, "@ID: %s\n", issue.ID)
	fmt.Fprintf(w, "@TYPE: %s\n", issue.Type)
	fmt.Fprintf(w, "@STATUS: %s\n", issue.Status)

	if issue.Priority != "" {
		fmt.Fprintf(w, "@PRIORITY: %s\n", issue.Priority)
	}

	fmt.Fprintf(w, "@TITLE: %s\n", issue.Title)

	if issue.EpicID != "" {
		fmt.Fprintf(w, "@EPIC: %s\n", issue.EpicID)
	}

	if len(issue.BlockedBy) > 0 {
		for _, dep := range issue.BlockedBy {
			fmt.Fprintf(w, "@DEP: %s\n", dep)
		}
	}

	if len(issue.PRs) > 0 {
		for _, pr := range issue.PRs {
			fmt.Fprintf(w, "@PR: %s\n", pr)
		}
	}

	if issue.Description != "" {
		fmt.Fprintf(w, "@DESC: %s\n", issue.Description)
	}

	return nil
}

// RenderIssueList renders a list of issues in L-SON format
func (r *LSONRenderer) RenderIssueList(issues []*models.Issue, w io.Writer) error {
	for i, issue := range issues {
		if i > 0 {
			fmt.Fprintf(w, "\n")
		}
		fmt.Fprintf(w, "@ID: %s\n", issue.ID)
		fmt.Fprintf(w, "@TITLE: %s\n", issue.Title)
		fmt.Fprintf(w, "@STATUS: %s\n", issue.Status)
		if issue.Priority != "" {
			fmt.Fprintf(w, "@PRIORITY: %s\n", issue.Priority)
		}
		if issue.Type != "" {
			fmt.Fprintf(w, "@TYPE: %s\n", issue.Type)
		}
	}
	return nil
}

// RenderEpic renders an epic in L-SON format
func (r *LSONRenderer) RenderEpic(epic *models.Epic, w io.Writer) error {
	fmt.Fprintf(w, "@ID: %s\n", epic.ID)
	fmt.Fprintf(w, "@TITLE: %s\n", epic.Title)
	if epic.Status != "" {
		fmt.Fprintf(w, "@STATUS: %s\n", epic.Status)
	}
	if epic.Description != "" {
		fmt.Fprintf(w, "@DESC: %s\n", epic.Description)
	}
	return nil
}

// RenderProjectIndex renders a project index in L-SON format
func (r *LSONRenderer) RenderProjectIndex(index *models.ProjectIndex, w io.Writer) error {
	fmt.Fprintf(w, "@PROJECT: %s\n", index.ProjectKey)
	if index.ProjectName != "" {
		fmt.Fprintf(w, "@NAME: %s\n", index.ProjectName)
	}
	for _, entry := range index.Issues {
		fmt.Fprintf(w, "@ISSUE: %s | %s | %s | %s\n", entry.ID, entry.Title, entry.Status, entry.Type)
	}
	return nil
}
