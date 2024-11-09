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
# Section One

This is the first section.

## Section Two

This is a reference to the Section One heading [secone].

Reference to the anchor below [ref1].

## References

[ref1]: A bibliographic reference.
```

#### Output

```markdown
<a name="secone"></a>
# Section One [secone]

This is the first section.

<a name="01ba4719c80b6fe911b091a7c05124b64eeece964e09c058ef8f9805daca546"></a>
## Section Two

This is a reference to the Section One heading [SECONE].

<a name="ref1"></a>
Reference to the anchor below [ref1].

## References

[ref1]: A bibliographic reference.
```

