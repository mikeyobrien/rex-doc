---
title: rdxt sample
status: draft
---

# rdxt sample document

This paragraph has **bold**, *italic*, ***both***, ~~strikethrough~~, `inline code`, a [link to the design](../DESIGN.md), an autolink https://example.com, and a footnote.[^1] It's long enough to wrap across a few lines so we can see how wrapping behaves at different widths. :rex: says hi.

## Lists

- First item
- Second item with a longer line that should wrap neatly under itself instead of under the bullet
  - Nested item
    - Deeper item
- Third item

1. One
2. Two
10. Ten (alignment check)

- [ ] An open task
- [x] A finished task

## Quotes and alerts

> A plain quote, which also wraps when the line is long enough to need it in a narrow pane.
>
> Second paragraph of the quote.

> [!NOTE]
> Useful information that users should know.

> [!WARNING]
> Critical content demanding immediate user attention.

## Code

```go
// main says hello.
func main() {
	fmt.Println("hello, rdxt", 42)
}
```

```
plain code block with a very long line that will certainly not fit in a narrow terminal pane at all
```

## Tables

| Mode | What it is | Key |
|:-----|:-----------|:---:|
| **View** | The rendered document, with links you can follow | `Esc` |
| **Source** | The raw Markdown, always editable | `e` |

---

### Heading 3
#### Heading 4
##### Heading 5
###### Heading 6

Hard line break here\
and the next line.

[^1]: The footnote text.
