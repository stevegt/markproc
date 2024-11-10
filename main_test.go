package main

import (
	"bytes"
	"math/rand"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"

	. "github.com/stevegt/goadapt"
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

func TestVerify(t *testing.T) {
	lines := []string{
		`<a name="sec1"></a>`,
		`# 1. Title`,
		`<a href="#sec1">link to title</a>`,
		`<a href="#missing">link to missing</a>`,
		`<a name="sec1"></a>`,
	}

	err := verify(lines)
	Tassert(t, err != nil, "verify did not catch any errors")

	lines = []string{
		`<a name="sec1"></a>`,
		`# 1. Title`,
		`<a href="#sec1">link to title</a>`,
		`<a href="http://example.com">link to example</a>`,
	}

	err = verify(lines)
	Tassert(t, err == nil, "verify failed: %v", err)
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

func TestComplexSectionStructure(t *testing.T) {
	rand.Seed(1)

	createSection := func(level int, index int) string {
		return strings.Repeat("#", level) + " Section " + strconv.Itoa(level) + "." + strconv.Itoa(index)
	}

	sections := []string{}
	// Generate 10 sections with at least 10 subsections each
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			level := rand.Intn(5) + 1
			sections = append(sections, createSection(level, j))
		}
	}

	// Randomly shuffle the sections
	rand.Shuffle(len(sections), func(i, j int) {
		sections[i], sections[j] = sections[j], sections[i]
	})

	// Run the input through passMkHeads
	inlines := sections
	outlines := passMkHeads(inlines)

	expectedNums := generateExpectedSectionNumbers(inlines)

	// Check if the section numbers are correctly ordered
	index := 0
	for _, num := range expectedNums {
		if idx := findSection(outlines, num); idx != -1 {
			if idx < index {
				t.Errorf("section numbers are not correctly ordered for %s", num)
			}
			index = idx
		}
	}
}

// Helper function to generate expected section numbers based on input order
func generateExpectedSectionNumbers(lines []string) []string {
	sectionNumbers := make([]int, 5) // Support for up to 5 levels of headings
	numbers := []string{}
	for _, line := range lines {
		if headerMatch := headerRegexp.FindStringSubmatch(line); len(headerMatch) > 0 {
			level := len(headerMatch[1])
			sectionNumbers[level-1]++
			for i := level; i < 5; i++ {
				sectionNumbers[i] = 0
			}
			var parentNumber string
			if level > 1 {
				parentNumber = strconv.Itoa(sectionNumbers[level-2])
			}
			sectionNumber := generateSectionNumber(level, sectionNumbers[level-1]-1, parentNumber)
			numbers = append(numbers, sectionNumber)
		}
	}
	return numbers
}

// Helper function to find the section header in the output
func findSection(lines []string, sectionNumber string) int {
	for idx, line := range lines {
		if strings.Contains(line, sectionNumber) {
			return idx
		}
	}
	return -1
}
