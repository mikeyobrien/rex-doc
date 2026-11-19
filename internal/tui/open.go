package tui

import (
	"net/url"
	"os"
	"os/exec"
	"runtime"
)

func pathUnescape(s string) (string, error) { return url.PathUnescape(s) }

// openExternal opens a URL or file with the system's default handler.
func openExternal(target string) error {
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	cmd := exec.Command(name, target)
	cmd.Stdout, cmd.Stderr = nil, nil
	return cmd.Start()
}

// openOther opens a file that isn't Markdown. In Rex it opens in the user's
// editor in a split beside rdxt; elsewhere the system decides.
func (v *viewer) openOther(file string) {
	if os.Getenv("REX_SESSION") != "" {
		editor := os.Getenv("VISUAL")
		if editor == "" {
			editor = os.Getenv("EDITOR")
		}
		if editor == "" {
			editor = "vi"
		}
		cmd := exec.Command("rex", "split", "--", editor, file)
		if err := cmd.Run(); err == nil {
			v.setFlash("opened beside")
			return
		}
	}
	if err := openExternal(file); err != nil {
		v.setFlash("can't open " + file)
		return
	}
	v.setFlash("opened")
}
