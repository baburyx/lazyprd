package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestViewFitsWindowGeometry(t *testing.T) {
	project := t.TempDir()
	path := writePRD(t, project, "feature", strings.Join([]string{
		"# Feature",
		"",
		"## Problem Statement",
		"This is a long problem statement that should wrap inside the preview pane instead of making the terminal scroll or hiding the top border.",
		"",
		"## Implementation Tasks",
		"- [ ] First task that has a long title and needs to remain visible inside the list pane",
		"- [ ] Second task",
		"",
	}, "\n"))
	doc, err := parsePRD(path)
	if err != nil {
		t.Fatal(err)
	}
	m := model{
		project: project,
		prds: []PRDSummary{{
			Path:     path,
			Slug:     doc.Slug,
			Title:    doc.Title,
			Done:     0,
			Total:    len(doc.Tasks),
			Document: doc,
		}},
		focus:  focusMiddle,
		status: "loaded",
	}
	m.selectDefaultMiddle()

	for _, size := range []struct{ width, height int }{{130, 40}, {90, 30}, {60, 22}} {
		m.width = size.width
		m.height = size.height
		m.ensureVisible()
		view := m.View()
		if got := lipgloss.Height(view); got != size.height {
			t.Fatalf("view height at %dx%d = %d, want %d\n%s", size.width, size.height, got, size.height, view)
		}
		for i, line := range strings.Split(view, "\n") {
			if got := lipgloss.Width(line); got > size.width {
				t.Fatalf("line %d width at %dx%d = %d, want <= %d: %q", i+1, size.width, size.height, got, size.width, line)
			}
		}
	}
}
