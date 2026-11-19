// rdxt reads Markdown in a terminal.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"golang.org/x/term"

	"github.com/travelingflwr/rex-doc/internal/doc"
	"github.com/travelingflwr/rex-doc/internal/tui"
)

var version = "dev"

const usage = `rdxt reads Markdown in a terminal.

Usage:
  rdxt FILE           open FILE full screen
  rdxt -p FILE        print FILE laid out and exit (also when output is piped)
  rdxt -p -           print Markdown read from stdin

Options:
  -p, --print         print instead of opening full screen
  -w, --width N       layout width for --print (default: the terminal's, or 80)
      --no-color      print without colors (also when NO_COLOR is set)
  -v, --version       print the version

Keys:
  j/k  ↓/↑           scroll a line           Space/b  PgDn/PgUp   scroll a page
  Ctrl-d/Ctrl-u      scroll half a page      gg/G  Home/End       top / end
  }/{                next / previous heading o                    outline
  Tab/Shift-Tab      select a link           Enter or click       follow it
  f                  link hints              H / L                back / forward
  /                  search                  n / N                next / previous match
  m                  show frontmatter        r                    reload
  Esc                clear selection         q                    quit

★ Like rdxt? A star helps others find it: https://github.com/travelingflwr/rex-doc
`

func main() {
	fs := flag.NewFlagSet("rdxt", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	var print, noColor, showVersion bool
	var imagePath string
	var width int
	fs.BoolVar(&print, "p", false, "")
	fs.BoolVar(&print, "print", false, "")
	fs.IntVar(&width, "w", 0, "")
	fs.IntVar(&width, "width", 0, "")
	fs.BoolVar(&noColor, "no-color", false, "")
	fs.BoolVar(&showVersion, "v", false, "")
	fs.BoolVar(&showVersion, "version", false, "")
	fs.StringVar(&imagePath, "image", "", "")
	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			return
		}
		os.Exit(2)
	}
	if showVersion {
		fmt.Println("rdxt", version)
		fmt.Println("★ Like it? Star it: https://github.com/travelingflwr/rex-doc")
		return
	}
	if imagePath != "" {
		if err := tui.RunImage(imagePath); err != nil {
			fmt.Fprintln(os.Stderr, "rdxt:", err)
			os.Exit(1)
		}
		return
	}
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}
	path := fs.Arg(0)
	stdoutTTY := term.IsTerminal(int(os.Stdout.Fd()))
	if print || !stdoutTTY || path == "-" {
		if os.Getenv("NO_COLOR") != "" {
			noColor = true
		}
		if err := printDoc(path, width, !noColor); err != nil {
			fmt.Fprintln(os.Stderr, "rdxt:", err)
			os.Exit(1)
		}
		return
	}
	if err := tui.Run(path); err != nil {
		fmt.Fprintln(os.Stderr, "rdxt:", err)
		os.Exit(1)
	}
}

func printDoc(path string, width int, color bool) error {
	var src []byte
	var err error
	if path == "-" {
		src, err = io.ReadAll(os.Stdin)
	} else {
		src, err = os.ReadFile(path)
	}
	if err != nil {
		return err
	}
	if width <= 0 {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			width = w
		} else if c, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && c > 0 {
			width = c
		} else {
			width = 80
		}
	}
	opts := doc.OptionsFor(path)
	opts.ShowFrontmatter = true
	d := doc.Render(src, width, opts)
	for i := range d.Lines {
		fmt.Println(d.Format(i, color))
	}
	return nil
}
