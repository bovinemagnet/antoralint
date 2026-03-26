package scan

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/bovinemagnet/antoralint/internal/model"
)

var (
	// xref:target[label] or xref:target[]
	reXref = regexp.MustCompile(`xref:([^\[]+)\[`)
	// include::target[opts]
	reInclude = regexp.MustCompile(`include::([^\[]+)\[`)
	// image::target[alt] (block)
	reImage = regexp.MustCompile(`image::([^\[]+)\[`)
	// image:target[alt] (inline, not followed by another colon)
	reImageInline = regexp.MustCompile(`image:([^:\[]+)\[`)
	// link:{attachmentsdir}/target[label] — Antora attachment reference
	reAttachment = regexp.MustCompile(`link:\{attachmentsdir\}/([^\[]+)\[`)
	// link:https://...[label] — AsciiDoc link macro with URL
	reLinkMacro = regexp.MustCompile(`link:(https?://[^\[]+)\[`)
	// bare URL: https://... (not preceded by link:); excludes trailing punctuation
	reURL = regexp.MustCompile(`(?:^|[^:])(?:^|[\s(])((https?://[^\s\[\]<>"]+[^\s\[\]<>".,;:!?)]))`)
)

// ScanOptions controls optional scanning behaviour.
type ScanOptions struct {
	ExtractExternalLinks bool
}

// blockDelimiters are line content that toggles "in block" state.
// Inside a delimited block, xref and image refs are skipped but include:: is still processed.
var blockDelimiters = map[string]bool{
	"----": true,
	"....": true,
	"```":  true,
}

// ScanFile scans a single .adoc file and returns all references found.
func ScanFile(path string, srcComponent, srcVersion, srcModule string, srcFamily model.Family) ([]*model.Reference, error) {
	return ScanFileWithOptions(path, srcComponent, srcVersion, srcModule, srcFamily, ScanOptions{})
}

// ScanFileWithOptions scans a single .adoc file with configurable options.
func ScanFileWithOptions(path string, srcComponent, srcVersion, srcModule string, srcFamily model.Family, opts ScanOptions) ([]*model.Reference, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var refs []*model.Reference
	scanner := bufio.NewScanner(f)
	lineNum := 0
	inBlock := false

	for scanner.Scan() {
		lineNum++
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

		// Always extract include:: directives (AsciiDoc processes them before other substitutions)
		for _, m := range reInclude.FindAllStringSubmatchIndex(line, -1) {
			target := line[m[2]:m[3]]
			refs = append(refs, &model.Reference{
				SourceFile:   path,
				Line:         lineNum,
				Column:       m[0] + 1,
				RawText:      line[m[0]:m[1]],
				RefType:      model.RefTypeInclude,
				Target:       target,
				SrcComponent: srcComponent,
				SrcVersion:   srcVersion,
				SrcModule:    srcModule,
				SrcFamily:    srcFamily,
			})
		}

		// Skip xref and image detection inside delimited blocks
		if inBlock {
			continue
		}

		// Extract xref references
		for _, m := range reXref.FindAllStringSubmatchIndex(line, -1) {
			target := line[m[2]:m[3]]
			target, fragment := splitFragment(target)
			refs = append(refs, &model.Reference{
				SourceFile:   path,
				Line:         lineNum,
				Column:       m[0] + 1,
				RawText:      line[m[0]:m[1]],
				RefType:      model.RefTypeXref,
				Target:       target,
				Fragment:     fragment,
				SrcComponent: srcComponent,
				SrcVersion:   srcVersion,
				SrcModule:    srcModule,
				SrcFamily:    srcFamily,
			})
		}

		// Extract image block references (image::)
		for _, m := range reImage.FindAllStringSubmatchIndex(line, -1) {
			target := line[m[2]:m[3]]
			refs = append(refs, &model.Reference{
				SourceFile:   path,
				Line:         lineNum,
				Column:       m[0] + 1,
				RawText:      line[m[0]:m[1]],
				RefType:      model.RefTypeImage,
				Target:       target,
				SrcComponent: srcComponent,
				SrcVersion:   srcVersion,
				SrcModule:    srcModule,
				SrcFamily:    srcFamily,
			})
		}

		// Extract attachment references (link:{attachmentsdir}/target[label])
		for _, m := range reAttachment.FindAllStringSubmatchIndex(line, -1) {
			target := line[m[2]:m[3]]
			refs = append(refs, &model.Reference{
				SourceFile:   path,
				Line:         lineNum,
				Column:       m[0] + 1,
				RawText:      line[m[0]:m[1]],
				RefType:      model.RefTypeAttachment,
				Target:       target,
				SrcComponent: srcComponent,
				SrcVersion:   srcVersion,
				SrcModule:    srcModule,
				SrcFamily:    srcFamily,
			})
		}

		// Extract inline image references (image: not image::)
		for _, m := range reImageInline.FindAllStringSubmatchIndex(line, -1) {
			start := m[0]
			// Avoid matching image:: (block) — reImageInline already excludes ':' as first char
			// but double-check if the preceding char makes this image::
			if start > 0 && line[start-1] == ':' {
				continue
			}
			target := line[m[2]:m[3]]
			refs = append(refs, &model.Reference{
				SourceFile:   path,
				Line:         lineNum,
				Column:       m[0] + 1,
				RawText:      line[m[0]:m[1]],
				RefType:      model.RefTypeImage,
				Target:       target,
				SrcComponent: srcComponent,
				SrcVersion:   srcVersion,
				SrcModule:    srcModule,
				SrcFamily:    srcFamily,
			})
		}

		// Extract external links (only when enabled)
		if opts.ExtractExternalLinks {
			refs = append(refs, extractExternalLinks(line, lineNum, path, srcComponent, srcVersion, srcModule, srcFamily)...)
		}
	}

	return refs, scanner.Err()
}

// extractExternalLinks extracts HTTP/HTTPS URLs from a line.
func extractExternalLinks(line string, lineNum int, path, srcComponent, srcVersion, srcModule string, srcFamily model.Family) []*model.Reference {
	var refs []*model.Reference
	seen := make(map[string]bool)

	// First: link macros (link:https://...[label])
	for _, m := range reLinkMacro.FindAllStringSubmatchIndex(line, -1) {
		url := line[m[2]:m[3]]
		if seen[url] {
			continue
		}
		seen[url] = true
		refs = append(refs, &model.Reference{
			SourceFile:   path,
			Line:         lineNum,
			Column:       m[0] + 1,
			RawText:      line[m[0]:m[1]],
			RefType:      model.RefTypeLink,
			Target:       url,
			SrcComponent: srcComponent,
			SrcVersion:   srcVersion,
			SrcModule:    srcModule,
			SrcFamily:    srcFamily,
		})
	}

	// Second: bare URLs not already captured by link macros
	for _, m := range reURL.FindAllStringSubmatchIndex(line, -1) {
		url := line[m[2]:m[3]]
		if seen[url] {
			continue
		}
		seen[url] = true
		refs = append(refs, &model.Reference{
			SourceFile:   path,
			Line:         lineNum,
			Column:       m[2] + 1,
			RawText:      url,
			RefType:      model.RefTypeLink,
			Target:       url,
			SrcComponent: srcComponent,
			SrcVersion:   srcVersion,
			SrcModule:    srcModule,
			SrcFamily:    srcFamily,
		})
	}

	return refs
}

func splitFragment(target string) (string, string) {
	if idx := strings.LastIndex(target, "#"); idx >= 0 {
		return target[:idx], target[idx+1:]
	}
	return target, ""
}
