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

	lines = passMkExterns(lines)
	lines = passMkHeads(lines)
	lines = passLinkExterns(lines)
	lines = passLinkHeads(lines)

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

func passLinkExterns(lines []string) []string {
	newLines := []string{}
	for _, line := range lines {
		if refMatch := refRegexp.FindAllStringSubmatch(line, -1); len(refMatch) > 0 {
			for _, match := range refMatch {
				ref := match[1]
				// use an HTML link, not a markdown link
				link := fmt.Sprintf(`<a href="#%s">%s</a>`, ref, ref)
				oldStr := fmt.Sprintf("[%s]", ref)
				newStr := fmt.Sprintf("[%s]", link)
				line = strings.Replace(line, oldStr, newStr, -1)
			}
		}
		newLines = append(newLines, line)
	}
	return newLines
}

func passMkExterns(lines []string) []string {
	newLines := []string{}
	targets := Targets{}
	for _, line := range lines {
		if extMatch := extLinkRegexp.FindStringSubmatch(line); len(extMatch) > 0 {
			ref := extMatch[1]
			targets = append(targets, Target{Name: ref, Heading: line, HeadingLower: strings.ToLower(line)})
			line = fmt.Sprintf(`<a name="%s"></a>`, ref) + line
		}
		newLines = append(newLines, line)
	}
	return newLines
}

func passMkHeads(lines []string) []string {
	hash := sha256.New()
	for _, line := range lines {
		hash.Write([]byte(line))
	}

	newLines := []string{}
	sectionNumbers := []int{0, 0, 0, 0, 0} // Support for up to 5 levels of headings
	targets := Targets{}
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
			targets = append(targets, Target{Name: headerLink, Heading: headerName, HeadingLower: strings.ToLower(headerName), Number: sectionNumber})

			line = fmt.Sprintf(`<a name="%s"></a>%s %s. %s`, headerLink, headerMatch[1], sectionNumber, headerName)
		}
		newLines = append(newLines, line)
	}
	return newLines
}

func passLinkHeads(lines []string) (newLines []string) {
	newLines = append(newLines, lines...)
	references := References{}
	targets := Targets{}
	loweredTargets := map[string]string{}
	for _, target := range targets {
		loweredTargets[target.HeadingLower] = target.Name
	}

	for i, ref := range references {
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
			newLines = append(newLines, lines[ref.Line])
		case 1:
			resolvedName := loweredTargets[insertionOnly[0]]
			// find the target
			var target Target
			for _, t := range targets {
				if t.Name == resolvedName {
					target = t
					break
				}
			}

			// rewrite the reference in newLines
			line := lines[ref.Line]
			linkContent := fmt.Sprintf("sec %s", target.Number)
			link := fmt.Sprintf("[%s](#%s)", linkContent, resolvedName)
			refStr := fmt.Sprintf("[%s]", link)
			line = strings.Replace(line, fmt.Sprintf("[%s]", ref.Name), refStr, -1)
			ref.Name = resolvedName
			newLines = append(newLines, line)

			// Mark ref as resolved
			references[i].Resolved = true
		default:
			fmt.Fprintf(os.Stderr, "Warning: Multiple matches for unresolved reference [%s]\n", ref.Name)
			newLines = append(newLines, lines[ref.Line])
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
