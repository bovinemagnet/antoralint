package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bovinemagnet/antoralint/internal/index"
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
		outputFormat string
		failOn       string
		verbose      bool
		excludes     []string
		includes     []string
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
			resolver := resolve.New(idx)
			var allDiagnostics []*model.Diagnostic

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

				refs, err := scan.ScanFile(res.AbsPath, res.Component, res.Version, res.Module, res.Family)
				if err != nil {
					fmt.Fprintf(os.Stderr, "WARNING: could not scan %s: %v\n", res.RelPath, err)
					continue
				}

				for _, ref := range refs {
					// Make file path relative to repo root for output
					relPath, _ := filepath.Rel(absRoot, ref.SourceFile)
					ref.SourceFile = filepath.ToSlash(relPath)

					result := resolver.Resolve(ref)
					diags := rules.Evaluate(result)
					allDiagnostics = append(allDiagnostics, diags...)
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

	rootCmd.AddCommand(scanCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(2)
	}
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
