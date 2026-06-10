package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiscoverPRDsFindsScratchPRDsSortedByModTime(t *testing.T) {
	project := t.TempDir()
	older := writePRD(t, project, "older", "# Older\n\n## Implementation Tasks\n\n- [ ] old\n")
	newer := writePRD(t, project, "newer", "# Newer\n\n## Implementation Tasks\n\n- [x] new\n")
	mustWriteFile(t, filepath.Join(project, ".scratch", "not-prd", "README.md"), "# ignored\n")

	oldTime := time.Now().Add(-2 * time.Hour)
	newTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(older, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newer, newTime, newTime); err != nil {
		t.Fatal(err)
	}

	got, err := discoverPRDs(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d PRDs, want 2", len(got))
	}
	if got[0].Slug != "newer" || got[1].Slug != "older" {
		t.Fatalf("unexpected sort/order: %#v", []string{got[0].Slug, got[1].Slug})
	}
	if got[0].Done != 1 || got[0].Total != 1 {
		t.Fatalf("newer progress = %d/%d, want 1/1", got[0].Done, got[0].Total)
	}
}

func TestParsePRDExtractsHeadingsAndOnlyImplementationTasks(t *testing.T) {
	project := t.TempDir()
	path := writePRD(t, project, "feature", strings.Join([]string{
		"# Feature PRD",
		"",
		"## Notes",
		"- [ ] not an implementation task",
		"",
		"## Implementation Tasks",
		"",
		"- [ ] first task",
		"  - [X] second task",
		"",
		"## Done",
		"- [ ] not counted",
		"",
	}, "\n"))

	doc, err := parsePRD(path)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title != "Feature PRD" {
		t.Fatalf("title = %q", doc.Title)
	}
	if len(doc.Sections) != 4 {
		t.Fatalf("sections = %d, want 4", len(doc.Sections))
	}
	if len(doc.Tasks) != 2 {
		t.Fatalf("tasks = %d, want 2", len(doc.Tasks))
	}
	if doc.Tasks[0].Text != "first task" || doc.Tasks[0].Checked {
		t.Fatalf("first task parsed incorrectly: %#v", doc.Tasks[0])
	}
	if doc.Tasks[1].Text != "second task" || !doc.Tasks[1].Checked {
		t.Fatalf("second task parsed incorrectly: %#v", doc.Tasks[1])
	}
	section := doc.Sections[implementationTasksSection(doc)]
	if section.DoneCount != 1 || section.TaskCount != 2 {
		t.Fatalf("implementation progress = %d/%d, want 1/2", section.DoneCount, section.TaskCount)
	}
}

func TestToggleTaskLinePreservesUnrelatedContentAndCRLF(t *testing.T) {
	project := t.TempDir()
	path := writePRD(t, project, "feature", "# Feature\r\n\r\n## Implementation Tasks\r\n\r\n  - [ ] keep spacing  \r\n\r\nTail\r\n")

	if err := toggleTaskLine(path, 4); err != nil {
		t.Fatal(err)
	}
	gotBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "# Feature\r\n\r\n## Implementation Tasks\r\n\r\n  - [x] keep spacing  \r\n\r\nTail\r\n"
	if string(gotBytes) != want {
		t.Fatalf("file content changed unexpectedly:\n got %q\nwant %q", string(gotBytes), want)
	}
}

func TestToggleTaskLineRejectsInlineCheckbox(t *testing.T) {
	project := t.TempDir()
	path := writePRD(t, project, "feature", "# Feature\n\nThis mentions - [ ] inline but is not a task.\n")

	if err := toggleTaskLine(path, 2); err == nil {
		t.Fatal("expected inline checkbox toggle to fail")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "# Feature\n\nThis mentions - [ ] inline but is not a task.\n" {
		t.Fatal("file changed after failed toggle")
	}
}

func TestGenerateTaskPromptIncludesRequiredContext(t *testing.T) {
	project := t.TempDir()
	path := writePRD(t, project, "feature", strings.Join([]string{
		"# Feature",
		"",
		"## Problem Statement",
		"Problem details.",
		"",
		"## Implementation Decisions",
		"Use Go.",
		"",
		"## Implementation Tasks",
		"- [ ] selected task",
		"- [ ] nearby task",
		"",
		"## Testing Decisions",
		"Run go test.",
		"",
	}, "\n"))
	doc, err := parsePRD(path)
	if err != nil {
		t.Fatal(err)
	}

	prompt := generateTaskPrompt(project, doc, 0)
	for _, want := range []string{"selected task", "Problem context", "Problem details.", "Implementation context", "Use Go.", "Testing expectations", "Run go test.", "nearby task"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func writePRD(t *testing.T, project, slug, content string) string {
	t.Helper()
	path := filepath.Join(project, ".scratch", slug, "PRD.md")
	mustWriteFile(t, path, content)
	return path
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
