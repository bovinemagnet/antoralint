package rules

import (
	"fmt"

	"path/filepath"
	"strings"

	"github.com/bovinemagnet/antoralint/internal/linkcheck"
	"github.com/bovinemagnet/antoralint/internal/model"
	"github.com/bovinemagnet/antoralint/internal/resolve"
)

const (
	RuleBrokenXref       = "broken-xref"
	RuleBrokenInclude    = "broken-include"
	RuleBrokenImage      = "broken-image"
	RuleBrokenAttachment = "broken-attachment"
	RuleCaseMismatch     = "case-mismatch"
	RuleUnresolvedAttr   = "unresolved-attribute"
	RuleIncludeCycle       = "include-cycle"
	RuleExternalLinkDead   = "external-link-dead"
	RuleExternalLinkTimeout = "external-link-timeout"
)

// Evaluate converts a resolve.Result into zero or more Diagnostics.
func Evaluate(result *resolve.Result) []*model.Diagnostic {
	ref := result.Ref

	if result.HasUnresolvedAttr {
		return []*model.Diagnostic{{
			Severity: model.SeverityWarning,
			RuleID:   RuleUnresolvedAttr,
			Message:  fmt.Sprintf("target contains unresolved attribute: %s", ref.Target),
			File:     ref.SourceFile,
			Line:     ref.Line,
			Column:   ref.Column,
			Target:   ref.Target,
		}}
	}

	if result.Found && result.CaseMismatch {
		return []*model.Diagnostic{{
			Severity: model.SeverityWarning,
			RuleID:   RuleCaseMismatch,
			Message:  fmt.Sprintf("case mismatch in reference: %s (actual: %s)", ref.Target, result.Resource.RelPath),
			File:     ref.SourceFile,
			Line:     ref.Line,
			Column:   ref.Column,
			Target:   ref.Target,
		}}
	}

	if !result.Found {
		return []*model.Diagnostic{notFoundDiagnostic(ref)}
	}

	return nil
}

// EvaluateCycles converts detected include cycles into diagnostics.
// Each cycle is a slice of absolute file paths forming the loop.
// The rootDir is used to make paths relative for output.
func EvaluateCycles(cycles [][]string, rootDir string) []*model.Diagnostic {
	var diags []*model.Diagnostic
	for _, cycle := range cycles {
		relPaths := make([]string, len(cycle))
		for i, p := range cycle {
			rel, err := filepath.Rel(rootDir, p)
			if err != nil {
				relPaths[i] = p
			} else {
				relPaths[i] = filepath.ToSlash(rel)
			}
		}
		chain := strings.Join(relPaths, " -> ")
		diags = append(diags, &model.Diagnostic{
			Severity: model.SeverityError,
			RuleID:   RuleIncludeCycle,
			Message:  fmt.Sprintf("include cycle detected: %s", chain),
			File:     relPaths[0],
			Line:     1,
			Target:   chain,
		})
	}
	return diags
}

// EvaluateLinkResult converts a link check result into a diagnostic.
// Returns nil if the link is OK.
func EvaluateLinkResult(ref *model.Reference, result *linkcheck.Result) *model.Diagnostic {
	if result.IsOK() {
		return nil
	}

	if result.TimedOut {
		return &model.Diagnostic{
			Severity: model.SeverityWarning,
			RuleID:   RuleExternalLinkTimeout,
			Message:  fmt.Sprintf("external link timed out: %s", ref.Target),
			File:     ref.SourceFile,
			Line:     ref.Line,
			Column:   ref.Column,
			Target:   ref.Target,
		}
	}

	if result.IsDead() {
		return &model.Diagnostic{
			Severity: model.SeverityError,
			RuleID:   RuleExternalLinkDead,
			Message:  fmt.Sprintf("external link dead (HTTP %d): %s", result.StatusCode, ref.Target),
			File:     ref.SourceFile,
			Line:     ref.Line,
			Column:   ref.Column,
			Target:   ref.Target,
		}
	}

	if result.IsTransient() {
		return &model.Diagnostic{
			Severity: model.SeverityWarning,
			RuleID:   RuleExternalLinkTimeout,
			Message:  fmt.Sprintf("external link unreachable: %s (%v)", ref.Target, result.Error),
			File:     ref.SourceFile,
			Line:     ref.Line,
			Column:   ref.Column,
			Target:   ref.Target,
		}
	}

	return nil
}

func notFoundDiagnostic(ref *model.Reference) *model.Diagnostic {
	switch ref.RefType {
	case model.RefTypeXref:
		return &model.Diagnostic{
			Severity: model.SeverityError,
			RuleID:   RuleBrokenXref,
			Message:  fmt.Sprintf("xref target not found: %s", ref.Target),
			File:     ref.SourceFile,
			Line:     ref.Line,
			Column:   ref.Column,
			Target:   ref.Target,
		}
	case model.RefTypeInclude:
		return &model.Diagnostic{
			Severity: model.SeverityError,
			RuleID:   RuleBrokenInclude,
			Message:  fmt.Sprintf("include target not found: %s", ref.Target),
			File:     ref.SourceFile,
			Line:     ref.Line,
			Column:   ref.Column,
			Target:   ref.Target,
		}
	case model.RefTypeImage:
		return &model.Diagnostic{
			Severity: model.SeverityError,
			RuleID:   RuleBrokenImage,
			Message:  fmt.Sprintf("image target not found: %s", ref.Target),
			File:     ref.SourceFile,
			Line:     ref.Line,
			Column:   ref.Column,
			Target:   ref.Target,
		}
	case model.RefTypeAttachment:
		return &model.Diagnostic{
			Severity: model.SeverityError,
			RuleID:   RuleBrokenAttachment,
			Message:  fmt.Sprintf("attachment target not found: %s", ref.Target),
			File:     ref.SourceFile,
			Line:     ref.Line,
			Column:   ref.Column,
			Target:   ref.Target,
		}
	}
	return &model.Diagnostic{
		Severity: model.SeverityError,
		RuleID:   "unknown",
		Message:  fmt.Sprintf("reference target not found: %s", ref.Target),
		File:     ref.SourceFile,
		Line:     ref.Line,
		Column:   ref.Column,
		Target:   ref.Target,
	}
}
