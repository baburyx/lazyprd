# PRD - PRD TUI

> Synthesized 2026-06-10 from the design discussion for a standalone local TUI that navigates `/to-prd` output and tracks implementation progress through Markdown checkboxes.

## Problem Statement

The current `/to-prd` workflow creates local Markdown PRDs in `.scratch/<feature-slug>/PRD.md`. That is durable and agent-friendly, but it is not a pleasant way to inspect active PRDs, navigate their sections, or track which implementation tasks are complete. The user wants a pretty, keyboard-driven terminal UI with a lazygit-like split-pane workflow for browsing project PRDs and turning PRD tasklists into actionable implementation prompts.

## Solution

Build a standalone Go TUI app, `prd-tui`, using Bubble Tea. The app scans the selected project for `.scratch/*/PRD.md`, lists PRDs by recent modification time, parses ATX Markdown headings, treats only checkboxes under `## Implementation Tasks` as implementation progress, and lets the user toggle those checkboxes by mutating the source Markdown file. The app provides a three-pane layout: PRDs, sections/tasks, and preview. It also generates focused task prompts that can be previewed and copied to the clipboard.

The Markdown PRD remains the source of truth. There is no sidecar database in v1.

## User Stories

1. As a developer using `/to-prd`, I want to see all current project PRDs in one TUI, so that I can quickly choose what to work on.
2. As a developer, I want a lazygit-like split-pane layout, so that I can browse PRDs, sections, tasks, and previews without opening multiple editor buffers.
3. As a developer, I want `Implementation Tasks` checkboxes to represent progress, so that the PRD file itself tracks what has been implemented.
4. As a developer, I want pressing `Space` on a task to toggle the Markdown checkbox, so that progress updates are fast and visible in Git diffs.
5. As a developer, I want PRD-level and section-level progress counts, so that I can see how much work remains.
6. As a developer, I want vim-like keyboard navigation, so that the app feels fast and terminal-native.
7. As a developer, I want fuzzy filtering in list panes, so that I can jump to PRDs, sections, or tasks quickly.
8. As a developer, I want to preview a focused agent prompt for a selected task, so that I can hand off one task to a coding agent without dumping the whole PRD.
9. As a developer, I want to copy prompts or selected content to the clipboard, so that I can paste them into another agent session.
10. As a developer, I want older PRDs without `Implementation Tasks` to remain readable, so that existing PRDs are not broken by the new tool.

## Implementation Decisions

- The app is a standalone Go project outside any product repo. It is reusable across projects that follow the `.scratch/*/PRD.md` convention.
- Use Homebrew Go as the active Go toolchain. The current machine uses Go `1.26.4` from Homebrew.
- Use Bubble Tea for the TUI and Lip Gloss for restrained terminal styling.
- Use `github.com/sahilm/fuzzy` for fuzzy matching rather than implementing ranking in-house.
- The command should support both `prd-tui` and `prd-tui --project /path/to/project`.
- The default project root is the current working directory.
- Discovery is intentionally narrow in v1: only `<project>/.scratch/*/PRD.md`.
- PRDs are sorted by modification time descending.
- Display title comes from the first H1 heading, with the `.scratch/<slug>` folder name as fallback.
- The main layout has three panes: PRD list, sections/tasks, and preview.
- The middle pane is one mixed navigable list containing headings and `Implementation Tasks` rows.
- Only checkboxes under the normalized `## Implementation Tasks` heading count as implementation progress.
- Existing PRDs without `Implementation Tasks` stay readable and show `0/0`; the app does not auto-insert a task section in v1.
- Checkbox state is persisted by directly changing the selected source line from `- [ ]` to `- [x]` or back.
- Task identity is line-number based for mutation, with best-effort selection restoration by path, heading, line, and text after reloads.
- The parser is line-based and supports ATX headings only (`#` through `######`). This preserves the source file exactly except for checkbox markers.
- Rendering is light Markdown styling, not full Markdown rendering. Headings, selected rows, progress, and checked tasks get simple terminal styling.
- Navigation is pane-focus based. `Ctrl+h/j/k/l` moves focus between panes where applicable; `j/k` moves selection inside the focused pane; `Tab` and `Shift+Tab` cycle focus for non-vim fallback.
- The app is keyboard-only. Mouse input is out of scope.
- `/` enters fuzzy filter mode for the focused list pane. `Esc` clears/exits filter mode.
- `p` previews a generated task prompt in the right pane. `y` copies the current prompt, or selected content if no prompt is active.
- Clipboard support shells out to platform tools: `pbcopy` on macOS, `wl-copy` on Wayland, `xclip` or `xsel` on X11, and `clip` on Windows. If unavailable, show a status message.
- File watching is out of scope for v1. `r` manually rescans and reloads; toggling a task reloads the selected PRD automatically.
- A partial implementation already exists with PRD parsing, checkbox mutation, middle item generation, and prompt generation. The module metadata needs to be corrected because the initial `go mod init` was run from the wrong directory.

## Implementation Tasks

- [ ] Move or recreate Go module metadata inside the `prd-tui` project directory and remove any accidental parent-level module files if present.
- [ ] Re-run dependency setup in the project directory for Bubble Tea, Lip Gloss, and fuzzy matching.
- [ ] Add a command entrypoint that parses `--project`, resolves the project root, and starts the Bubble Tea program.
- [ ] Finish PRD discovery for `.scratch/*/PRD.md`, including empty-state handling and modified-time sorting.
- [ ] Finish the line-based Markdown parser for ATX headings, `Implementation Tasks` extraction, progress counts, and safe checkbox mutation.
- [ ] Add parser and mutator tests covering heading parsing, task extraction, discovery, and preservation of unrelated Markdown content.
- [ ] Implement the three-pane Bubble Tea model with PRD list, mixed sections/tasks list, and preview pane.
- [ ] Implement keyboard navigation: `Ctrl+h/j/k/l`, `Tab`, `Shift+Tab`, `j/k`, `Enter`, `Space`, `r`, `/`, `Esc`, `p`, `y`, and `q`.
- [ ] Implement fuzzy filtering for the focused PRD and section/task list panes.
- [ ] Implement light Markdown preview styling and progress display in the PRD list, task section label, and preview/status area.
- [ ] Implement task prompt preview using selected task, nearby tasks, problem context, implementation decisions, and testing decisions.
- [ ] Implement clipboard copying with platform command fallback and visible status messages.
- [ ] Verify the app against `/Users/hw3124/projects/AI-days-exhibit/.scratch/livechan-infinitrip/PRD.md`, including the no-task-section read-only behavior.
- [ ] Add a minimal README with install/run examples for `go run .`, `go install`, and `prd-tui --project /path/to/project`.
- [ ] Run `gofmt`, `go test ./...`, and a manual TUI smoke test before marking v1 complete.

## Testing Decisions

Good tests should protect external behavior and file safety rather than Bubble Tea internals. The most important invariant is that toggling a task mutates only the selected checkbox marker and preserves the rest of the PRD byte-for-byte as much as practical.

Test seams:

1. PRD discovery seam: given a temporary project tree, only `.scratch/*/PRD.md` files are found and sorted by modification time.
2. Parser seam: given Markdown text, ATX headings, sections, H1 title fallback, and `Implementation Tasks` checkboxes are extracted correctly.
3. Mutation seam: given a task source line, toggling changes only `- [ ]` to `- [x]` or back, preserves indentation and trailing newline behavior, and fails clearly if the line no longer contains a checkbox.
4. Prompt generation seam: given a parsed PRD and selected task, generated output includes selected task, nearby tasks, problem context, implementation context, testing expectations, and scoped implementation instructions.

Full terminal rendering tests are out of scope for v1. Manual smoke testing should cover launch, navigation, filtering, task toggling, prompt preview, clipboard copy, reload, and quit.

## Out of Scope

- Generic Markdown browsing outside `.scratch/*/PRD.md`.
- Editing arbitrary PRD prose from inside the TUI.
- Creating, renaming, deleting, or reordering tasks from inside the TUI.
- Nested tasklists or parent/child progress semantics.
- Mouse support.
- Filesystem watching.
- Launching OpenCode, Claude, or any other agent from inside the TUI.
- Full Markdown rendering with tables, images, HTML, or frontmatter support.
- Sidecar state files for task metadata, assignees, timestamps, dependencies, or notes.
- Special semantic extraction for dependencies, open decisions, or pending verifications outside `Implementation Tasks`.

## Further Notes

- `/to-prd` was updated on disk to include mandatory `## Implementation Tasks`, but the running OpenCode session still loaded the old skill content. Restart OpenCode for the updated skill to take effect.
- The generated PRD is intentionally more specific than the generic `/to-prd` template because it captures the full design-tree decisions from the grilling session.
- Existing old PRDs, including the AI-days Live-chan PRD, do not yet contain `Implementation Tasks`; the TUI should still display them without trying to repair them automatically.
