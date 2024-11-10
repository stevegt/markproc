# Markdown Preprocessor

This project provides a Go-based preprocessor for Markdown files that
creates anchors and links for references and headings. The tool is a
command-line utility that reads a Markdown file from standard input
and writes the processed content to standard output.

## Features

- Creates anchor links for lines starting with `[REF]:`
- Converts `[REF]` references to links and validates them
- Tracks other references and attempts to link them to headings using fuzzy matching
- Prints warnings for references that cannot be conclusively matched

## Usage

To use the preprocessor, run it as a command in your terminal:

```bash
go run main.go < your_markdown_file.md > processed_markdown.md
```

Here, `your_markdown_file.md` is the Markdown file you want to process, and `processed_markdown.md` is the output file with processed content.

### Example

#### Input

```markdown
# A Top-Level Heading

This is the first section.

## A Section Heading

This is a reference to the Section One heading [sec top].

Reference to the anchor below [ref1].

## References

[ref1]: A bibliographic reference.
```

#### Output

```markdown
<a name="sec1"></a>
# 1. A Top-Level Heading

This is the first section.

<a name="sec1_1"></a>
## 1.1 A Section Heading

This is a reference to the Section One heading [<a href="#sec1">sec 1</a>].

Reference to the anchor below [<a href="#ref1">ref1</a>].

## References

<a name="ref1"></a>
[ref1]: A bibliographic reference.
```

In this example:

- Internal section references use section numbers, allowing them to be easily distinguished from external references.
- Each section heading gets a unique numeric section identifier and an associated anchor.

