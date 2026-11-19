package tui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// starAskLaunch is the one launch on which rdxt asks for a GitHub star, in
// the status line for a few seconds. It never asks again.
const starAskLaunch = 10

// launches counts how many times the viewer has been opened and returns the
// new count. Errors just mean no count, and no ask.
func launches() int {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return 0
		}
		dir = filepath.Join(home, ".local", "state")
	}
	dir = filepath.Join(dir, "rdxt")
	file := filepath.Join(dir, "launches")
	n := 0
	if b, err := os.ReadFile(file); err == nil {
		n, _ = strconv.Atoi(strings.TrimSpace(string(b)))
	}
	if n > starAskLaunch {
		return n // Stop counting once the ask has happened.
	}
	n++
	if os.MkdirAll(dir, 0o755) != nil || os.WriteFile(file, []byte(strconv.Itoa(n)), 0o644) != nil {
		return 0
	}
	return n
}
