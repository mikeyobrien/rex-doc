package tui

import (
	"os"
	"path/filepath"
	"strings"
)

// Recent documents, newest first, for the Rex palette's Open Document.

func recentFile() string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "rdxt", "recent")
}

// Recent returns recently opened documents that still exist, newest first.
func Recent() []string {
	b, err := os.ReadFile(recentFile())
	if err != nil {
		return nil
	}
	var out []string
	for _, p := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			out = append(out, p)
		}
	}
	return out
}

func recordRecent(path string) {
	list := []string{path}
	for _, p := range Recent() {
		if p != path && len(list) < 50 {
			list = append(list, p)
		}
	}
	f := recentFile()
	if os.MkdirAll(filepath.Dir(f), 0o755) == nil {
		os.WriteFile(f, []byte(strings.Join(list, "\n")+"\n"), 0o644)
	}
}
