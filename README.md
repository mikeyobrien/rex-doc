# rex-doc

[![GitHub stars](https://img.shields.io/github/stars/travelingflwr/rex-doc?style=social)](https://github.com/travelingflwr/rex-doc/stargazers)

**The best way to read Markdown in a terminal.** It works in any terminal and looks best in [Rex](https://superlogical.com).

`rdxt` ("read text") opens a Markdown file as real terminal text, not a picture. So it uses your terminal's font, size and theme, zooms with ⌘+, and works with selection and scrolling, including on the iPad.

> **Status:** early design. Nothing to install yet. See [DESIGN.md](DESIGN.md).

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
