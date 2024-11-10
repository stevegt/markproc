package main

import (
	"bytes"
	"os/exec"
	"testing"
)

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
