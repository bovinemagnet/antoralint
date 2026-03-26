package cycles

import (
	"testing"

	"github.com/bovinemagnet/antoralint/internal/model"
	"github.com/bovinemagnet/antoralint/internal/resolve"
)

func makeIncludeResult(src, dst string) *resolve.Result {
	return &resolve.Result{
		Ref: &model.Reference{
			RefType:    model.RefTypeInclude,
			SourceFile: src,
			Target:     dst,
		},
		Resource: &model.Resource{AbsPath: dst},
		Found:    true,
	}
}

func TestDetectCycles_NoCycle(t *testing.T) {
	results := []*resolve.Result{
		makeIncludeResult("/a.adoc", "/b.adoc"),
		makeIncludeResult("/b.adoc", "/c.adoc"),
	}
	g := Build(results)
	cycles := g.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("expected no cycles, got %d: %v", len(cycles), cycles)
	}
}

func TestDetectCycles_SimpleCycle(t *testing.T) {
	results := []*resolve.Result{
		makeIncludeResult("/a.adoc", "/b.adoc"),
		makeIncludeResult("/b.adoc", "/a.adoc"),
	}
	g := Build(results)
	cycles := g.DetectCycles()
	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d: %v", len(cycles), cycles)
	}
	// Cycle should contain both nodes plus closing node
	if len(cycles[0]) < 3 {
		t.Errorf("expected cycle of length >= 3, got %v", cycles[0])
	}
}

func TestDetectCycles_LongerCycle(t *testing.T) {
	// a -> b -> c -> b (cycle is b -> c -> b)
	results := []*resolve.Result{
		makeIncludeResult("/a.adoc", "/b.adoc"),
		makeIncludeResult("/b.adoc", "/c.adoc"),
		makeIncludeResult("/c.adoc", "/b.adoc"),
	}
	g := Build(results)
	cycles := g.DetectCycles()
	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d: %v", len(cycles), cycles)
	}
	cycle := cycles[0]
	// First and last element should be the same (cycle closure)
	if cycle[0] != cycle[len(cycle)-1] {
		t.Errorf("cycle should be closed: %v", cycle)
	}
}

func TestDetectCycles_SkipsUnresolved(t *testing.T) {
	results := []*resolve.Result{
		makeIncludeResult("/a.adoc", "/b.adoc"),
		{
			Ref: &model.Reference{
				RefType:    model.RefTypeInclude,
				SourceFile: "/b.adoc",
				Target:     "/a.adoc",
			},
			Found: false, // unresolved — should be ignored
		},
	}
	g := Build(results)
	cycles := g.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("expected no cycles from unresolved includes, got %d", len(cycles))
	}
}

func TestDetectCycles_SkipsNonInclude(t *testing.T) {
	results := []*resolve.Result{
		{
			Ref: &model.Reference{
				RefType:    model.RefTypeXref,
				SourceFile: "/a.adoc",
				Target:     "/b.adoc",
			},
			Resource: &model.Resource{AbsPath: "/b.adoc"},
			Found:    true,
		},
	}
	g := Build(results)
	cycles := g.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("expected no cycles from xref results, got %d", len(cycles))
	}
}
