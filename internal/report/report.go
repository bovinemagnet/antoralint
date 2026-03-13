package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/bovinemagnet/antoralint/internal/model"
)

// Format represents the output format.
type Format string

const (
	FormatText  Format = "text"
	FormatJSON  Format = "json"
	FormatSARIF Format = "sarif"
)

// Writer writes diagnostics in a given format.
type Writer struct {
	format Format
	out    io.Writer
}

// New creates a new report Writer.
func New(format Format, out io.Writer) *Writer {
	return &Writer{format: format, out: out}
}

// Write writes diagnostics to the output.
func (w *Writer) Write(diagnostics []*model.Diagnostic) error {
	switch w.format {
	case FormatJSON:
		return w.writeJSON(diagnostics)
	case FormatSARIF:
		return w.writeSARIF(diagnostics)
	default:
		return w.writeText(diagnostics)
	}
}

func (w *Writer) writeText(diagnostics []*model.Diagnostic) error {
	for _, d := range diagnostics {
		severity := strings.ToUpper(string(d.Severity))
		if d.Severity == model.SeverityWarning {
			severity = "WARN "
		} else {
			severity = fmt.Sprintf("%-5s", severity)
		}
		fmt.Fprintf(w.out, "%s %s %s:%d %s\n",
			severity, d.RuleID, d.File, d.Line, d.Message)
	}
	return nil
}

// jsonDiagnostic is the JSON-serializable form of a diagnostic.
type jsonDiagnostic struct {
	Severity string `json:"severity"`
	RuleID   string `json:"ruleId"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column,omitempty"`
	Message  string `json:"message"`
	Target   string `json:"target,omitempty"`
}

func (w *Writer) writeJSON(diagnostics []*model.Diagnostic) error {
	out := make([]jsonDiagnostic, 0, len(diagnostics))
	for _, d := range diagnostics {
		out = append(out, jsonDiagnostic{
			Severity: string(d.Severity),
			RuleID:   d.RuleID,
			File:     d.File,
			Line:     d.Line,
			Column:   d.Column,
			Message:  d.Message,
			Target:   d.Target,
		})
	}
	enc := json.NewEncoder(w.out)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// writeSARIF emits SARIF 2.1.0 output.
func (w *Writer) writeSARIF(diagnostics []*model.Diagnostic) error {
	type sarifMessage struct {
		Text string `json:"text"`
	}
	type sarifArtifactLocation struct {
		URI string `json:"uri"`
	}
	type sarifRegion struct {
		StartLine   int `json:"startLine"`
		StartColumn int `json:"startColumn,omitempty"`
	}
	type sarifPhysicalLocation struct {
		ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
		Region           sarifRegion           `json:"region"`
	}
	type sarifLocation struct {
		PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
	}
	type sarifResult struct {
		RuleID    string          `json:"ruleId"`
		Level     string          `json:"level"`
		Message   sarifMessage    `json:"message"`
		Locations []sarifLocation `json:"locations"`
	}
	type sarifRun struct {
		Tool    map[string]interface{} `json:"tool"`
		Results []sarifResult          `json:"results"`
	}
	type sarifRoot struct {
		Version string     `json:"version"`
		Schema  string     `json:"$schema"`
		Runs    []sarifRun `json:"runs"`
	}

	results := make([]sarifResult, 0, len(diagnostics))
	for _, d := range diagnostics {
		level := "error"
		if d.Severity == model.SeverityWarning {
			level = "warning"
		} else if d.Severity == model.SeverityInfo {
			level = "note"
		}
		col := d.Column
		if col == 0 {
			col = 1
		}
		results = append(results, sarifResult{
			RuleID:  d.RuleID,
			Level:   level,
			Message: sarifMessage{Text: d.Message},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: d.File},
					Region:           sarifRegion{StartLine: d.Line, StartColumn: col},
				},
			}},
		})
	}

	root := sarifRoot{
		Version: "2.1.0",
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Runs: []sarifRun{{
			Tool: map[string]interface{}{
				"driver": map[string]interface{}{
					"name":           "adoclint",
					"version":        "0.1.0",
					"informationUri": "https://github.com/bovinemagnet/antoralint",
				},
			},
			Results: results,
		}},
	}

	enc := json.NewEncoder(w.out)
	enc.SetIndent("", "  ")
	return enc.Encode(root)
}

// Summary prints a summary of diagnostics to the provided writer.
func (w *Writer) Summary(diagnostics []*model.Diagnostic, out io.Writer) {
	errors, warnings := 0, 0
	for _, d := range diagnostics {
		if d.Severity == model.SeverityError {
			errors++
		} else if d.Severity == model.SeverityWarning {
			warnings++
		}
	}
	fmt.Fprintf(out, "\nSummary: %d error(s), %d warning(s)\n", errors, warnings)
}
