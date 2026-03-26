package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bovinemagnet/antoralint/internal/cycles"
	"github.com/bovinemagnet/antoralint/internal/index"
	"github.com/bovinemagnet/antoralint/internal/linkcheck"
	"github.com/bovinemagnet/antoralint/internal/model"
	"github.com/bovinemagnet/antoralint/internal/report"
	"github.com/bovinemagnet/antoralint/internal/repo"
	"github.com/bovinemagnet/antoralint/internal/resolve"
	"github.com/bovinemagnet/antoralint/internal/rules"
	"github.com/bovinemagnet/antoralint/internal/scan"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

func main() {
	rootCmd := &cobra.Command{
		Use:     "adoclint",
		Short:   "Antora/AsciiDoc repository linter",
		Long:    "adoclint scans Antora-based AsciiDoc repositories for broken references and structural issues.",
		Version: version,
	}

	var (
		outputFormat  string
		failOn        string
		verbose       bool
		excludes      []string
		includes      []string
		externalLinks bool
		timeout       time.Duration
		concurrency   int
		idPrefix      string
		idSeparator   string
	)

	scanCmd := &cobra.Command{
		Use:   "scan [directory]",
		Short: "Scan an Antora repository for broken references",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir := "."
			if len(args) > 0 {
				rootDir = args[0]
			}
			absRoot, err := filepath.Abs(rootDir)
			if err != nil {
				return fmt.Errorf("invalid path: %w", err)
			}

			if verbose {
				fmt.Fprintf(os.Stderr, "Scanning: %s\n", absRoot)
			}

			// Discover Antora components
			components, err := repo.Discover(absRoot)
			if err != nil {
				return fmt.Errorf("discovery failed: %w", err)
			}
			if verbose {
				fmt.Fprintf(os.Stderr, "Found %d component(s)\n", len(components))
			}

			// Build resource index
			idx, err := index.Build(absRoot, components)
			if err != nil {
				return fmt.Errorf("indexing failed: %w", err)
			}
			if verbose {
				fmt.Fprintf(os.Stderr, "Indexed %d resource(s)\n", len(idx.Resources))
			}

			// Scan .adoc files and collect diagnostics
			scanOpts := scan.ScanOptions{
				ExtractExternalLinks: externalLinks,
			}
			anchorCache := scan.NewAnchorCache(idPrefix, idSeparator)
			resolver := resolve.New(idx, anchorCache)
			var allDiagnostics []*model.Diagnostic
			var includeResults []*resolve.Result
			var linkRefs []*model.Reference
			// Track which files include which, for include chain reporting
			includedFrom := make(map[string]model.IncludeStep) // key=included file abs path

			for _, res := range idx.Resources {
				if res.Family != model.FamilyPages && res.Family != model.FamilyPartials {
					continue
				}
				if !strings.HasSuffix(res.AbsPath, ".adoc") {
					continue
				}
				if shouldExclude(res.RelPath, excludes) {
					continue
				}
				if len(includes) > 0 && !shouldInclude(res.RelPath, includes) {
					continue
				}

				refs, err := scan.ScanFileWithOptions(res.AbsPath, res.Component, res.Version, res.Module, res.Family, scanOpts)
				if err != nil {
					fmt.Fprintf(os.Stderr, "WARNING: could not scan %s: %v\n", res.RelPath, err)
					continue
				}

				for _, ref := range refs {
					absSourceFile := ref.SourceFile
					// Make file path relative to repo root for output
					relPath, _ := filepath.Rel(absRoot, ref.SourceFile)
					ref.SourceFile = filepath.ToSlash(relPath)

					// External links are checked separately
					if ref.RefType == model.RefTypeLink {
						linkRefs = append(linkRefs, ref)
						continue
					}

					result := resolver.Resolve(ref)
					diags := rules.Evaluate(result)
					allDiagnostics = append(allDiagnostics, diags...)

					// Collect resolved include results for cycle detection (using absolute paths)
					if ref.RefType == model.RefTypeInclude && result.Found {
						cycleResult := &resolve.Result{
							Ref: &model.Reference{
								SourceFile: absSourceFile,
								RefType:    ref.RefType,
								Target:     ref.Target,
							},
							Resource: result.Resource,
							Found:    true,
						}
						includeResults = append(includeResults, cycleResult)

						// Track include provenance for chain reporting
						if result.Resource != nil {
							relSrc, _ := filepath.Rel(absRoot, absSourceFile)
							includedFrom[result.Resource.AbsPath] = model.IncludeStep{
								File: filepath.ToSlash(relSrc),
								Line: ref.Line,
							}
						}
					}
				}
			}

			// Detect include cycles
			graph := cycles.Build(includeResults)
			detectedCycles := graph.DetectCycles()
			if len(detectedCycles) > 0 {
				cycleDiags := rules.EvaluateCycles(detectedCycles, absRoot)
				allDiagnostics = append(allDiagnostics, cycleDiags...)
			}

			// Check external links
			if externalLinks && len(linkRefs) > 0 {
				if verbose {
					fmt.Fprintf(os.Stderr, "Checking %d external link(s)...\n", len(linkRefs))
				}
				linkDiags := checkExternalLinks(linkRefs, concurrency, timeout)
				allDiagnostics = append(allDiagnostics, linkDiags...)
			}

			// Annotate diagnostics with include chain information
			if len(includedFrom) > 0 {
				for _, d := range allDiagnostics {
					chain := buildIncludeChain(d.File, includedFrom, absRoot)
					if len(chain) > 0 {
						d.IncludeChain = chain
					}
				}
			}

			// Write output
			format := report.Format(outputFormat)
			w := report.New(format, os.Stdout)
			if err := w.Write(allDiagnostics); err != nil {
				return fmt.Errorf("output failed: %w", err)
			}

			if format == report.FormatText {
				w.Summary(allDiagnostics, os.Stderr)
			}

			return exitWithCode(allDiagnostics, failOn)
		},
	}

	scanCmd.Flags().StringVar(&outputFormat, "format", "text", "Output format: text, json, sarif")
	scanCmd.Flags().StringVar(&failOn, "fail-on", "error", "Fail on: error, warning, none")
	scanCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	scanCmd.Flags().StringArrayVar(&excludes, "exclude", nil, "Exclude path patterns")
	scanCmd.Flags().StringArrayVar(&includes, "include", nil, "Include path patterns")
	scanCmd.Flags().BoolVar(&externalLinks, "external-links", false, "Enable external link checking")
	scanCmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "Timeout per external link check")
	scanCmd.Flags().IntVar(&concurrency, "concurrency", 5, "Maximum concurrent external link checks")
	scanCmd.Flags().StringVar(&idPrefix, "id-prefix", "_", "Asciidoctor idprefix for heading ID generation")
	scanCmd.Flags().StringVar(&idSeparator, "id-separator", "_", "Asciidoctor idseparator for heading ID generation")

	rootCmd.AddCommand(scanCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(2)
	}
}

// checkExternalLinks validates external URLs and returns diagnostics.
// URLs are deduplicated — each unique URL is checked once, but diagnostics
// are emitted for every reference that uses it.
func checkExternalLinks(refs []*model.Reference, concurrency int, timeout time.Duration) []*model.Diagnostic {
	// Deduplicate URLs
	urlToRefs := make(map[string][]*model.Reference)
	var uniqueURLs []string
	for _, ref := range refs {
		if _, exists := urlToRefs[ref.Target]; !exists {
			uniqueURLs = append(uniqueURLs, ref.Target)
		}
		urlToRefs[ref.Target] = append(urlToRefs[ref.Target], ref)
	}

	checker := linkcheck.New(concurrency, timeout)
	results := checker.Check(uniqueURLs)

	var diags []*model.Diagnostic
	for i, result := range results {
		url := uniqueURLs[i]
		for _, ref := range urlToRefs[url] {
			if d := rules.EvaluateLinkResult(ref, result); d != nil {
				diags = append(diags, d)
			}
		}
	}
	return diags
}

// buildIncludeChain traces the include provenance for a diagnostic file.
// It returns the chain from outermost includer to innermost, or nil if
// the file is not included from anywhere.
func buildIncludeChain(relFile string, includedFrom map[string]model.IncludeStep, absRoot string) []model.IncludeStep {
	absFile := filepath.Join(absRoot, filepath.FromSlash(relFile))
	var chain []model.IncludeStep
	visited := make(map[string]bool)

	current := absFile
	for {
		if visited[current] {
			break // cycle protection
		}
		visited[current] = true

		step, ok := includedFrom[current]
		if !ok {
			break
		}
		chain = append(chain, step)
		current = filepath.Join(absRoot, filepath.FromSlash(step.File))
	}

	return chain
}

func exitWithCode(diagnostics []*model.Diagnostic, failOn string) error {
	if failOn == "none" {
		return nil
	}
	for _, d := range diagnostics {
		if failOn == "warning" && (d.Severity == model.SeverityWarning || d.Severity == model.SeverityError) {
			os.Exit(1)
		}
		if failOn == "error" && d.Severity == model.SeverityError {
			os.Exit(1)
		}
	}
	return nil
}

func shouldExclude(path string, patterns []string) bool {
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, path); matched {
			return true
		}
		if strings.Contains(path, p) {
			return true
		}
	}
	return false
}

func shouldInclude(path string, patterns []string) bool {
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, path); matched {
			return true
		}
		if strings.Contains(path, p) {
			return true
		}
	}
	return false
}
