# Contributing to rex-doc

Thanks for helping! Bug reports, ideas and pull requests are welcome.

**For now, open an issue before a pull request.** rdxt is young and its design is still settling, so let's agree on a change before you spend time on it. Pull requests without an agreed issue may be closed, kindly.

Please read this page before sending a pull request. rdxt is deliberately **opinionated**: its look and behavior come from a set of design decisions. A pull request that goes against one of them will be declined, however well it's written. That isn't about the quality of your work. It keeps rdxt consistent. If you'd like to challenge a decision, open an issue to discuss it first.

## What rdxt is for

rdxt is Markdown, made for Rex: real text in the terminal, fast, with links you can follow. Its main use is **reviewing what coding agents write**: you open a plan or report, read it, and approve it. So it opens fast and starts in View mode. It works in any terminal and looks best in [Rex](https://superlogical.com).

## The design principles, and why

### 1. Text, not pictures
Every word is a real terminal character. rdxt never draws the page as an image.

**Why:** an image ignores the terminal's font, size and theme. It can't be zoomed with ⌘+, selected or copied, and it's slow to scroll. That's the main problem with other terminal Markdown viewers. Images appear only where the document itself has an image.

### 2. The terminal sets the colors
rdxt uses only the terminal's 16-color palette plus its default foreground and background. That includes syntax highlighting.

**Why:** rdxt then matches whatever theme you use, light or dark, in any terminal, with no configuration. Hardcoded RGB colors would look wrong in someone's theme.

**Declined:** truecolor or 256-color values, built-in color themes.

### 3. No settings while in development
There's one look, chosen carefully. No themes, no style options.

**Why:** every option doubles what has to be designed and tested, and it lets "good enough" stay in. We'd rather get one design right. Settings may come once the design has settled.

**Declined:** options that change the look or toggle features.

### 4. Never lose your place
Switching modes, reloading, resizing and going Back all keep your spot in the document. Every rendered line keeps the source offset it came from, which is what makes this work.

**Declined:** changes that drop source offsets, or that jump back to the top.

### 5. Show each thing once, quietly
There's one status line, dim, with one blank line above it. No duplicate paths, no unlabeled widgets.

### 6. It stays until you close it
No popups or overlays that close by themselves.

## Decisions that are settled

These were chosen on purpose, often after comparing options side by side in a real terminal. Pull requests that change them will be declined unless an issue discussion has agreed on the change first.

| Decision | Why |
|---|---|
| **Two modes, View and Source.** Source mode is always editable. | There's no separate edit mode to explain. |
| **`e` enters Source mode, `Esc` goes back toward View.** `Tab` doesn't switch modes. | `Tab` is needed for indenting in Source mode and for moving between links in View mode. `Esc` always goes one level back. |
| **View mode is the default.** | The main use is reading and approving agent output. |
| **Frontmatter is hidden in View mode.** `m` shows it as a card. | Most previews show frontmatter badly. When shown, it should look deliberate. |
| **Task markers are bold round brackets**: `( )` when open, `(✔)` with a green heavy check when done, and done items dimmed. | Chosen from several options. They don't clash with bullets or alert icons, and finished items fade back in long review lists. |
| **Alert icons and markers must be one cell wide** (● ◆ ■ ▲ ▼). | Characters with ambiguous width (like ⓘ) are drawn two cells wide in some terminals and run into the next character. |
| **H1 is always in capitals, with a heavy magenta rule; H2 gets a thin rule.** A frontmatter `title` is shown as the H1 when the document has no H1. | Headings need weight without big text. Most terminals, including Rex, don't support OSC 66 text sizing yet. rdxt detects it and will use it when available. |
| **The reading width is at most 88 columns**, centered, with a compact layout under 60 columns. | Long lines are hard to read, and narrow panes (iPad split view, phones) need tighter margins. |

## Welcome contributions

- **Rendering bugs:** anything that renders incorrectly compared with CommonMark or GFM
- **Terminal compatibility:** something that breaks in a particular terminal, with the terminal's name and version
- **Speed:** startup time, scrolling, large files
- **Tests**, especially test files with tricky Markdown in `testdata/`
- **Elements that aren't supported yet**, as long as they follow the principles above

## Checking your change

```sh
go vet ./...
go test ./...
go build -o rdxt . && ./rdxt testdata/sample.md
```

`./rdxt -p --no-color -w 44 FILE` prints the layout at a given width, which is handy for checking narrow panes.

If your change affects how something looks, include before-and-after screenshots from a real terminal at a normal width and at a narrow one (under 60 columns).

## Dependencies

Keep them few. rdxt is a single static binary that should start in under 50 ms. Explain any new dependency in the pull request.
