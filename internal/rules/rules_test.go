package rules

import (
	"testing"

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
