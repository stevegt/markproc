package main

import (
	"bytes"
	"os/exec"
	"reflect"
	"testing"
)

func TestPassFindRefs(t *testing.T) {
	lines := []string{
		"This is a [reference] to something.",
		"No refs here.",
	}
	expectedLines := []string{
		"This is a [reference](#reference) to something.",
		"No refs here.",
	}

	result := passFindRefs(lines)
	if !reflect.DeepEqual(result, expectedLines) {
		t.Errorf("passFindRefs failed:\nexpected: %v\ngot: %v", expectedLines, result)
	}
}

func TestPassMkExterns(t *testing.T) {
	lines := []string{
		"[ref1]: A bibliographic reference.",
		"No externs here.",
	}
	expectedLines := []string{
		`<a name="ref1"></a>[ref1]: A bibliographic reference.`,
		"No externs here.",
	}

	result := passMkExterns(lines)
	if !reflect.DeepEqual(result, expectedLines) {
		t.Errorf("passMkExterns failed:\nexpected: %v\ngot: %v", expectedLines, result)
	}
}

func TestPassMkHeads(t *testing.T) {
	lines := []string{
		"# Top-Level Header",
		"This is a paragraph.",
		"## Sub-Level Header",
	}
	expectedLines := []string{
		`<a name="sec1"></a># 1. Top-Level Header`,
		"This is a paragraph.",
		`<a name="sec1_1"></a>## 1.1. Sub-Level Header`,
	}

	result := passMkHeads(lines)
	if !reflect.DeepEqual(result, expectedLines) {
		t.Errorf("passMkHeads failed:\nwant: %v\nhave: %v", expectedLines, result)
	}
}

func TestMarkdownPreprocessor(t *testing.T) {
	input := `# A Top-Level Header

This is the first section.

## A Sub-Level Header

This is a reference to the Section One heading [sec top].

Reference to the anchor below [ref1].

## References

[ref1]: A bibliographic reference.`

	expectedOutput := `<a name="sec1"></a>
# 1. A Top-Level Header

This is the first section.

<a name="sec1_1"></a>
## 1.1 A Sub-Level Header

This is a reference to the Section One heading [<a href="#sec1">sec 1</a>].

Reference to the anchor below [<a href="#ref1">ref1</a>].

## References

<a name="ref1"></a>
[ref1]: A bibliographic reference.`

	cmd := exec.Command("go", "run", "main.go")
	cmd.Stdin = bytes.NewReader([]byte(input))

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cmd.Run() failed with %s\n", err)
	}

	actualOutput := string(output)
	if actualOutput != expectedOutput {
		t.Errorf("Expected:\n%s\nGot:\n%s\n", expectedOutput, actualOutput)
	}
}
