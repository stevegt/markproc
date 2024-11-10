package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/stevegt/fuzzy"
)

type Reference struct {
	Name     string
	Line     int
	Resolved bool
}

type Target struct {
	Name         string
	Heading      string
	Number       string
	HeadingLower string
}

type References []Reference
type Targets []Target

var (
	refRegexp     = regexp.MustCompile(`\[(\w+)\][^:]`)
	extLinkRegexp = regexp.MustCompile(`^\[(\w+)\]:\s+`)
	headerRegexp  = regexp.MustCompile(`^(#+)\s+(.+)`)
)

func main() {
	references := References{}
	targets := Targets{}
	scanner := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	lines := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	lines = passFindRefs(lines, &references, writer)
	lines = passMkExterns(lines, &targets, writer)
	lines = passMkHeads(lines, &targets, writer)
	lines = passLinkExterns(lines, &references, &targets)
	lines = passLinkHeads(lines, &references, &targets, writer)

	for _, line := range lines {
		writer.WriteString(line + "\n")
	}
	writer.Flush()
}

func generateSectionNumber(level int, index int, parentNumber string) string {
	if parentNumber == "" {
		return fmt.Sprintf("%d", index+1)
	}
	return fmt.Sprintf("%s.%d", parentNumber, index+1)
}

func passFindRefs(lines []string, references *References, writer *bufio.Writer) []string {
	newLines := []string{}
	for i, line := range lines {
		if refMatch := refRegexp.FindAllStringSubmatch(line, -1); len(refMatch) > 0 {
			for _, match := range refMatch {
				ref := match[1]
				*references = append(*references, Reference{Name: ref, Line: i})
				line = strings.Replace(line, fmt.Sprintf("[%s]", ref), fmt.Sprintf("[%s](#%s)", ref, ref), -1)
			}
		}
		newLines = append(newLines, line)
		writer.WriteString(line + "\n")
	}
	return newLines
}

func passMkExterns(lines []string, targets *Targets, writer *bufio.Writer) []string {
	newLines := []string{}
	for _, line := range lines {
		if extMatch := extLinkRegexp.FindStringSubmatch(line); len(extMatch) > 0 {
			ref := extMatch[1]
			*targets = append(*targets, Target{Name: ref, Heading: line, HeadingLower: strings.ToLower(line)})
			writer.WriteString(fmt.Sprintf(`<a name="%s"></a>`, ref) + "\n")
		}
		newLines = append(newLines, line)
	}
	return newLines
}

func passMkHeads(lines []string, targets *Targets, writer *bufio.Writer) []string {
	hash := sha256.New()
	for _, line := range lines {
		hash.Write([]byte(line))
	}

	newLines := []string{}
	sectionNumbers := []int{0, 0, 0, 0, 0} // Support for up to 5 levels of headings
	for _, line := range lines {
		if headerMatch := headerRegexp.FindStringSubmatch(line); len(headerMatch) > 0 {
			level := len(headerMatch[1])
			sectionNumbers[level-1]++
			for i := level; i < 5; i++ {
				sectionNumbers[i] = 0
			}
			var parentNumber string
			if level > 1 {
				parentNumber = fmt.Sprintf("%d", sectionNumbers[level-2])
			}
			sectionNumber := generateSectionNumber(level, sectionNumbers[level-1]-1, parentNumber)
			headerName := headerMatch[2]
			headerLink := fmt.Sprintf("sec%s", strings.Replace(sectionNumber, ".", "_", -1))
			*targets = append(*targets, Target{Name: headerLink, Heading: headerName, HeadingLower: strings.ToLower(headerName), Number: sectionNumber})

			writer.WriteString(fmt.Sprintf(`<a name="%s"></a>`, headerLink) + "\n")
			line = fmt.Sprintf("%s %s. %s", headerMatch[1], sectionNumber, headerName)
		}
		newLines = append(newLines, line)
		writer.WriteString(line + "\n")
	}
	return newLines
}

func passLinkExterns(lines []string, references *References, targets *Targets) []string {
	for i, ref := range *references {
		for _, target := range *targets {
			if ref.Name == target.Name {
				(*references)[i].Resolved = true
				break
			}
		}
	}
	return lines
}

func passLinkHeads(lines []string, references *References, targets *Targets, writer *bufio.Writer) (newLines []string) {
	loweredTargets := map[string]string{}
	for _, target := range *targets {
		loweredTargets[target.HeadingLower] = target.Name
	}

	// Copy lines to newLines
	newLines = append([]string(nil), lines...)

	for i, ref := range *references {
		if ref.Resolved {
			continue
		}
		matches := fuzzy.Match(strings.ToLower(ref.Name), keys(loweredTargets))
		insertionOnly := []string{}
		for _, match := range matches {
			if match.Insertions > 0 && match.Substitutions == 0 && match.Deletions == 0 {
				insertionOnly = append(insertionOnly, match.Original)
			}
		}
		switch len(insertionOnly) {
		case 0:
			fmt.Fprintf(os.Stderr, "Warning: No matches for unresolved reference [%s]\n", ref.Name)
		case 1:
			resolvedName := loweredTargets[insertionOnly[0]]
			// find the target
			var target Target
			for _, t := range *targets {
				if target.Name == resolvedName {
					target = t
					break
				}
			}

			// rewrite the reference in newLines
			line := newLines[ref.Line]
			linkContent := fmt.Sprintf("sec %s", target.Number)
			link := fmt.Sprintf("[%s](#%s)", linkContent, resolvedName)
			refStr := fmt.Sprintf("[%s]", link)
			line = strings.Replace(line, fmt.Sprintf("[%s]", ref.Name), refStr, -1)
			ref.Name = resolvedName
			newLines[ref.Line] = line

			// Mark ref as resolved
			(*references)[i].Resolved = true
		default:
			fmt.Fprintf(os.Stderr, "Warning: Multiple matches for unresolved reference [%s]\n", ref.Name)
		}
	}

	return newLines
}

func keys(m map[string]string) []string {
	s := make([]string, 0, len(m))
	for key := range m {
		s = append(s, key)
	}
	return s
}