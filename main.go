package main

import (
	"bufio"
	"crypto/sha256"
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
	Ck(err)

	for _, line := range lines {
		_, err = writer.WriteString(line + "\n")
		Ck(err)
	}
	err = writer.Flush()
	Ck(err)
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
			headerLink := fmt.Sprintf("sec%s", strings.Replace(sectionNumber, ".", "_", -1))

			// insert the anchor link before the header
			newLines = append(newLines, fmt.Sprintf(`<a name="%s"></a>`, headerLink))

			// insert the section number before the header
			line = fmt.Sprintf("%s %s. %s", headerMatch[1], sectionNumber, headerMatch[2])
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
			name := fmt.Sprintf("sec%s", strings.Replace(numStr, ".", "_", -1))
			sectionTargets[lowerText] = Target{Name: name, Heading: text, Number: number, HeadingLower: lowerText}
		}
	}

	// spew.Dump(sectionTargets)

	for _, line := range lines {
		if secRefMatches := sectionRefRegexp.FindAllStringSubmatch(line, -1); secRefMatches != nil {
			for _, match := range secRefMatches {
				acronym := match[1]
				lowerAcronym := strings.ToLower(acronym)
				fuzzyMatches := fuzzy.Match(lowerAcronym, keys(sectionTargets))
				insertionOnly := []fuzzy.MatchResult{}
				for _, match := range fuzzyMatches {
					if match.Insertions > 0 && match.Substitutions == 0 && match.Deletions == 0 {
						insertionOnly = append(insertionOnly, match)
					}
				}

				// spew.Dump(insertionOnly)

				switch len(insertionOnly) {
				case 0:
					fmt.Fprintf(os.Stderr, "Warning: [sec %s] no fuzzy match found\n", acronym)
				case 1:
					target := sectionTargets[insertionOnly[0].Original]
					// spew.Dump(target)
					anchorLink := fmt.Sprintf(`<a href="#%s">sec %s</a>`, target.Name, target.Number)
					// spew.Dump(anchorLink)
					oldStr := fmt.Sprintf("[sec %s]", acronym)
					newStr := fmt.Sprintf("[%s]", anchorLink)
					// spew.Dump(oldStr, newStr)
					line = strings.Replace(line, oldStr, newStr, -1)
				default:
					fmt.Fprintf(os.Stderr, "Warning: [sec %s] multiple fuzzy matches found:\n", acronym)
					for _, match := range insertionOnly {
						fmt.Fprintf(os.Stderr, "  %s\n", sectionTargets[match.Original].Heading)
					}
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
