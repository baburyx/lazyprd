package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type PRDSummary struct {
	Path     string
	Slug     string
	Title    string
	Done     int
	Total    int
	ModTime  time.Time
	Document *PRD
}

type PRD struct {
	Path     string
	Slug     string
	Title    string
	Lines    []string
	Sections []Section
	Tasks    []Task
}

type Section struct {
	Level     int
	Title     string
	Line      int
	EndLine   int
	TaskCount int
	DoneCount int
}

type Task struct {
	Text    string
	Line    int
	Checked bool
}

type MiddleItem struct {
	Kind         string
	Label        string
	SectionIndex int
	TaskIndex    int
	Line         int
}

func discoverPRDs(project string) ([]PRDSummary, error) {
	pattern := filepath.Join(project, ".scratch", "*", "PRD.md")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	summaries := make([]PRDSummary, 0, len(paths))
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		doc, err := parsePRD(path)
		if err != nil {
			summaries = append(summaries, PRDSummary{Path: path, Slug: slugFromPath(path), Title: slugFromPath(path), ModTime: info.ModTime()})
			continue
		}
		done, total := taskProgress(doc.Tasks)
		summaries = append(summaries, PRDSummary{Path: path, Slug: doc.Slug, Title: doc.Title, Done: done, Total: total, ModTime: info.ModTime(), Document: doc})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].ModTime.After(summaries[j].ModTime)
	})
	return summaries, nil
}

func prdProjectSignature(project string) (string, error) {
	pattern := filepath.Join(project, ".scratch", "*", "PRD.md")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	sort.Strings(paths)

	var b strings.Builder
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "%s\t%d\t%d\n", path, info.Size(), info.ModTime().UnixNano())
	}
	return b.String(), nil
}

func parsePRD(path string) (*PRD, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	doc := &PRD{Path: path, Slug: slugFromPath(path), Lines: lines, Title: slugFromPath(path)}
	for i, line := range lines {
		level, title, ok := parseHeading(line)
		if !ok {
			continue
		}
		if level == 1 && doc.Title == doc.Slug {
			doc.Title = title
		}
		doc.Sections = append(doc.Sections, Section{Level: level, Title: title, Line: i})
	}

	for i := range doc.Sections {
		doc.Sections[i].EndLine = len(lines)
		for j := i + 1; j < len(doc.Sections); j++ {
			if doc.Sections[j].Level <= doc.Sections[i].Level {
				doc.Sections[i].EndLine = doc.Sections[j].Line
				break
			}
		}
	}

	taskSection := -1
	for i, section := range doc.Sections {
		if normalizeHeading(section.Title) == "implementation tasks" {
			taskSection = i
			break
		}
	}
	if taskSection >= 0 {
		section := doc.Sections[taskSection]
		for lineNo := section.Line + 1; lineNo < section.EndLine; lineNo++ {
			text, checked, ok := parseCheckbox(lines[lineNo])
			if !ok {
				continue
			}
			doc.Tasks = append(doc.Tasks, Task{Text: text, Checked: checked, Line: lineNo})
		}
		done, total := taskProgress(doc.Tasks)
		doc.Sections[taskSection].DoneCount = done
		doc.Sections[taskSection].TaskCount = total
	}

	return doc, nil
}

func middleItems(doc *PRD) []MiddleItem {
	items := make([]MiddleItem, 0, len(doc.Sections)+len(doc.Tasks))
	for i, section := range doc.Sections {
		label := section.Title
		if normalizeHeading(section.Title) == "implementation tasks" {
			if section.TaskCount > 0 {
				label = fmt.Sprintf("%s %d/%d", section.Title, section.DoneCount, section.TaskCount)
			} else {
				label = section.Title + " 0/0"
			}
		}
		items = append(items, MiddleItem{Kind: "section", Label: label, SectionIndex: i, Line: section.Line})
		if normalizeHeading(section.Title) == "implementation tasks" {
			for ti, task := range doc.Tasks {
				box := "☐"
				if task.Checked {
					box = "☑"
				}
				items = append(items, MiddleItem{Kind: "task", Label: "  " + box + " " + task.Text, TaskIndex: ti, Line: task.Line})
			}
		}
	}
	return items
}

func sectionText(doc *PRD, sectionIndex int) string {
	if doc == nil || sectionIndex < 0 || sectionIndex >= len(doc.Sections) {
		return ""
	}
	section := doc.Sections[sectionIndex]
	return strings.Join(doc.Lines[section.Line:section.EndLine], "\n")
}

func fullText(doc *PRD) string {
	if doc == nil {
		return ""
	}
	return strings.Join(doc.Lines, "\n")
}

func implementationTasksSection(doc *PRD) int {
	if doc == nil {
		return -1
	}
	for i, section := range doc.Sections {
		if normalizeHeading(section.Title) == "implementation tasks" {
			return i
		}
	}
	return -1
}

func toggleTaskLine(path string, line int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	sep := "\n"
	if strings.Contains(content, "\r\n") {
		sep = "\r\n"
	}
	trailingNewline := strings.HasSuffix(content, sep)
	lines := strings.Split(content, sep)
	if trailingNewline {
		lines = lines[:len(lines)-1]
	}
	if line < 0 || line >= len(lines) {
		return errors.New("task line moved; reload")
	}
	updated, ok := toggleCheckbox(lines[line])
	if !ok {
		return errors.New("selected line is no longer a checkbox; reload")
	}
	lines[line] = updated
	out := strings.Join(lines, sep)
	if trailingNewline {
		out += sep
	}
	return os.WriteFile(path, []byte(out), 0644)
}

func parseHeading(line string) (int, string, bool) {
	if !strings.HasPrefix(line, "#") {
		return 0, "", false
	}
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level == 0 || level > 6 || level >= len(line) || line[level] != ' ' {
		return 0, "", false
	}
	title := strings.TrimSpace(line[level+1:])
	if strings.HasSuffix(title, "#") {
		title = strings.TrimSpace(strings.TrimRight(title, "#"))
	}
	if title == "" {
		return 0, "", false
	}
	return level, title, true
}

func parseCheckbox(line string) (string, bool, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, "- [ ] ") && !strings.HasPrefix(strings.ToLower(trimmed), "- [x] ") {
		return "", false, false
	}
	checked := strings.EqualFold(trimmed[2:5], "[x]")
	return strings.TrimSpace(trimmed[6:]), checked, true
}

func toggleCheckbox(line string) (string, bool) {
	indent := len(line) - len(strings.TrimLeft(line, " \t"))
	trimmed := line[indent:]
	if strings.HasPrefix(trimmed, "- [ ] ") {
		return line[:indent] + "- [x] " + trimmed[6:], true
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "- [x] ") {
		return line[:indent] + "- [ ] " + trimmed[6:], true
	}
	return line, false
}

func taskProgress(tasks []Task) (int, int) {
	done := 0
	for _, task := range tasks {
		if task.Checked {
			done++
		}
	}
	return done, len(tasks)
}

func slugFromPath(path string) string {
	return filepath.Base(filepath.Dir(path))
}

func normalizeHeading(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
