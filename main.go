package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/stevegt/fuzzy"
	. "github.com/stevegt/goadapt"
)

type Target struct {
	Name         string
	Heading      string
	Number       string
	HeadingLower string
}

var (
	exitCode         = 0
	refRegexp        = regexp.MustCompile(`\[(\w+)\][^:]`)
	extLinkRegexp    = regexp.MustCompile(`^\[(\w+)\]:\s+`)
	headerRegexp     = regexp.MustCompile(`^(#+)\s+(.+)`)
	numberedHeaderRe = regexp.MustCompile(`^(#+)\s+([\d\.]+)\s+(.+)`)
	sectionRefRegexp = regexp.MustCompile(`\[sec\s+([^\]]+)\]`)
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
	err := verify(lines)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Verification error: %v\n", err)
		exitCode = 1
	}

	for _, line := range lines {
		_, err = writer.WriteString(line + "\n")
		Ck(err)
	}
	err = writer.Flush()
	Ck(err)

	os.Exit(exitCode)
}

func generateSectionNumber(level int, number int, parentNumber string) string {
	if parentNumber == "" {
		return fmt.Sprintf("%d", number)
	}
	return fmt.Sprintf("%s.%d", parentNumber, number)
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
	for _, line := range lines {
		if extMatch := extLinkRegexp.FindStringSubmatch(line); len(extMatch) > 0 {
			ref := extMatch[1]
			// insert the anchor link before the reference
			newLine := fmt.Sprintf(`<a name="%s"></a>`, ref)
			newLines = append(newLines, newLine)
		}
		newLines = append(newLines, line)
	}
	return newLines
}

func passMkHeads(lines []string) []string {
	newLines := []string{}
	sectionNumbers := []int{}

	prevLevel := 0
	for _, line := range lines {
		if headerMatch := headerRegexp.FindStringSubmatch(line); len(headerMatch) > 0 {
			level := len(headerMatch[1])
			title := headerMatch[2]

			if level-prevLevel > 1 {
				fmt.Fprintf(os.Stderr, "Warning: Header level gap up: %s\n", title)
			}
			prevLevel = level

			// Extend sectionNumbers slice if current level exceeds its length
			for len(sectionNumbers) < level {
				sectionNumbers = append(sectionNumbers, 0)
			}

			// Increment the current level's count
			sectionNumbers[level-1]++

			// Reset counts for deeper levels
			for i := level; i < len(sectionNumbers); i++ {
				sectionNumbers[i] = 0
			}

			// Build the section number string
			sectionNumberParts := []string{}
			for i := 0; i < level; i++ {
				sectionNumberParts = append(sectionNumberParts, fmt.Sprintf("%d", sectionNumbers[i]))
			}
			sectionNumber := strings.Join(sectionNumberParts, ".")

			// Generate the anchor link
			headerLink := fmt.Sprintf("sec%s", strings.Replace(sectionNumber, ".", "_", -1))

			// Insert the anchor link before the header
			newLines = append(newLines, fmt.Sprintf(`<a name="%s"></a>`, headerLink))

			// Insert the section number after the header hashes
			line = fmt.Sprintf("%s %s. %s", headerMatch[1], sectionNumber, title)
		}
		newLines = append(newLines, line)
	}
	return newLines
}

func passLinkHeads(lines []string) []string {
	newLines := []string{}
	sectionTargets := map[string]Target{}

	for _, line := range lines {
		if headerMatch := numberedHeaderRe.FindStringSubmatch(line); len(headerMatch) > 0 {
			number := headerMatch[2]
			number = strings.TrimSuffix(number, ".")
			text := headerMatch[3]
			lowerText := strings.ToLower(text)
			numStr := strings.Replace(number, ".", "_", -1)
			name := fmt.Sprintf("sec%s", numStr)
			sectionTargets[lowerText] = Target{Name: name, Heading: text, Number: number, HeadingLower: lowerText}
		}
	}

	for _, line := range lines {
		if secRefMatches := sectionRefRegexp.FindAllStringSubmatch(line, -1); secRefMatches != nil {
			for _, match := range secRefMatches {
				acronym := match[1]
				lowerAcronym := strings.ToLower(acronym)
				fuzzyMatches := fuzzy.Match(lowerAcronym, keys(sectionTargets))
				insertionOnly := []fuzzy.MatchResult{}
				for _, fm := range fuzzyMatches {
					if fm.Insertions > 0 && fm.Substitutions == 0 && fm.Deletions == 0 {
						insertionOnly = append(insertionOnly, fm)
					}
				}

				switch len(insertionOnly) {
				case 0:
					fmt.Fprintf(os.Stderr, "Warning: [sec %s] no fuzzy match found\n", acronym)
					exitCode = 1
				case 1:
					target := sectionTargets[insertionOnly[0].Original]
					anchorLink := fmt.Sprintf(`<a href="#%s">sec %s</a>`, target.Name, target.Number)
					oldStr := fmt.Sprintf("[sec %s]", acronym)
					newStr := fmt.Sprintf("[%s]", anchorLink)
					line = strings.Replace(line, oldStr, newStr, -1)
				default:
					fmt.Fprintf(os.Stderr, "Warning: [sec %s] multiple fuzzy matches found:\n", acronym)
					for _, fm := range insertionOnly {
						fmt.Fprintf(os.Stderr, "  %s\n", sectionTargets[fm.Original].Heading)
					}
					exitCode = 1
				}
			}
		}
		newLines = append(newLines, line)
	}

	return newLines
}

func verify(lines []string) (err error) {
	links := make(map[string]bool)
	duplicateChecker := make(map[string]bool)

	// Collect all anchor names
	for _, line := range lines {
		if nameMatch := regexp.MustCompile(`<a name="([^"]+)"></a>`).FindStringSubmatch(line); len(nameMatch) > 0 {
			anchorName := nameMatch[1]
			if _, exists := duplicateChecker[anchorName]; exists {
				err = fmt.Errorf("Duplicate target found: #%s", anchorName)
				exitCode = 1
				return
			} else {
				duplicateChecker[anchorName] = true
			}
		}
	}

	// Collect all hrefs
	for _, line := range lines {
		if linkMatch := regexp.MustCompile(`<a href="#([^"]+)">`).FindStringSubmatch(line); len(linkMatch) > 0 {
			linkName := linkMatch[1]
			links[linkName] = true
		}
	}

	// Verify all links point to a valid target
	for link := range links {
		if _, exists := duplicateChecker[link]; !exists {
			err = fmt.Errorf("Link points to an undefined target: #%s", link)
			exitCode = 1
			return
		}
	}
	return
}

func keys(m map[string]Target) []string {
	s := make([]string, 0, len(m))
	for key := range m {
		s = append(s, key)
	}
	return s
}
