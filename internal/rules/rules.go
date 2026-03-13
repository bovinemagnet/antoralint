package rules

import (
	"fmt"

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
