# rex-doc

[![GitHub stars](https://img.shields.io/github/stars/travelingflwr/rex-doc?style=social)](https://github.com/travelingflwr/rex-doc/stargazers)

**Made for [Rex](https://superlogical.com).** The best way to read Markdown in a terminal. It works in any terminal, but it's designed around Rex: its panes, its floating windows and its palette.

`rdxt` ("read text") opens a Markdown file as real terminal text, not a picture. So it uses your terminal's font, size and theme, zooms with ⌘+, and works with selection and scrolling, including on the iPad.

## Install

```sh
go install github.com/travelingflwr/rex-doc/cmd/rdxt@latest
rdxt setup
```

`rdxt setup` makes rdxt feel built into Rex:
- **Cmd-click** a `.md` or `.dok` path in Rex, or double-click one in Finder, and it opens in a floating rdxt window.
- **The Rex palette** gets *rdxt: Open Document…* and *rdxt: New Document…* (with Blank, Plan, Notes and Dok templates).
- **Coding agents** (Claude Code, Codex, Cursor, OpenCode, Pi) get an rdxt skill, so they open the plans and reports they write.
- **zsh** reports its folder to Rex, so relative names are clickable, and typing `notes.md` + Enter opens it.

`rdxt setup --undo` puts everything back, including the app that opened Markdown before.

## Use

```sh
rdxt notes.md                    # read it here
rdxt open --float notes.md       # in a floating Rex window (--split, --tab)
rdxt new --template plan fix-login
```

Press `?` in rdxt for every key.

## The idea

- **View mode** (the default): the rendered document. Links to other `.md` files open in place, with Back and Forward.
- **Source mode** (`e`): the raw Markdown, always editable, with vim-like keys. `Esc` goes back to View.
- **Built for reviewing what agents write:** it opens fast and starts in View mode.
- **Adapts to the pane:** compact in narrow panes (phone, iPad split view), with an outline sidebar in wide ones.
- **Supports** CommonMark, GFM, GitHub alerts, frontmatter, math and `:::` directives.

## ⭐ Like it? Star it

If rdxt makes reading Markdown in your terminal better, **please star the repo**. Stars are how other people find tools like this, and they tell us it's worth building more.

## Contributing

rdxt is deliberately opinionated. Please read [CONTRIBUTING.md](CONTRIBUTING.md) before sending a pull request. It explains the design decisions and why pull requests that go against them are declined.

## License

MIT
