package main

import (
	"bytes"
	"errors"
	"os/exec"
	"runtime"
)

func copyToClipboard(text string) error {
	if text == "" {
		return errors.New("nothing to copy")
	}

	var candidates [][]string
	switch runtime.GOOS {
	case "darwin":
		candidates = [][]string{{"pbcopy"}}
	case "windows":
		candidates = [][]string{{"clip"}}
	default:
		candidates = [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard"}, {"xsel", "--clipboard", "--input"}}
	}

	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate[0]); err != nil {
			continue
		}
		cmd := exec.Command(candidate[0], candidate[1:]...)
		cmd.Stdin = bytes.NewBufferString(text)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return errors.New("no clipboard command found")
}
