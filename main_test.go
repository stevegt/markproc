package main

import (
	"bytes"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestPassMkExterns(t *testing.T) {
	lines := []string{
		"[ref1]: A bibliographic reference.",
		"No externs here.",
	}
	expectedLines := []string{
		`<a name="ref1"></a>`,
		`[ref1]: A bibliographic reference.`,
		"No externs here.",
	}

	result := passMkExterns(lines)
	if !reflect.DeepEqual(result, expectedLines) {
		t.Errorf("passMkExterns failed:\nwant: %v\nhave: %v", expectedLines, result)
	}
}

func TestPassMkHeads(t *testing.T) {
	lines := []string{
		"# Top-Level Header",
		"This is a paragraph.",
		"## Sub-Level Header",
	}
	expectedLines := []string{
		`<a name="sec1"></a>`,
		`# 1. Top-Level Header`,
		"This is a paragraph.",
		`<a name="sec1_1"></a>`,
		`## 1.1. Sub-Level Header`,
	}

	result := passMkHeads(lines)
	if !reflect.DeepEqual(result, expectedLines) {
		t.Errorf("passMkHeads failed:\nwant: %v\nhave: %v", expectedLines, result)
	}
}

func TestPassLinkExterns(t *testing.T) {
	lines := []string{
		"This is a [reference] to something.",
		"This is a [sec fooee] reference.",
		"No refs here.",
	}
	expectedLines := []string{
		"This is a [<a href=\"#reference\">reference</a>] to something.",
		"This is a [sec fooee] reference.",
		"No refs here.",
	}

	result := passLinkExterns(lines)
	if !reflect.DeepEqual(result, expectedLines) {
		t.Errorf("passLinkExterns failed:\nwant: %v\nhave: %v", expectedLines, result)
	}
}

func TestPassLinkHeads(t *testing.T) {
	lines := []string{
		"This is a [reference] to something.",
		"This is a [sec fooee] reference.",
		"No refs here.",
		`<a name="sec1"></a>`,
		`## 1. Title`,
		`<a name="sec2_3"></a>`,
		`## 2.3. Fun Object Overtone`,
		`<a name="sec7_9">`,
		`</a>## 7.9. Something`,
	}
	expectedLines := []string{
		"This is a [reference] to something.",
		"This is a [<a href=\"#sec2_3\">sec 2.3</a>] reference.",
		"No refs here.",
		`<a name="sec1"></a>`,
		`## 1. Title`,
		`<a name="sec2_3"></a>`,
		`## 2.3. Fun Object Overtone`,
		`<a name="sec7_9">`,
		`</a>## 7.9. Something`,
	}

	result := passLinkHeads(lines)
	if !reflect.DeepEqual(result, expectedLines) {
		t.Errorf("\nwant: %v\nhave: %v", expectedLines, result)
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
## 1.1. A Sub-Level Header

This is a reference to the Section One heading [<a href="#sec1">sec 1</a>].

Reference to the anchor below [<a href="#ref1">ref1</a>].

<a name="sec1_2"></a>
## 1.2. References

<a name="ref1"></a>
[ref1]: A bibliographic reference.`

	cmd := exec.Command("go", "run", "main.go")
	cmd.Stdin = bytes.NewReader([]byte(input))

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cmd.Run() failed with %s\n", err)
	}

	actualOutput := string(output)
	// line-by-line comparison
	wantLines := strings.Split(expectedOutput, "\n")
	haveLines := strings.Split(actualOutput, "\n")
	for i, want := range wantLines {
		if i >= len(haveLines) {
			t.Errorf("output is shorter than expected")
			break
		}
		if want != haveLines[i] {
			t.Errorf("line %d:\nwant: %q\nhave: %q", i, want, haveLines[i])
		}
	}
}
