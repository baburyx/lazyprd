package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	projectFlag := flag.String("project", ".", "project root containing .scratch/*/PRD.md")
	flag.Parse()

	project, err := filepath.Abs(*projectFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve project: %v\n", err)
		os.Exit(1)
	}
	info, err := os.Stat(project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "project: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "project is not a directory: %s\n", project)
		os.Exit(1)
	}

	program := tea.NewProgram(newModel(project), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "lazyprd: %v\n", err)
		os.Exit(1)
	}
}
