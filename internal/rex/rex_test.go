package rex

import (
	"os"
	"strings"
	"testing"
)

func TestSessionFromRexPreferences(t *testing.T) {
	if _, err := os.Stat(os.ExpandEnv("$HOME/Library/Preferences/com.superlogical.rex.plist")); err != nil {
		t.Skip("Rex isn't installed here")
	}
	t.Setenv("REX_SESSION", "")
	if s := Session(); !strings.HasPrefix(s, "session:") {
		t.Fatalf("Session() = %q, want a session: ID from Rex's preferences", s)
	}
}
