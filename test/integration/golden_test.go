package integration

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/bovinemagnet/antoralint/internal/cycles"
	"github.com/bovinemagnet/antoralint/internal/index"
	"github.com/bovinemagnet/antoralint/internal/model"
	"github.com/bovinemagnet/antoralint/internal/report"
	"github.com/bovinemagnet/antoralint/internal/repo"
	"github.com/bovinemagnet/antoralint/internal/resolve"
	"github.com/bovinemagnet/antoralint/internal/rules"
	"github.com/bovinemagnet/antoralint/internal/scan"
	"strings"
)

var update = flag.Bool("update", false, "update golden files")

// runPipeline runs the full adoclint pipeline against a fixture directory and
// returns the formatted output in the requested format.
func runPipeline(t *testing.T, fixtureDir string, format report.Format) []byte {
	t.Helper()

	absRoot, err := filepath.Abs(fixtureDir)
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}

	components, err := repo.Discover(absRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	idx, err := index.Build(absRoot, components)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	resolver := resolve.New(idx)
	var allDiags []*model.Diagnostic

	for _, res := range idx.Resources {
		if res.Family != model.FamilyPages && res.Family != model.FamilyPartials {
			continue
		}
		if !strings.HasSuffix(res.AbsPath, ".adoc") {
			continue
		}

		refs, err := scan.ScanFile(res.AbsPath, res.Component, res.Version, res.Module, res.Family)
		if err != nil {
			t.Logf("WARNING: could not scan %s: %v", res.RelPath, err)
			continue
		}

		for _, ref := range refs {
			relPath, _ := filepath.Rel(absRoot, ref.SourceFile)
			ref.SourceFile = filepath.ToSlash(relPath)

			result := resolver.Resolve(ref)
			diags := rules.Evaluate(result)
			allDiags = append(allDiags, diags...)
		}
	}

	// Sort diagnostics for deterministic output
	sort.Slice(allDiags, func(i, j int) bool {
		if allDiags[i].File != allDiags[j].File {
			return allDiags[i].File < allDiags[j].File
		}
		if allDiags[i].Line != allDiags[j].Line {
			return allDiags[i].Line < allDiags[j].Line
		}
		return allDiags[i].RuleID < allDiags[j].RuleID
	})

	var buf bytes.Buffer
	w := report.New(format, &buf)
	if err := w.Write(allDiags); err != nil {
		t.Fatalf("Write: %v", err)
	}

	return buf.Bytes()
}

func goldenPath(name string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "golden", name)
}

func fixturePath(name string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "fixtures", name)
}

func assertGolden(t *testing.T, goldenName string, actual []byte) {
	t.Helper()
	path := goldenPath(goldenName)

	if *update {
		if err := os.WriteFile(path, actual, 0644); err != nil {
			t.Fatalf("update golden: %v", err)
		}
		t.Logf("updated golden file: %s", path)
		return
	}

	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file %s: %v (run with -update to generate)", path, err)
	}

	if !bytes.Equal(expected, actual) {
		t.Errorf("output does not match golden file %s\n--- expected ---\n%s\n--- actual ---\n%s",
			goldenName, string(expected), string(actual))
	}
}

func TestGolden_BrokenText(t *testing.T) {
	output := runPipeline(t, fixturePath("broken"), report.FormatText)
	assertGolden(t, "broken-text.golden", output)
}

func TestGolden_BrokenJSON(t *testing.T) {
	output := runPipeline(t, fixturePath("broken"), report.FormatJSON)
	assertGolden(t, "broken-json.golden", output)
}

func TestGolden_BrokenSARIF(t *testing.T) {
	output := runPipeline(t, fixturePath("broken"), report.FormatSARIF)
	assertGolden(t, "broken-sarif.golden", output)
}

func TestGolden_CaseMismatchText(t *testing.T) {
	output := runPipeline(t, fixturePath("casemismatch"), report.FormatText)
	assertGolden(t, "casemismatch-text.golden", output)
}

func TestGolden_MultiComponentText(t *testing.T) {
	output := runPipeline(t, fixturePath("multicomponent"), report.FormatText)
	assertGolden(t, "multicomponent-text.golden", output)
}

func TestGolden_CyclesText(t *testing.T) {
	output := runPipelineWithCycles(t, fixturePath("cycles"), report.FormatText)
	assertGolden(t, "cycles-text.golden", output)
}

// runPipelineWithCycles runs the full pipeline including cycle detection.
func runPipelineWithCycles(t *testing.T, fixtureDir string, format report.Format) []byte {
	t.Helper()

	absRoot, err := filepath.Abs(fixtureDir)
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}

	components, err := repo.Discover(absRoot)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	idx, err := index.Build(absRoot, components)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	resolver := resolve.New(idx)
	var allDiags []*model.Diagnostic
	var includeResults []*resolve.Result

	for _, res := range idx.Resources {
		if res.Family != model.FamilyPages && res.Family != model.FamilyPartials {
			continue
		}
		if !strings.HasSuffix(res.AbsPath, ".adoc") {
			continue
		}

		refs, err := scan.ScanFile(res.AbsPath, res.Component, res.Version, res.Module, res.Family)
		if err != nil {
			t.Logf("WARNING: could not scan %s: %v", res.RelPath, err)
			continue
		}

		for _, ref := range refs {
			absSourceFile := ref.SourceFile
			relPath, _ := filepath.Rel(absRoot, ref.SourceFile)
			ref.SourceFile = filepath.ToSlash(relPath)

			result := resolver.Resolve(ref)
			diags := rules.Evaluate(result)
			allDiags = append(allDiags, diags...)

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
			}
		}
	}

	// Detect cycles
	graph := cycles.Build(includeResults)
	detected := graph.DetectCycles()
	if len(detected) > 0 {
		cycleDiags := rules.EvaluateCycles(detected, absRoot)
		allDiags = append(allDiags, cycleDiags...)
	}

	sort.Slice(allDiags, func(i, j int) bool {
		if allDiags[i].File != allDiags[j].File {
			return allDiags[i].File < allDiags[j].File
		}
		if allDiags[i].Line != allDiags[j].Line {
			return allDiags[i].Line < allDiags[j].Line
		}
		return allDiags[i].RuleID < allDiags[j].RuleID
	})

	var buf bytes.Buffer
	w := report.New(format, &buf)
	if err := w.Write(allDiags); err != nil {
		t.Fatalf("Write: %v", err)
	}

	return buf.Bytes()
}
