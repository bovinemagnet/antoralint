package model

// Severity represents the severity level of a diagnostic.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Family represents an Antora resource family.
type Family string

const (
	FamilyPages       Family = "pages"
	FamilyPartials    Family = "partials"
	FamilyExamples    Family = "examples"
	FamilyImages      Family = "images"
	FamilyAttachments Family = "attachments"
	FamilyUnknown     Family = ""
)

// RefType represents the type of a reference.
type RefType string

const (
	RefTypeXref       RefType = "xref"
	RefTypeInclude    RefType = "include"
	RefTypeImage      RefType = "image"
	RefTypeAttachment RefType = "attachment"
	RefTypeLink       RefType = "link"
)

// Resource represents a discovered repository resource.
type Resource struct {
	AbsPath   string
	RelPath   string
	Component string
	Version   string
	Module    string
	Family    Family
	LogicalID string
}

// Reference represents a parsed reference from an AsciiDoc file.
type Reference struct {
	SourceFile string
	Line       int
	Column     int
	RawText    string
	RefType    RefType
	Target     string
	Fragment   string
	// context from source file
	SrcComponent string
	SrcVersion   string
	SrcModule    string
	SrcFamily    Family
}

// IncludeStep represents one step in an include chain,
// showing which file included the diagnostic's file.
type IncludeStep struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// Diagnostic represents a reported issue.
type Diagnostic struct {
	Severity     Severity
	RuleID       string
	Message      string
	File         string
	Line         int
	Column       int
	Target       string
	Fix          string
	IncludeChain []IncludeStep
}
