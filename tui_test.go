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

func TestCompactLayoutShowsFocusedPRDPicker(t *testing.T) {
	m := testModelWithTwoPRDs(t)

	m.focus = focusPRDs
	m.width = 60
	m.height = 22
	m.ensureVisible()
	view := m.View()
	if !strings.Contains(view, "PRDs 2") {
		t.Fatalf("narrow PRD focus should show PRD picker:\n%s", view)
	}
	if strings.Contains(view, "Task Preview") {
		t.Fatalf("narrow PRD focus should not show preview pane:\n%s", view)
	}

	m.width = 90
	m.height = 24
	m.ensureVisible()
	view = m.View()
	if !strings.Contains(view, "PRDs 2") || !strings.Contains(view, "Outline / Tasks") {
		t.Fatalf("medium PRD focus should show PRD picker and tasks:\n%s", view)
	}
}

func TestPreviewVimMotionsAndSearch(t *testing.T) {
	m := testModelWithTwoPRDs(t)
	m.focus = focusPreview
	m.width = 90
	m.height = 24
	m.ensureVisible()

	m.movePreviewBlock(1)
	if m.previewTop == 0 {
		t.Fatal("expected } to move preview down to next block")
	}

	m.movePreviewBlock(-1)
	if m.previewTop != 0 {
		t.Fatalf("expected { to move preview back to top, got %d", m.previewTop)
	}

	m.filter = "nearby"
	m.jumpToPreviewSearch()
	if m.previewTop == 0 {
		t.Fatal("expected preview search to jump to matching content")
	}
	if m.status != "preview match" {
		t.Fatalf("status = %q, want preview match", m.status)
	}
}

func testModelWithTwoPRDs(t *testing.T) model {
	t.Helper()
	project := t.TempDir()
	path1 := writePRD(t, project, "one", "# One\n\n## Implementation Tasks\n\n- [ ] first\n")
	path2 := writePRD(t, project, "two", "# Two\n\n## Implementation Tasks\n\n- [ ] second\n")
	doc1, err := parsePRD(path1)
	if err != nil {
		t.Fatal(err)
	}
	doc2, err := parsePRD(path2)
	if err != nil {
		t.Fatal(err)
	}
	m := model{
		project: project,
		prds: []PRDSummary{
			{Path: path1, Slug: doc1.Slug, Title: doc1.Title, Total: len(doc1.Tasks), Document: doc1},
			{Path: path2, Slug: doc2.Slug, Title: doc2.Title, Total: len(doc2.Tasks), Document: doc2},
		},
		focus:  focusMiddle,
		status: "loaded",
	}
	m.selectDefaultMiddle()
	return m
}
