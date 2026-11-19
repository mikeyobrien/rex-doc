package setup

import "testing"

func TestRemoveBlockLeavesTheRest(t *testing.T) {
	text := "a\n\n" + markStart + "\ndofile(\"x\")\n" + markEnd + "\nb\n"
	if got := removeBlock(text, markStart, markEnd); got != "a\nb\n" {
		t.Fatalf("removeBlock = %q, want %q", got, "a\nb\n")
	}
	if got := removeBlock("nothing here\n", markStart, markEnd); got != "nothing here\n" {
		t.Fatalf("removeBlock without a block changed the text: %q", got)
	}
}

func TestNewDocumentNames(t *testing.T) {
	dir := t.TempDir()
	p, err := NewDocument(dir, "meeting-notes", "notes")
	if err != nil || p != dir+"/meeting-notes.md" {
		t.Fatalf("NewDocument = %q, %v", p, err)
	}
	if _, err := NewDocument(dir, "meeting-notes", "notes"); err == nil {
		t.Fatal("NewDocument overwrote an existing file")
	}
	if p, _ := NewDocument(dir, "case", "dok"); p != dir+"/case.dok" {
		t.Fatalf("dok template made %q, want case.dok", p)
	}
	if got := titleFrom("login_fix-plan"); got != "Login fix plan" {
		t.Fatalf("titleFrom = %q", got)
	}
}
