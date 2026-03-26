package scan

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"unicode"
)

var (
	// Section headings: = Title, == Section, etc.
	reHeading = regexp.MustCompile(`^(={1,6})\s+(.+)`)
	// Explicit block anchor: [[anchor-id]] or [[anchor-id,label]]
	reBlockAnchor = regexp.MustCompile(`^\[\[([^\],]+)`)
	// Shorthand anchor: [#anchor-id] (may have additional attributes)
	reShorthandAnchor = regexp.MustCompile(`^\[#([^\],\]]+)`)
	// Inline anchor: anchor:id[]
	reInlineAnchor = regexp.MustCompile(`anchor:([^\[]+)\[`)
)

// ExtractAnchors scans an .adoc file and returns all anchor IDs found.
// idPrefix and idSeparator control heading ID generation to match
// Asciidoctor's idprefix/idseparator settings.
func ExtractAnchors(path string, idPrefix string, idSeparator string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	anchors := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	inBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Toggle block state on delimiter lines
		if blockDelimiters[trimmed] {
			inBlock = !inBlock
			continue
		}

		// Skip comment lines
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Skip content inside delimited blocks
		if inBlock {
			continue
		}

		// Check for section headings
		if m := reHeading.FindStringSubmatch(line); m != nil {
			title := strings.TrimSpace(m[2])
			id := headingToID(title, idPrefix, idSeparator)
			anchors[id] = true
			continue
		}

		// Check for explicit block anchors: [[id]]
		if m := reBlockAnchor.FindStringSubmatch(trimmed); m != nil {
			anchors[m[1]] = true
			continue
		}

		// Check for shorthand anchors: [#id]
		if m := reShorthandAnchor.FindStringSubmatch(trimmed); m != nil {
			anchors[m[1]] = true
			continue
		}

		// Check for inline anchors: anchor:id[]
		for _, m := range reInlineAnchor.FindAllStringSubmatch(line, -1) {
			anchors[m[1]] = true
		}
	}

	return anchors, scanner.Err()
}

// headingToID converts a heading title to an Asciidoctor-style ID.
// Matches Asciidoctor behaviour: lowercase, replace runs of non-alphanumeric
// characters with idSeparator, prepend idPrefix.
func headingToID(title string, idPrefix string, idSeparator string) string {
	// Remove inline formatting markup
	title = stripInlineMarkup(title)

	var b strings.Builder
	b.WriteString(idPrefix)
	prevSep := true // true to avoid leading separator

	for _, r := range strings.ToLower(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevSep = false
		} else if !prevSep {
			b.WriteString(idSeparator)
			prevSep = true
		}
	}

	// Trim trailing separator
	result := b.String()
	result = strings.TrimSuffix(result, idSeparator)
	return result
}

// stripInlineMarkup removes common AsciiDoc inline formatting from a title.
func stripInlineMarkup(s string) string {
	// Remove bold, italic, monospace markers
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "``", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "`", "")
	return s
}
