package scan

import (
	"testing"
)

func TestExtractAnchors_Headings_DefaultPrefixSeparator(t *testing.T) {
	content := `= Document Title

== My First Section

Some content here.

=== Sub Section Here

More content.
`
	f := writeTempFile(t, "test.adoc", content)
	anchors, err := ExtractAnchors(f, "_", "_")
	if err != nil {
		t.Fatalf("ExtractAnchors error: %v", err)
	}

	expected := []string{"_document_title", "_my_first_section", "_sub_section_here"}
	for _, id := range expected {
		if !anchors[id] {
			t.Errorf("expected anchor %q not found; got %v", id, anchors)
		}
	}
}

func TestExtractAnchors_Headings_CustomPrefixSeparator(t *testing.T) {
	content := `= Document Title

== My First Section
`
	f := writeTempFile(t, "test.adoc", content)
	anchors, err := ExtractAnchors(f, "", "-")
	if err != nil {
		t.Fatalf("ExtractAnchors error: %v", err)
	}

	expected := []string{"document-title", "my-first-section"}
	for _, id := range expected {
		if !anchors[id] {
			t.Errorf("expected anchor %q not found; got %v", id, anchors)
		}
	}
}

func TestExtractAnchors_ExplicitAnchors(t *testing.T) {
	content := `= Page

[[custom-anchor]]
== Section One

[#another-anchor]
Some paragraph.

This has an anchor:inline-id[] in it.
`
	f := writeTempFile(t, "test.adoc", content)
	anchors, err := ExtractAnchors(f, "_", "_")
	if err != nil {
		t.Fatalf("ExtractAnchors error: %v", err)
	}

	expected := []string{"custom-anchor", "another-anchor", "inline-id", "_page", "_section_one"}
	for _, id := range expected {
		if !anchors[id] {
			t.Errorf("expected anchor %q not found; got %v", id, anchors)
		}
	}
}

func TestExtractAnchors_SkipCodeBlocks(t *testing.T) {
	content := `= Page

== Real Section

----
== Not A Section

[[not-an-anchor]]
anchor:not-inline[]
----

== Another Real Section
`
	f := writeTempFile(t, "test.adoc", content)
	anchors, err := ExtractAnchors(f, "_", "_")
	if err != nil {
		t.Fatalf("ExtractAnchors error: %v", err)
	}

	if anchors["_not_a_section"] {
		t.Error("heading inside code block should be skipped")
	}
	if anchors["not-an-anchor"] {
		t.Error("anchor inside code block should be skipped")
	}
	if anchors["not-inline"] {
		t.Error("inline anchor inside code block should be skipped")
	}
	if !anchors["_real_section"] {
		t.Error("expected _real_section anchor")
	}
	if !anchors["_another_real_section"] {
		t.Error("expected _another_real_section anchor")
	}
}

func TestExtractAnchors_SkipComments(t *testing.T) {
	content := `= Page

// == Commented Section
// [[commented-anchor]]

== Real Section
`
	f := writeTempFile(t, "test.adoc", content)
	anchors, err := ExtractAnchors(f, "_", "_")
	if err != nil {
		t.Fatalf("ExtractAnchors error: %v", err)
	}

	if anchors["_commented_section"] {
		t.Error("commented heading should be skipped")
	}
	if anchors["commented-anchor"] {
		t.Error("commented anchor should be skipped")
	}
	if !anchors["_real_section"] {
		t.Error("expected _real_section anchor")
	}
}

func TestExtractAnchors_AnchorWithLabel(t *testing.T) {
	content := `= Page

[[my-anchor,My Label]]
== Section
`
	f := writeTempFile(t, "test.adoc", content)
	anchors, err := ExtractAnchors(f, "_", "_")
	if err != nil {
		t.Fatalf("ExtractAnchors error: %v", err)
	}

	if !anchors["my-anchor"] {
		t.Error("expected my-anchor (label should be stripped)")
	}
}

func TestHeadingToID(t *testing.T) {
	tests := []struct {
		title       string
		idPrefix    string
		idSeparator string
		expected    string
	}{
		{"My Section", "_", "_", "_my_section"},
		{"My Section", "", "-", "my-section"},
		{"Hello World!", "_", "_", "_hello_world"},
		{"API v2.0 Guide", "_", "_", "_api_v2_0_guide"},
		{"  Trimmed  ", "_", "_", "_trimmed"},
	}

	for _, tt := range tests {
		got := headingToID(tt.title, tt.idPrefix, tt.idSeparator)
		if got != tt.expected {
			t.Errorf("headingToID(%q, %q, %q) = %q, want %q",
				tt.title, tt.idPrefix, tt.idSeparator, got, tt.expected)
		}
	}
}
