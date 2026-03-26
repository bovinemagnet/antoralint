package rules

import (
	"fmt"
	"testing"

	"github.com/bovinemagnet/antoralint/internal/linkcheck"
	"github.com/bovinemagnet/antoralint/internal/model"
	"github.com/bovinemagnet/antoralint/internal/resolve"
)

func TestEvaluate_BrokenXref(t *testing.T) {
	result := &resolve.Result{
		Ref: &model.Reference{
			RefType:    model.RefTypeXref,
			Target:     "missing-page.adoc",
			SourceFile: "docs/page.adoc",
			Line:       10,
		},
		Found: false,
	}
	diags := Evaluate(result)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].RuleID != RuleBrokenXref {
		t.Errorf("expected ruleID %q, got %q", RuleBrokenXref, diags[0].RuleID)
	}
	if diags[0].Severity != model.SeverityError {
		t.Errorf("expected error severity, got %s", diags[0].Severity)
	}
}

func TestEvaluate_UnresolvedAttribute(t *testing.T) {
	result := &resolve.Result{
		Ref: &model.Reference{
			RefType:    model.RefTypeXref,
			Target:     "{product-page}",
			SourceFile: "docs/page.adoc",
			Line:       5,
		},
		HasUnresolvedAttr: true,
	}
	diags := Evaluate(result)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].RuleID != RuleUnresolvedAttr {
		t.Errorf("expected ruleID %q, got %q", RuleUnresolvedAttr, diags[0].RuleID)
	}
	if diags[0].Severity != model.SeverityWarning {
		t.Errorf("expected warning severity, got %s", diags[0].Severity)
	}
}

func TestEvaluate_FoundNoIssue(t *testing.T) {
	result := &resolve.Result{
		Ref: &model.Reference{
			RefType:    model.RefTypeXref,
			Target:     "valid-page.adoc",
			SourceFile: "docs/page.adoc",
			Line:       3,
		},
		Found: true,
	}
	diags := Evaluate(result)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for valid ref, got %d", len(diags))
	}
}

func TestEvaluate_CaseMismatch(t *testing.T) {
	result := &resolve.Result{
		Ref: &model.Reference{
			RefType:    model.RefTypeXref,
			Target:     "Page.adoc",
			SourceFile: "docs/page.adoc",
			Line:       7,
		},
		Found:        true,
		CaseMismatch: true,
		Resource: &model.Resource{
			RelPath: "modules/ROOT/pages/page.adoc",
		},
	}
	diags := Evaluate(result)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic for case mismatch, got %d", len(diags))
	}
	if diags[0].RuleID != RuleCaseMismatch {
		t.Errorf("expected ruleID %q, got %q", RuleCaseMismatch, diags[0].RuleID)
	}
}

func TestEvaluateCycles(t *testing.T) {
	cycles := [][]string{
		{"/tmp/a.adoc", "/tmp/b.adoc", "/tmp/a.adoc"},
	}
	diags := EvaluateCycles(cycles, "/tmp")
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].RuleID != RuleIncludeCycle {
		t.Errorf("expected ruleID %q, got %q", RuleIncludeCycle, diags[0].RuleID)
	}
	if diags[0].Severity != model.SeverityError {
		t.Errorf("expected error severity, got %s", diags[0].Severity)
	}
}

func TestEvaluateCycles_Empty(t *testing.T) {
	diags := EvaluateCycles(nil, "/tmp")
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for nil cycles, got %d", len(diags))
	}
}

func TestEvaluateLinkResult_Dead(t *testing.T) {
	ref := &model.Reference{
		RefType:    model.RefTypeLink,
		Target:     "https://example.com/missing",
		SourceFile: "docs/page.adoc",
		Line:       5,
	}
	result := &linkcheck.Result{
		URL:        "https://example.com/missing",
		StatusCode: 404,
	}
	d := EvaluateLinkResult(ref, result)
	if d == nil {
		t.Fatal("expected diagnostic for dead link")
	}
	if d.RuleID != RuleExternalLinkDead {
		t.Errorf("expected ruleID %q, got %q", RuleExternalLinkDead, d.RuleID)
	}
	if d.Severity != model.SeverityError {
		t.Errorf("expected error severity, got %s", d.Severity)
	}
}

func TestEvaluateLinkResult_Timeout(t *testing.T) {
	ref := &model.Reference{
		RefType:    model.RefTypeLink,
		Target:     "https://slow.example.com",
		SourceFile: "docs/page.adoc",
		Line:       10,
	}
	result := &linkcheck.Result{
		URL:      "https://slow.example.com",
		TimedOut: true,
		Error:    fmt.Errorf("timeout"),
	}
	d := EvaluateLinkResult(ref, result)
	if d == nil {
		t.Fatal("expected diagnostic for timed out link")
	}
	if d.RuleID != RuleExternalLinkTimeout {
		t.Errorf("expected ruleID %q, got %q", RuleExternalLinkTimeout, d.RuleID)
	}
	if d.Severity != model.SeverityWarning {
		t.Errorf("expected warning severity, got %s", d.Severity)
	}
}

func TestEvaluateLinkResult_OK(t *testing.T) {
	ref := &model.Reference{
		RefType:    model.RefTypeLink,
		Target:     "https://example.com",
		SourceFile: "docs/page.adoc",
		Line:       3,
	}
	result := &linkcheck.Result{
		URL:        "https://example.com",
		StatusCode: 200,
	}
	d := EvaluateLinkResult(ref, result)
	if d != nil {
		t.Errorf("expected no diagnostic for OK link, got %v", d)
	}
}
