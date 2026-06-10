package main

import (
	"fmt"
	"strings"
)

func generateTaskPrompt(project string, doc *PRD, taskIndex int) string {
	if doc == nil || taskIndex < 0 || taskIndex >= len(doc.Tasks) {
		return ""
	}
	task := doc.Tasks[taskIndex]
	var b strings.Builder
	fmt.Fprintf(&b, "Implement this PRD task in the project at `%s`.\n\n", project)
	fmt.Fprintf(&b, "PRD: %s\n", doc.Title)
	fmt.Fprintf(&b, "PRD file: `%s`\n\n", doc.Path)
	fmt.Fprintf(&b, "Selected task:\n- [ ] %s\n\n", task.Text)

	if summary := firstSectionSnippet(doc, "problem statement", 900); summary != "" {
		fmt.Fprintf(&b, "Problem context:\n%s\n\n", summary)
	}

	fmt.Fprintf(&b, "Nearby task context:\n%s\n\n", nearbyTasks(doc, taskIndex, 3))

	if impl := firstSectionSnippet(doc, "implementation decisions", 1600); impl != "" {
		fmt.Fprintf(&b, "Implementation context:\n%s\n\n", impl)
	}
	if tests := firstSectionSnippet(doc, "testing decisions", 1600); tests != "" {
		fmt.Fprintf(&b, "Testing expectations:\n%s\n\n", tests)
	}

	b.WriteString("Instructions:\n")
	b.WriteString("- Implement only the selected task unless a direct dependency makes that impossible.\n")
	b.WriteString("- Prefer the smallest correct change.\n")
	b.WriteString("- Preserve unrelated user changes.\n")
	b.WriteString("- Run focused verification and report what passed or why it could not be run.\n")
	return b.String()
}

func firstSectionSnippet(doc *PRD, heading string, max int) string {
	if doc == nil {
		return ""
	}
	for i, section := range doc.Sections {
		if normalizeHeading(section.Title) != heading {
			continue
		}
		text := sectionText(doc, i)
		if len(text) <= max {
			return text
		}
		return strings.TrimSpace(text[:max]) + "..."
	}
	return ""
}

func nearbyTasks(doc *PRD, taskIndex int, radius int) string {
	start := taskIndex - radius
	if start < 0 {
		start = 0
	}
	end := taskIndex + radius + 1
	if end > len(doc.Tasks) {
		end = len(doc.Tasks)
	}
	var b strings.Builder
	for i := start; i < end; i++ {
		box := "[ ]"
		if doc.Tasks[i].Checked {
			box = "[x]"
		}
		marker := ""
		if i == taskIndex {
			marker = " <-- selected"
		}
		fmt.Fprintf(&b, "- %s %s%s\n", box, doc.Tasks[i].Text, marker)
	}
	return strings.TrimRight(b.String(), "\n")
}
