rdxt shows Markdown (.md) and Dok (.dok) documents in Rex: rendered as real
text, with links that open other documents, and an editor. Made for Rex.

When to open a document:
- After you write or substantially update a document the user should read:
  a plan, report, design doc, review, README, meeting notes.
- When the user asks to open, show, preview or read a document.

When not to:
- Files you only read, scratch notes, or many files at once. Open one
  document per deliverable.

How:
- `rdxt open /absolute/path/to/file.md` opens it in a pane beside yours and
  keeps the focus on you. It prints the path when it worked.
- `rdxt open --float FILE` opens it in a floating window over the user's
  tab instead, for a quick look. `--tab` opens a new tab.
- `rdxt new --template plan|notes|dok|blank --dir DIR NAME --no-open`
  creates a document from a template and prints its path. Fill it in, then
  `rdxt open` it.
- rdxt reloads a document when the file changes, so after you edit an open
  document the user sees the change. Don't open it a second time.
- Always print the document's absolute path in your reply as well.

Writing for rdxt:
- Standard CommonMark and GitHub Flavored Markdown: tables, task lists
  (`- [ ]`), footnotes, and alerts (`> [!NOTE]`, `> [!WARNING]`, …).
- A frontmatter `title:` is shown as the document's title when there's no
  `# H1`.
- An image on its own line is shown as a figure with its alt text as the
  caption; write good alt text.
- Link related documents with relative links (`[plan](plan.md)`); the
  user can follow them inside rdxt.
- Task lists are tickable: the user can tick tasks in the document, and the
  file is saved.

The user may be editing a document in rdxt while you work. If you change a
file that's open, rdxt keeps any unsaved edits of theirs and tells them.

If rdxt or Rex is unavailable, mention the absolute path and continue. Do
not install anything.
