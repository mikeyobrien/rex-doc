package tui

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// inRex reports whether rdxt is running in a Rex terminal.
func inRex() bool {
	return os.Getenv("TERM_PROGRAM") == "rex" || os.Getenv("REX_SESSION") != ""
}

// rexFloating asks Rex whether rdxt's own block is in a floating layer. It
// runs a rex command, so call it off the main loop.
func rexFloating() bool {
	session, block := os.Getenv("REX_SESSION"), os.Getenv("REX_BLOCK")
	if session == "" || block == "" {
		return false
	}
	out, err := exec.Command("rex", "api", "call", "-s", session, "session.view").Output()
	if err != nil {
		return false
	}
	var view struct {
		Windows []struct {
			Layers []struct {
				Kind   string          `json:"kind"`
				Layout json.RawMessage `json:"layout"`
			} `json:"layers"`
		} `json:"windows"`
	}
	if json.Unmarshal(out, &view) != nil {
		return false
	}
	for _, w := range view.Windows {
		for _, l := range w.Layers {
			if l.Kind == "floating" && strings.Contains(string(l.Layout), `"`+block+`"`) {
				return true
			}
		}
	}
	return false
}

// rexLabel names rdxt's block after the document, so Rex's pane header
// shows it. It's best effort and runs in the background.
func rexLabel(path string) {
	block := os.Getenv("REX_BLOCK")
	if block == "" {
		return
	}
	cmd := exec.Command("rex", "block", "rename", block, filepath.Base(path))
	cmd.Stdout, cmd.Stderr = nil, nil
	go cmd.Run()
}
