package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
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

	docHash := hex.EncodeToString(hash.Sum(nil))
	newLines := []string{}
	for _, line := range lines {
		if headerMatch := headerRegexp.FindStringSubmatch(line); len(headerMatch) > 0 {
			headerName := headerMatch[2]
			headerLink := fmt.Sprintf("%s-%s", headerName, docHash)
			*targets = append(*targets, Target{Name: headerLink, Heading: headerName, HeadingLower: strings.ToLower(headerName)})

			writer.WriteString(fmt.Sprintf(`<a name="%s"></a>`, headerLink) + "\n")
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

	// copy lines to newLines
	newLines = lines[:]
	for _, ref := range *references {
		if !ref.Resolved {
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
				// XXX rename target.Name to ref.Name
				// XXX rewrite the anchor tag to ref.Name in newLines
				// XXX mark ref as resolved
			default:
				fmt.Fprintf(os.Stderr, "Warning: Multiple matches for unresolved reference [%s]\n", ref.Name)
			}
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
