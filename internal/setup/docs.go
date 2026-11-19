package setup

import (
	_ "embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

var (
	//go:embed assets/templates/blank.md
	tplBlank string
	//go:embed assets/templates/plan.md
	tplPlan string
	//go:embed assets/templates/notes.md
	tplNotes string
	//go:embed assets/templates/case.dok
	tplDok string
)

// NewDocument creates a document in dir from a template and returns its
// path. NAME gets .md (or .dok for the dok template) when it has no
// extension. It never overwrites a file.
func NewDocument(dir, name, template string) (string, error) {
	body := map[string]string{"blank": tplBlank, "plan": tplPlan, "notes": tplNotes, "dok": tplDok}[template]
	if body == "" {
		return "", errors.New("templates: blank, plan, notes, dok")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("a document needs a name")
	}
	if filepath.Ext(name) == "" {
		if template == "dok" {
			name += ".dok"
		} else {
			name += ".md"
		}
	}
	path, err := filepath.Abs(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return "", errors.New(filepath.Base(path) + " already exists")
	}
	title := titleFrom(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	text := strings.NewReplacer("{{title}}", title, "{{date}}", time.Now().Format("2006-01-02")).Replace(body)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = f.WriteString(text)
	return path, err
}

// titleFrom turns "meeting-notes" into "Meeting notes".
func titleFrom(name string) string {
	t := strings.TrimSpace(strings.NewReplacer("-", " ", "_", " ").Replace(name))
	if t == "" {
		return "Untitled"
	}
	r := []rune(t)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// skipDirs are folders FindDocuments doesn't look in.
var skipDirs = map[string]bool{"node_modules": true, "vendor": true, "target": true, "dist": true, "build": true}

// FindDocuments lists .md and .dok files under dir, two folders deep, up to
// limit, nearest first.
func FindDocuments(dir string, limit int) []string {
	var out []string
	base := strings.Count(filepath.Clean(dir), string(filepath.Separator))
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || len(out) >= limit {
			return filepath.SkipDir
		}
		if d.IsDir() {
			if p != dir && (strings.HasPrefix(d.Name(), ".") || skipDirs[d.Name()] ||
				strings.Count(p, string(filepath.Separator))-base > 2) {
				return filepath.SkipDir
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(p)) {
		case ".md", ".markdown", ".dok":
			out = append(out, p)
		}
		return nil
	})
	return out
}
