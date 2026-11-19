# rex-doc design

*Draft, 2026-11-19.*

## Goal

**Made for Rex.** The best way to read Markdown in a terminal. It works in any terminal, but it's designed around Rex first: Rex's panes, floating windows, palette and chrome. Rex is free and easy to get, so leaning on it is a feature, not a limit.

**Main use: reviewing what agents write**, at home and at work. An agent writes a plan or report, I open it, read it, and approve it. So it's built for:
- **Opening fast.** The first screen shows immediately (goal: under 50 ms). Syntax highlighting and other slow work happen after that.
- **Reading by default.** It opens in View mode. Editing is one key away, but it's not the starting point.

## Name

- Project and package: **`rex-doc`**
- Command: **`rdxt`**, with `rex-doc` installed alongside it as a symlink.
  - Chosen because all four keys sit close together on the left hand. If it needs a meaning: "**r**ea**d** te**xt**".
  - `rdxt` is free on this Mac (no binary, alias, function or Homebrew formula with that name), and no GitHub project uses it.
  - Rejected: `rdoc` (Ruby already uses it), `rd` (a common `rmdir` alias), `quill` (a Homebrew formula with that name exists), and `rxd` (also free, but `rdxt` was preferred).
  - Still to check: Linux hosts.

## Principles

1. **Text, not pictures.** Every word is a real terminal character, so it uses the terminal's font and size and zooms with ⌘+. It also works with selection, copy and the terminal's own scrolling. Images appear only where the document has an image.
2. **The terminal sets the colors.** Use only the terminal's 16-color palette plus its default foreground and background, so it matches Rex and any other terminal theme, light or dark. No hardcoded theme.
3. **Never lose your place.** Switching modes, reloading, resizing and going Back all keep the same spot in the document.
4. **Show each thing once.** There's one status line (file, mode, position), and it's quiet.
5. **It stays until you close it.** No popups that close by themselves (the lesson from Tern).

## Modes

There are exactly two modes: **View mode** and **Source mode**. Source mode is always editable; there's no separate edit mode.

| Mode | What it is | How you get there |
|---|---|---|
| **View** (default) | The rendered document, with links you can follow | Start-up, or `Esc` from Source |
| **Source** | The raw Markdown with syntax colors, **always editable** | `e` from View |
| *(Neovim)* | The same file and line in real Neovim, for heavy edits. This is a handoff, not a mode. | `E` from either mode |

**`Esc` always goes one level back toward View:**

```
View ──e──▶ Source (normal) ──i──▶ Source (insert: typing)
  ▲               │   ▲                    │
  └──────Esc──────┘   └────────Esc─────────┘
```

- `Tab` is **not** used to switch modes. In Source mode it indents and inserts tabs, as you'd expect.
- `e` ("edit") is free in View mode because nothing is editable there. `Esc` in vim normal mode normally does nothing, so it's free to mean "back to View".
- Source mode is vim-like by default: normal and insert modes; `hjkl`, `w/b`, `0/$`, `gg/G`; `dd`, `yy`, `p`; `u` and `Ctrl-R` for undo and redo; `:w`, `:q`. Arrow keys also work.
- **Modeless option:** `rdxt config set editing modeless`. Source mode then opens ready for typing, `Ctrl-S` saves, and `Esc` goes straight back to View.
- Switching modes keeps your place, using the source positions from the parser.
- Unsaved changes are shown in the status line. Going back to View renders the buffer you're editing, not the file on disk, so you can check an edit before saving.
- In Rex, `E` opens Neovim in a split. Elsewhere it runs `$EDITOR` in the same pane. The view reloads when you come back.
- Any non-Markdown text file opens straight into Source mode, which covers quick edits like the Tern `.bashrc` case.

## What Markdown it supports

Target: everything **Software Mansion's enriched-markdown** supports, so the same files render well in both.

- **CommonMark** plus **GFM**: tables, task lists, strikethrough, autolinks and footnotes
- **GitHub alerts**: `> [!NOTE]`, `[!TIP]`, `[!IMPORTANT]`, `[!WARNING]`, `[!CAUTION]`
- **YAML frontmatter**, **hidden by default in View mode.** `m` shows it as a card at the top: a rounded box titled "frontmatter", with keys in a column and values beside them (lists joined with commas). When it's hidden, the status line shows `m frontmatter` as a hint. Most editors show frontmatter badly in their preview, so it should look deliberate. `rdxt -p` always prints it.
- **Math**: inline and block, with a best-effort Unicode version in the view and always shown exactly in Source
- **`:::` directives** (the generic directive syntax that T3 Code, MyST and Comark use). These are also the extension point for later features.

### Dok

`.dok` files ([Dok](https://github.com/travelingflwr/dok) format) render too, chosen by the file's extension, with no option needed. Dok is meant to read well as plain text, so rdxt keeps its line structure:
- **Section headers** (ALL CAPS on their own line) are styled like H2 and appear in the outline.
- **`--` comment lines** are hidden.
- **List items** (`--`, `—`, `-`) get bullets.
- **Two-column rows** ("Term   Description") keep their alignment, with the term in bold, and long descriptions wrap under the description column.
- **Backticks** are inline code.
- **Long lines** wrap with a hanging indent. Nothing else is reflowed.

## How each element renders

| Element | View mode |
|---|---|
| H1/H2 | H1 is always in capitals. A frontmatter `title` is shown as the H1 when the document has none. Bigger text where the terminal supports kitty text sizing (OSC 66, detected at start-up). Otherwise bold, colored, with a rule under H1. |
| H3–H6 | Weight and color steps |
| Paragraphs | Wrapped to the reading width, never broken mid-word |
| Lists | Hanging indents. Task lists show round brackets: a bold `( )` when open, and a bold `(✔)` with a green heavy check when done. A finished task's text is dimmed. **`Space` toggles one and saves the file.** |
| Blockquotes | A colored left bar `▎` |
| Alerts | A left bar in the alert's color, an icon and a title row |
| Code blocks | A shaded background, the language label at top right, Chroma highlighting. In narrow panes, a long line scrolls sideways instead of wrapping. |
| Tables | Box-drawing borders and column alignment. In narrow panes, each row is shown as a stacked card. |
| Links | Underlined and colored, and clickable in other terminals through OSC 8 hyperlinks |
| Images | Kitty graphics where supported. Otherwise `🖼 alt text` as a link. |
| Rules | A thin full-width line |

### Easter egg: `:rex:`

`:rex:` always shows the Rex icon, **in both modes**.
- **View mode:** the Rex app icon drawn inline with kitty graphics, 2 cells wide. Rex supports kitty graphics (confirmed by the probe). Other terminals get 🦖.
- **Source mode:** the icon is shown in place of the text `:rex:` (like vim's conceal), except on the line the cursor is on. There it shows the raw `:rex:` so you can edit it.
- The Rex icon is Superlogical's artwork, so **it is not committed to this repo**. A local build step (`make rex-icon`) takes it from an installed Rex (`/Applications/Rex Beta.app/Contents/Resources/App Icon.icns`) and makes a small PNG in an ignored folder. Builds without it fall back to 🦖. We can ask Superlogical whether public builds may include it.
- Maybe later: the Discord `:rexie_*:` emoji (like `:rexie_starstruck:` from the changelog).

## Layout by width

| Pane width | Layout |
|---|---|
| < 60 cols (iPad split view, phone) | Compact: 1-column margins, tables as cards, small alert headers |
| 60–140 cols | Normal: reading width of about 88 columns, centered |
| > 140 cols | Wide: an outline sidebar with the headings, highlighting where you are |

Layout is redone on resize, and your place stays put.

## Navigation

- **Scrolling:** `j/k`, `Ctrl-d/u`, `Space/b`, `g/G`, plus the mouse wheel and touch (through the terminal's mouse reporting)
- **Headings:** `]]` / `[[` jump to the next or previous heading. `o` opens the outline.
- **Search:** `/`, then `n` / `N`
- **Links:**
  - In View mode, **`Tab` / `Shift-Tab` move between links** like a web browser, and **`Enter` follows** the selected one.
  - **`f` shows a hint letter on every link on screen** (like Vimium), and pressing a letter follows that link. Mouse clicks also work.
  - A `.md` link opens **in the same pane**, with history. `Ctrl-o` / `Ctrl-i` (or `H` / `L`) go back and forward.
  - A `#anchor` link jumps to that heading.
  - Other files open in a new Rex pane (or with `open` / `xdg-open` elsewhere).
  - URLs open in the system browser.
- **Live reload:** the view updates when the file changes on disk, which suits watching an agent write a report.

## Using it with Rex

Detected from `REX_SESSION`:

- `rdxt file.md` renders in the current pane.
- `rdxt open file.md` opens it in a new split beside you (useful for agents, and could replace md-viewer).
- `rdxt open --tab file.md` opens it in its own tab.
- An agent skill, `rdxt agent-guide`, like the one md-viewer has.

Outside Rex, everything except the split and tab commands works the same.

## Technology

- **Go**, as a single static binary for darwin and linux on arm64 and amd64, the same as rexpad.
- **goldmark** for parsing (CommonMark, GFM, extension points), **Chroma** for code highlighting, **Bubble Tea** for the event loop.
- **Our own layout engine:** goldmark's syntax tree is turned into lines for the current width, and every styled piece of text keeps its **source position**. That enables keeping your place across mode switches, search, and accurate jumps to the right line in Neovim.
- **Fenced-block renderer registry:** each language tag can have its own renderer. v1 has only `code`. Later: `mermaid` and `sequence` (ladder charts).

## Plan

1. **View mode:** parsing, layout, colors, width breakpoints, scrolling
2. **Navigation:** links, history, anchors, outline, search, link hints
3. **Source mode:** the vim-like editor, saving, keeping your place across modes, live reload, `E` to Neovim
4. **Rex integration**, installed with `rdxt setup`:
   - **Floating windows:** `rdxt open --float FILE` opens rdxt in a floating layer (`session.new_layer`), which closes when you quit.
   - **Cmd-click to open:** a small handler registered as the default app for `.md` and `.dok` opens Cmd-clicked paths in a floating rdxt. This also works for agents' `file://` links. Paths on remote hosts open with rdxt on that host.
   - **Rex's own UI for navigation:** a Lua file defines actions that use `rex.ui.pick` and `rex.ui.input`. rdxt calls them with `rex do`, so the outline, a link picker, search, a file picker and confirmations use Rex's native palette, which is touch-friendly on iPad. Without the actions (or outside Rex), rdxt uses its own versions.
   - A palette action, "rdxt: Open in floating window", plus `open`, splits and tabs, the agent guide, and the OSC 66 check.
5. **Packaging:** builds for each platform, installing on Linux hosts

**v2 ideas:** Mermaid and sequence diagrams drawn as text (with `mmdc` as a fallback), CSV and JSON views, piping into `rdxt` through stdin, streaming.

## Side project 1: working on a document with an agent

Turn Source mode into a place where you and an agent work on the same document. Rough ideas, not part of v1:

- **Review actions in View mode:** approve, or request changes, with the result sent back to the agent that's waiting (for example, through the exit code and a short note on stdout, or through a message).
- **Inline comments:** select a passage and leave a note for the agent, like a review comment on a pull request.
- **Live agent edits:** the agent's changes appear as they're written (live reload already covers part of this), highlighted until you accept or reject them.

v1 shouldn't block any of this: keep the source positions and the reload path clean.

## Decided

- **No settings while in development.** rdxt is 100% opinionated: one look, no themes. Themes may come later.
- **Status line:** has one blank line of padding above it.
- **Asking for stars:** the README has a star badge and a short ask. In the app, the ask appears in `--help` and `--version`, and once in the status line on the 10th launch, for 8 seconds. Never again after that.
- **Task markers:** round brackets, `( )` and `(✔)`, with finished tasks dimmed. Other styles may be added later.

- **Mode keys:** `e` goes from View to Source, and `Esc` goes back. Not `Tab`.
- **Source editing:** vim-like by default, with a modeless setting.

## What Rex supports (probe results, 2026-11-19)

Rex 0.1.0-preview.21 on macOS, using `scripts/probe.py` and `scripts/mouse-probe.py`:

- **Kitty graphics, kitty keyboard protocol, synchronized output, truecolor:** supported
- **OSC 66 text sizing:** not supported, so headings use the fallback style. Detect it at start-up anyway, in case Rex adds it.
- **OSC 10/11 color queries:** no reply. The 16-color palette approach doesn't need them.
- **Mouse:** clicks, Option-clicks, drags and scroll events (including horizontal) reach programs. **Shift-click and Cmd-click don't**: Rex keeps them for itself. Touch swipes on iPad most likely arrive as scroll events.
- **Block types:** `rex block types` lists only Shell, two tunnel types and Onboarding. There's no document block type yet.

## Open questions

- **Text selection for copying:** tabled for later. Rex keeps Shift-click for itself, so Shift-drag may already select text while rdxt has the mouse.
- **"Silent" edits in View mode:** should interactive elements change the file directly from View mode? Toggling a task checkbox is the first example. Others could be ticking off a review item or editing a frontmatter value. If yes, each one needs a visible flash ("saved") and an undo.
- **Resizing on iPad:** Rex on iPad doesn't seem to send SIGWINCH, so rdxt now also checks the size every 400 ms.
