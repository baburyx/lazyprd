package main

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

const (
	focusPRDs = iota
	focusMiddle
	focusPreview
)

type model struct {
	project string
	width   int
	height  int

	prds       []PRDSummary
	prdCursor  int
	prdTop     int
	midCursor  int
	midTop     int
	previewTop int
	focus      int

	filtering   bool
	filter      string
	status      string
	prompt      string
	confirmQuit bool
}

type reloadMsg struct {
	prds   []PRDSummary
	status string
	err    error
}

type copyMsg struct {
	err error
}

type footerPair struct {
	key    string
	action string
}

var (
	headerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("235"))
	headerTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("75")).Padding(0, 1)
	headerMetaStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("110")).Background(lipgloss.Color("235"))
	pathStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("110"))
	focusedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	blurredStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	paneTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("81"))
	sectionStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("147"))
	selectedStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("75"))
	dimStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	doneStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	warnStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("221"))
	accentStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("213"))
	statusStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("236"))
	statusModeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("221")).Padding(0, 1)
	keyStyle         = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("110")).Padding(0, 1)
	actionStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("236"))
	sepStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Background(lipgloss.Color("236"))
)

func newModel(project string) model {
	return model{project: project, focus: focusMiddle, status: "loading PRDs"}
}

func (m model) Init() tea.Cmd {
	return loadPRDs(m.project, "loaded")
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureVisible()
		return m, tea.ClearScreen
	case reloadMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			return m, nil
		}
		oldPath := m.selectedPRDPath()
		m.prds = msg.prds
		m.restorePRDSelection(oldPath)
		if oldPath == "" {
			m.selectDefaultMiddle()
		}
		m.clampCursors()
		m.ensureVisible()
		m.status = fmt.Sprintf("%s: %d PRD(s)", msg.status, len(m.prds))
		return m, nil
	case copyMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.status = "copied"
		}
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	}
	return m, nil
}

func (m model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.confirmQuit {
		switch key {
		case "y", "enter", "q", "ctrl+c":
			return m, tea.Quit
		case "n", "esc":
			m.confirmQuit = false
			m.status = "quit cancelled"
			return m, nil
		default:
			m.confirmQuit = false
			m.status = "quit cancelled"
			return m, nil
		}
	}
	if m.filtering {
		switch key {
		case "esc":
			m.filtering = false
			m.filter = ""
			m.clampCursors()
			m.ensureVisible()
		case "enter":
			m.filtering = false
		case "backspace", "ctrl+h":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.clampCursors()
				m.ensureVisible()
			}
		default:
			if len(msg.Runes) > 0 {
				m.filter += string(msg.Runes)
				m.clampCursors()
				m.ensureVisible()
			}
		}
		return m, nil
	}

	switch key {
	case "q", "ctrl+c":
		m.confirmQuit = true
		m.status = "quit?"
		return m, nil
	case "tab":
		m.focus = (m.focus + 1) % 3
	case "shift+tab":
		m.focus = (m.focus + 2) % 3
	case "ctrl+h":
		if m.focus > focusPRDs {
			m.focus--
		}
	case "ctrl+l", "enter":
		if m.focus < focusPreview {
			m.focus++
		}
	case "ctrl+j", "j", "down":
		m.move(1)
	case "ctrl+k", "k", "up":
		m.move(-1)
	case "pgdown":
		m.move(8)
	case "pgup":
		m.move(-8)
	case "g", "home":
		m.moveToStart()
	case "G", "end":
		m.moveToEnd()
	case "/":
		if m.focus != focusPreview {
			m.filtering = true
			m.filter = ""
		}
	case "esc":
		m.prompt = ""
		m.filter = ""
		m.filtering = false
	case " ":
		return m.toggleSelectedTask()
	case "r":
		m.status = "reloading"
		return m, loadPRDs(m.project, "reloaded")
	case "p":
		m.previewPrompt()
	case "y":
		text := m.copyText()
		m.status = "copying"
		return m, func() tea.Msg { return copyMsg{err: copyToClipboard(text)} }
	}
	return m, nil
}

func loadPRDs(project, status string) tea.Cmd {
	return func() tea.Msg {
		prds, err := discoverPRDs(project)
		return reloadMsg{prds: prds, status: status, err: err}
	}
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.height < 6 || m.width < 30 {
		return truncate("lazyprd: terminal too small", m.width)
	}

	top := m.renderHeader(m.width)
	status := m.renderStatus(m.width)
	bodyTotalHeight := max(3, m.height-3)
	body := ""
	if m.width < 70 {
		if m.focus == focusPreview {
			body = m.renderPreview(m.width, bodyTotalHeight)
		} else {
			body = m.renderMiddle(m.width, bodyTotalHeight)
		}
	} else if m.width < 110 {
		midW := clamp(m.width*46/100, 34, 54)
		rightW := max(30, m.width-midW)
		middle := m.renderMiddle(midW, bodyTotalHeight)
		right := m.renderPreview(rightW, bodyTotalHeight)
		body = lipgloss.JoinHorizontal(lipgloss.Top, middle, right)
	} else {
		leftTotal, midTotal, rightTotal := paneWidths(m.width)
		left := m.renderPRDs(leftTotal, bodyTotalHeight)
		middle := m.renderMiddle(midTotal, bodyTotalHeight)
		right := m.renderPreview(rightTotal, bodyTotalHeight)
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, middle, right)
	}
	return lipgloss.JoinVertical(lipgloss.Left, top, body, status)
}

func (m model) renderHeader(width int) string {
	inner := max(20, width)
	project := truncate(m.project, max(12, inner-50))
	progress := "0/0 tasks"
	selected := "No PRD selected"
	next := "No implementation task found"
	if doc := m.selectedDoc(); doc != nil {
		done, total := taskProgress(doc.Tasks)
		progress = fmt.Sprintf("%d/%d tasks", done, total)
		selected = truncate(doc.Title, max(14, inner/3))
		if item, ok := m.selectedMiddleItem(); ok && item.Kind == "task" {
			next = "Next: " + truncate(doc.Tasks[item.TaskIndex].Text, max(12, inner/2))
		}
	}
	line1Left := headerTitleStyle.Render(" lazyprd ") + " " + headerMetaStyle.Render(project)
	line1Right := fmt.Sprintf("%d PRD(s)  %s", len(m.prds), progress)
	line1 := padBetween(line1Left, line1Right, inner)
	line2Left := accentStyle.Render("Working on ") + selected
	line2Right := warnStyle.Render(next)
	line2 := padBetween(line2Left, line2Right, inner)
	return lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(padLine(line1, inner)), headerStyle.Render(padLine(line2, inner)))
}

func (m model) renderPRDs(width, height int) string {
	innerWidth := max(1, width-2)
	innerHeight := max(1, height-2)
	rows := []string{}
	items := m.filteredPRDs()
	listRows := m.prdListRows(innerHeight)

	if len(items) == 0 {
		rows = append(rows, dimStyle.Render("No .scratch/*/PRD.md files"))
	} else {
		start, end := visibleRange(m.prdTop, listRows, len(items))
		for visible := start; visible < end; visible++ {
			idx := items[visible]
			prd := m.prds[idx]
			label := fmt.Sprintf("%s %s", progressLabel(prd.Done, prd.Total), truncate(prd.Title, max(8, innerWidth-8)))
			if visible == m.prdCursor {
				label = selectedStyle.Render(padLine(label, innerWidth))
			}
			rows = append(rows, label)
		}
	}

	if innerHeight >= 14 {
		rows = append(rows, "")
		rows = append(rows, paneTitleStyle.Render("● Selected"))
		rows = append(rows, m.selectedPRDDetails(innerWidth)...)
	} else if doc := m.selectedDoc(); doc != nil && innerHeight >= 5 {
		done, total := taskProgress(doc.Tasks)
		rows = append(rows, dimStyle.Render(truncate(doc.Slug, innerWidth)))
		rows = append(rows, fmt.Sprintf("progress %s", progressLabel(done, total)))
	}
	return renderBox("PRDs "+fmt.Sprintf("%d", len(m.prds)), rows, width, height, m.focus == focusPRDs)
}

func (m model) renderMiddle(width, height int) string {
	innerWidth := max(1, width-2)
	innerHeight := max(1, height-2)
	rows := []string{}
	doc := m.selectedDoc()
	if doc == nil {
		rows = append(rows, dimStyle.Render("Select a PRD"))
		return renderBox("Outline / Tasks", rows, width, height, m.focus == focusMiddle)
	}

	items := m.filteredMiddleItems(doc)
	if len(items) == 0 {
		rows = append(rows, dimStyle.Render("No matching sections/tasks"))
		return renderBox("Outline / Tasks", rows, width, height, m.focus == focusMiddle)
	}

	if len(doc.Tasks) == 0 {
		rows = append(rows, dimStyle.Render("read-only: no Implementation Tasks"))
	}

	listRows := max(1, innerHeight-len(rows))
	start, end := visibleRange(m.midTop, listRows, len(items))
	for visible := start; visible < end; visible++ {
		item := items[visible]
		label := m.middleLabel(doc, item, innerWidth)
		if visible == m.midCursor {
			label = selectedStyle.Render(padLine(label, innerWidth))
		}
		rows = append(rows, label)
	}
	return renderBox("Outline / Tasks", rows, width, height, m.focus == focusMiddle)
}

func (m model) renderPreview(width, height int) string {
	innerWidth := max(1, width-2)
	innerHeight := max(1, height-2)
	title := "Preview"
	if m.prompt != "" {
		title = "Prompt Preview"
	} else if item, ok := m.selectedMiddleItem(); ok && item.Kind == "task" {
		title = "Task Preview"
	}

	contentRows := formatPreview(m.previewText(), innerWidth)
	if len(contentRows) == 0 {
		contentRows = []string{dimStyle.Render("No preview")}
	}
	listRows := max(1, innerHeight)
	top := clamp(m.previewTop, 0, max(0, len(contentRows)-1))
	start, end := visibleRange(top, listRows, len(contentRows))
	rows := contentRows[start:end]
	return renderBox(title, rows, width, height, m.focus == focusPreview)
}

func (m model) renderStatus(width int) string {
	if m.confirmQuit {
		mode := statusModeStyle.Render("QUIT?")
		pairs := footerPairs(width-lipgloss.Width(mode), []footerPair{{"y/enter", "quit"}, {"esc/n", "cancel"}})
		return statusStyle.Render(padLine(mode+pairs, width))
	}
	if m.filtering {
		mode := statusModeStyle.Render("FILTER")
		filterText := actionStyle.Render(" " + truncate(m.filter, max(1, width/3)) + " ")
		shortcuts := footerPairs(width-lipgloss.Width(mode)-lipgloss.Width(filterText), []footerPair{{"enter", "accept"}, {"esc", "clear"}})
		return statusStyle.Render(padLine(mode+filterText+shortcuts, width))
	}
	mode := statusModeStyle.Render(strings.ToUpper(m.focusName()))
	pairs := []footerPair{{"j/k", "move"}, {"tab", "focus"}, {"/", "filter"}, {"space", "toggle"}, {"p", "prompt"}, {"y", "copy"}, {"r", "reload"}, {"q", "quit"}}
	if width < 96 {
		pairs = []footerPair{{"j/k", "move"}, {"tab", "focus"}, {"/", "filter"}, {"space", "toggle"}, {"p", "prompt"}, {"q", "quit"}}
	}
	if width < 70 {
		pairs = []footerPair{{"j/k", "move"}, {"tab", "focus"}, {"space", "toggle"}, {"q", "quit"}}
	}
	if m.prompt != "" {
		pairs = []footerPair{{"y", "copy prompt"}, {"esc", "close"}, {"j/k", "scroll"}, {"tab", "focus"}, {"q", "quit"}}
	}
	status := strings.TrimSpace(m.status)
	left := mode
	if status != "" {
		left += actionStyle.Render(" "+truncate(status, max(8, width/4))+" ") + sepStyle.Render("│")
	}
	shortcuts := footerPairs(width-lipgloss.Width(left), pairs)
	return statusStyle.Render(padLine(left+shortcuts, width))
}

func (m model) selectedPRDDetails(width int) []string {
	doc := m.selectedDoc()
	if doc == nil {
		return []string{dimStyle.Render("None")}
	}
	rel, err := filepath.Rel(m.project, doc.Path)
	if err != nil {
		rel = doc.Path
	}
	done, total := taskProgress(doc.Tasks)
	rows := []string{
		accentStyle.Render(truncate(doc.Title, width)),
		dimStyle.Render(truncate(rel, width)),
		fmt.Sprintf("progress %s", progressLabel(done, total)),
	}
	if total == 0 {
		rows = append(rows, dimStyle.Render("no task section"))
	}
	rows = append(rows, "", paneTitleStyle.Render("Keys"))
	rows = append(rows, dimStyle.Render("/ filter"), dimStyle.Render("space toggle"), dimStyle.Render("p prompt  y copy"))
	return rows
}

func (m model) middleLabel(doc *PRD, item MiddleItem, width int) string {
	if item.Kind == "section" {
		section := doc.Sections[item.SectionIndex]
		indent := strings.Repeat("  ", max(0, section.Level-1))
		label := indent + section.Title
		if normalizeHeading(section.Title) == "implementation tasks" {
			label = fmt.Sprintf("%s%s %d/%d", indent, section.Title, section.DoneCount, section.TaskCount)
			return accentStyle.Render(truncate(label, width))
		}
		return sectionStyle.Render(truncate(label, width))
	}

	task := doc.Tasks[item.TaskIndex]
	box := "[ ]"
	style := warnStyle
	if task.Checked {
		box = "[x]"
		style = doneStyle
	}
	return style.Render(truncate("  "+box+" "+task.Text, width))
}

func (m model) focusName() string {
	switch m.focus {
	case focusPRDs:
		return "prds"
	case focusMiddle:
		return "tasks"
	case focusPreview:
		return "preview"
	default:
		return "ready"
	}
}

func (m *model) move(delta int) {
	switch m.focus {
	case focusPRDs:
		count := len(m.filteredPRDs())
		before := m.prdCursor
		m.prdCursor = clamp(m.prdCursor+delta, 0, max(0, count-1))
		if before != m.prdCursor {
			m.selectDefaultMiddle()
			m.previewTop = 0
			m.prompt = ""
		}
	case focusMiddle:
		count := 0
		if doc := m.selectedDoc(); doc != nil {
			count = len(m.filteredMiddleItems(doc))
		}
		before := m.midCursor
		m.midCursor = clamp(m.midCursor+delta, 0, max(0, count-1))
		if before != m.midCursor {
			m.previewTop = 0
			m.prompt = ""
		}
	case focusPreview:
		m.previewTop = max(0, m.previewTop+delta)
	}
	m.ensureVisible()
}

func (m *model) moveToStart() {
	if m.focus == focusPreview {
		m.previewTop = 0
		return
	}
	m.move(-100000)
}

func (m *model) moveToEnd() {
	if m.focus == focusPreview {
		m.previewTop = 100000
		return
	}
	m.move(100000)
}

func (m model) toggleSelectedTask() (tea.Model, tea.Cmd) {
	doc := m.selectedDoc()
	item, ok := m.selectedMiddleItem()
	if doc == nil || !ok || item.Kind != "task" {
		m.status = "select an implementation task to toggle"
		return m, nil
	}
	oldPath := doc.Path
	line := doc.Tasks[item.TaskIndex].Line
	if err := toggleTaskLine(oldPath, line); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.status = "toggled task"
	return m, loadPRDs(m.project, "reloaded")
}

func (m *model) previewPrompt() {
	doc := m.selectedDoc()
	item, ok := m.selectedMiddleItem()
	if doc == nil || !ok || item.Kind != "task" {
		m.status = "select an implementation task for prompt preview"
		return
	}
	m.prompt = generateTaskPrompt(m.project, doc, item.TaskIndex)
	m.previewTop = 0
	m.status = "prompt preview"
}

func (m model) copyText() string {
	if m.prompt != "" {
		return m.prompt
	}
	return m.previewText()
}

func (m model) previewText() string {
	if m.prompt != "" {
		return m.prompt
	}
	doc := m.selectedDoc()
	if doc == nil {
		return ""
	}
	item, ok := m.selectedMiddleItem()
	if !ok {
		return documentOverview(doc)
	}
	switch item.Kind {
	case "task":
		return taskPreview(doc, item.TaskIndex)
	case "section":
		return sectionText(doc, item.SectionIndex)
	default:
		return documentOverview(doc)
	}
}

func taskPreview(doc *PRD, taskIndex int) string {
	if doc == nil || taskIndex < 0 || taskIndex >= len(doc.Tasks) {
		return ""
	}
	task := doc.Tasks[taskIndex]
	box := "[ ]"
	status := "pending"
	if task.Checked {
		box = "[x]"
		status = "done"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Selected Task\n\n%s %s\n\n", box, task.Text)
	fmt.Fprintf(&b, "Status: %s\nLine: %d\n\n", status, task.Line+1)
	if problem := firstSectionSnippet(doc, "problem statement", 700); problem != "" {
		fmt.Fprintf(&b, "## Problem Context\n\n%s\n\n", problem)
	}
	fmt.Fprintf(&b, "## Nearby Tasks\n\n%s\n\n", nearbyTasks(doc, taskIndex, 3))
	b.WriteString("## Actions\n\nSpace toggles this checkbox. Press p to generate a focused agent prompt, then y to copy it.")
	return b.String()
}

func documentOverview(doc *PRD) string {
	done, total := taskProgress(doc.Tasks)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", doc.Title)
	fmt.Fprintf(&b, "Progress: %d/%d tasks\n\n", done, total)
	if problem := firstSectionSnippet(doc, "problem statement", 900); problem != "" {
		b.WriteString(problem)
		b.WriteString("\n")
	}
	return b.String()
}

func (m model) filteredPRDs() []int {
	if m.filter == "" || m.focus != focusPRDs {
		idx := make([]int, len(m.prds))
		for i := range m.prds {
			idx[i] = i
		}
		return idx
	}
	labels := make([]string, len(m.prds))
	for i, prd := range m.prds {
		labels[i] = prd.Title + " " + prd.Slug
	}
	matches := fuzzy.Find(m.filter, labels)
	idx := make([]int, 0, len(matches))
	for _, match := range matches {
		idx = append(idx, match.Index)
	}
	return idx
}

func (m model) filteredMiddleItems(doc *PRD) []MiddleItem {
	items := middleItems(doc)
	if m.filter == "" || m.focus != focusMiddle {
		return items
	}
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	matches := fuzzy.Find(m.filter, labels)
	filtered := make([]MiddleItem, 0, len(matches))
	for _, match := range matches {
		filtered = append(filtered, items[match.Index])
	}
	return filtered
}

func (m model) selectedDoc() *PRD {
	items := m.filteredPRDs()
	if len(items) == 0 || m.prdCursor >= len(items) {
		return nil
	}
	return m.prds[items[m.prdCursor]].Document
}

func (m model) selectedMiddleItem() (MiddleItem, bool) {
	doc := m.selectedDoc()
	if doc == nil {
		return MiddleItem{}, false
	}
	items := m.filteredMiddleItems(doc)
	if len(items) == 0 || m.midCursor >= len(items) {
		return MiddleItem{}, false
	}
	return items[m.midCursor], true
}

func (m model) selectedPRDPath() string {
	items := m.filteredPRDs()
	if len(items) == 0 || m.prdCursor >= len(items) {
		return ""
	}
	return m.prds[items[m.prdCursor]].Path
}

func (m *model) restorePRDSelection(path string) {
	m.prdCursor = 0
	if path == "" {
		return
	}
	for i, prd := range m.prds {
		if prd.Path == path {
			m.prdCursor = i
			return
		}
	}
}

func (m *model) selectDefaultMiddle() {
	m.midCursor = 0
	m.midTop = 0
	doc := m.selectedDoc()
	if doc == nil {
		return
	}
	items := middleItems(doc)
	for i, item := range items {
		if item.Kind == "task" && !doc.Tasks[item.TaskIndex].Checked {
			m.midCursor = i
			return
		}
	}
	for i, item := range items {
		if item.Kind == "section" && normalizeHeading(doc.Sections[item.SectionIndex].Title) == "problem statement" {
			m.midCursor = i
			return
		}
	}
}

func (m *model) clampCursors() {
	m.prdCursor = clamp(m.prdCursor, 0, max(0, len(m.filteredPRDs())-1))
	count := 0
	if doc := m.selectedDoc(); doc != nil {
		count = len(m.filteredMiddleItems(doc))
	}
	m.midCursor = clamp(m.midCursor, 0, max(0, count-1))
}

func (m *model) ensureVisible() {
	bodyHeight := max(1, m.height-3)
	contentHeight := max(1, bodyHeight-2)
	m.prdTop = ensureTop(m.prdTop, m.prdCursor, m.prdListRows(contentHeight))
	m.midTop = ensureTop(m.midTop, m.midCursor, max(1, contentHeight-1))
}

func (m model) prdListRows(height int) int {
	if height >= 14 {
		return max(1, height-10)
	}
	return max(1, height-1)
}

func paneWidths(width int) (int, int, int) {
	if width < 100 {
		left := max(24, width/4)
		mid := max(32, width/3)
		right := max(30, width-left-mid)
		return left, mid, right
	}
	left := clamp(width*24/100, 30, 44)
	mid := clamp(width*34/100, 40, 64)
	right := max(40, width-left-mid)
	return left, mid, right
}

func progressLabel(done, total int) string {
	if total == 0 {
		return dimStyle.Render("0/0")
	}
	label := fmt.Sprintf("%d/%d", done, total)
	if done == total {
		return doneStyle.Render(label)
	}
	return warnStyle.Render(label)
}

func formatPreview(text string, width int) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var rows []string
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			rows = append(rows, "")
			continue
		}
		level, title, ok := parseHeading(line)
		if ok {
			rows = append(rows, sectionStyle.Render(truncate(strings.Repeat("#", level)+" "+title, width)))
			continue
		}
		prefix := ""
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "- ") {
			prefix = strings.Repeat(" ", len(line)-len(trimmed))
		}
		for _, wrapped := range wrapLine(strings.TrimRight(line, " \t"), width) {
			if prefix != "" && strings.HasPrefix(strings.TrimSpace(wrapped), "- [x]") {
				rows = append(rows, doneStyle.Render(wrapped))
			} else {
				rows = append(rows, wrapped)
			}
		}
	}
	return rows
}

func wrapLine(line string, width int) []string {
	width = max(8, width)
	if lipgloss.Width(line) <= width {
		return []string{line}
	}
	indent := len(line) - len(strings.TrimLeft(line, " \t"))
	continuation := strings.Repeat(" ", indent)
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{""}
	}
	var rows []string
	current := words[0]
	for _, word := range words[1:] {
		if lipgloss.Width(current)+1+lipgloss.Width(word) > width {
			rows = append(rows, current)
			current = continuation + word
			continue
		}
		current += " " + word
	}
	rows = append(rows, current)
	return rows
}

func visibleRange(top, rows, total int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	rows = max(1, rows)
	top = clamp(top, 0, max(0, total-1))
	end := top + rows
	if end > total {
		end = total
	}
	return top, end
}

func ensureTop(top, cursor, rows int) int {
	rows = max(1, rows)
	if cursor < top {
		return cursor
	}
	if cursor >= top+rows {
		return cursor - rows + 1
	}
	return max(0, top)
}

func renderBox(title string, rows []string, width, height int, focused bool) string {
	width = max(4, width)
	height = max(2, height)
	innerWidth := max(1, width-2)
	innerHeight := max(0, height-2)
	color := blurredStyle
	if focused {
		color = focusedStyle
	}

	top := color.Render(borderTop(title, width))
	bottom := color.Render("╰" + strings.Repeat("─", width-2) + "╯")
	lines := []string{top}
	for i := 0; i < innerHeight; i++ {
		content := ""
		if i < len(rows) {
			content = rows[i]
		}
		lines = append(lines, color.Render("│")+padLine(content, innerWidth)+color.Render("│"))
	}
	lines = append(lines, bottom)
	return strings.Join(lines, "\n")
}

func borderTop(title string, width int) string {
	label := " " + title + " "
	if lipgloss.Width(label) > width-2 {
		label = truncate(label, width-2)
	}
	right := max(0, width-2-lipgloss.Width(label))
	return "╭" + label + strings.Repeat("─", right) + "╮"
}

func padLine(s string, width int) string {
	width = max(0, width)
	if lipgloss.Width(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-lipgloss.Width(s))
}

func footerPairs(width int, pairs []footerPair) string {
	if width <= 0 {
		return ""
	}
	parts := make([]string, 0, len(pairs)*2)
	used := 0
	for i, pair := range pairs {
		part := keyStyle.Render(pair.key) + actionStyle.Render(" "+pair.action)
		if i > 0 {
			part = sepStyle.Render("  ") + part
		}
		partWidth := lipgloss.Width(part)
		if used+partWidth > width {
			break
		}
		parts = append(parts, part)
		used += partWidth
	}
	return strings.Join(parts, "")
}

func truncate(s string, width int) string {
	width = max(1, width)
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	if width <= 1 {
		return "…"
	}
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func padBetween(left, right string, width int) string {
	if lipgloss.Width(left)+lipgloss.Width(right)+1 > width {
		left = truncate(left, max(1, width-lipgloss.Width(right)-1))
	}
	gap := max(1, width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", gap) + right
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
