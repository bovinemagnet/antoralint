package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bovinemagnet/antoralint/internal/model"
)

func TestScanFile_XrefDetection(t *testing.T) {
	content := `= Test Page

See xref:other-page.adoc[Other Page].

Also see xref:admin:settings.adoc[Settings].
`
	f := writeTempFile(t, "test.adoc", content)
	refs, err := ScanFile(f, "mycomp", "1.0", "mymodule", model.FamilyPages)
	if err != nil {
		t.Fatalf("ScanFile error: %v", err)
	}
	if len(refs) != 2 {
		t.Errorf("expected 2 xref refs, got %d", len(refs))
	}
	for _, r := range refs {
		if r.RefType != model.RefTypeXref {
			t.Errorf("expected xref type, got %s", r.RefType)
		}
	}
}

func TestScanFile_IncludeDetection(t *testing.T) {
	content := `= Test Page

include::partial$snippet.adoc[]

include::../relative/path.adoc[]
`
	f := writeTempFile(t, "test.adoc", content)
	refs, err := ScanFile(f, "mycomp", "1.0", "mymodule", model.FamilyPages)
	if err != nil {
		t.Fatalf("ScanFile error: %v", err)
	}
	if len(refs) != 2 {
		t.Errorf("expected 2 include refs, got %d", len(refs))
	}
}

func TestScanFile_ImageDetection(t *testing.T) {
	content := `= Test Page

image::diagram.png[Diagram]

image::admin:setup.png[Setup]
`
	f := writeTempFile(t, "test.adoc", content)
	refs, err := ScanFile(f, "mycomp", "1.0", "mymodule", model.FamilyPages)
	if err != nil {
		t.Fatalf("ScanFile error: %v", err)
	}
	if len(refs) != 2 {
		t.Errorf("expected 2 image refs, got %d", len(refs))
	}
}

func TestScanFile_SkipComments(t *testing.T) {
	content := `= Test Page

// xref:commented-out.adoc[This should be ignored]

Real content here.
`
	f := writeTempFile(t, "test.adoc", content)
	refs, err := ScanFile(f, "mycomp", "1.0", "mymodule", model.FamilyPages)
	if err != nil {
		t.Fatalf("ScanFile error: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("expected 0 refs (comment skipped), got %d", len(refs))
	}
}

func TestScanFile_SkipCodeBlocks(t *testing.T) {
	content := `= Test Page

----
xref:inside-code-block.adoc[Not a ref]
----

xref:real-ref.adoc[Real Ref]
`
	f := writeTempFile(t, "test.adoc", content)
	refs, err := ScanFile(f, "mycomp", "1.0", "mymodule", model.FamilyPages)
	if err != nil {
		t.Fatalf("ScanFile error: %v", err)
	}
	// xref inside code block should be skipped
	xrefCount := 0
	for _, r := range refs {
		if r.RefType == model.RefTypeXref {
			xrefCount++
		}
	}
	if xrefCount != 1 {
		t.Errorf("expected 1 xref (code block xref skipped), got %d", xrefCount)
	}
}

func TestScanFile_AttachmentDetection(t *testing.T) {
	content := `= Test Page

Download the link:{attachmentsdir}/report.pdf[report here].

Also see link:{attachmentsdir}/data/export.csv[the CSV export].
`
	f := writeTempFile(t, "test.adoc", content)
	refs, err := ScanFile(f, "mycomp", "1.0", "mymodule", model.FamilyPages)
	if err != nil {
		t.Fatalf("ScanFile error: %v", err)
	}
	attachCount := 0
	for _, r := range refs {
		if r.RefType == model.RefTypeAttachment {
			attachCount++
		}
	}
	if attachCount != 2 {
		t.Errorf("expected 2 attachment refs, got %d", attachCount)
	}
	// Verify targets are extracted correctly (without {attachmentsdir}/ prefix)
	for _, r := range refs {
		if r.RefType == model.RefTypeAttachment {
			if r.Target != "report.pdf" && r.Target != "data/export.csv" {
				t.Errorf("unexpected attachment target: %s", r.Target)
			}
		}
	}
}

func TestScanFile_AttachmentSkippedInCodeBlocks(t *testing.T) {
	content := `= Test Page

----
link:{attachmentsdir}/inside-block.pdf[Not a ref]
----

link:{attachmentsdir}/real-attachment.pdf[Real Ref]
`
	f := writeTempFile(t, "test.adoc", content)
	refs, err := ScanFile(f, "mycomp", "1.0", "mymodule", model.FamilyPages)
	if err != nil {
		t.Fatalf("ScanFile error: %v", err)
	}
	attachCount := 0
	for _, r := range refs {
		if r.RefType == model.RefTypeAttachment {
			attachCount++
		}
	}
	if attachCount != 1 {
		t.Errorf("expected 1 attachment ref (code block skipped), got %d", attachCount)
	}
}

func TestScanFileWithOptions_ExternalLinks(t *testing.T) {
	content := `= Test Page

Visit https://example.com for more info.

See link:https://docs.example.com/guide[the guide].

Also check http://legacy.example.com/old-page for legacy docs.
`
	f := writeTempFile(t, "test.adoc", content)
	opts := ScanOptions{ExtractExternalLinks: true}
	refs, err := ScanFileWithOptions(f, "mycomp", "1.0", "mymodule", model.FamilyPages, opts)
	if err != nil {
		t.Fatalf("ScanFileWithOptions error: %v", err)
	}
	linkCount := 0
	for _, r := range refs {
		if r.RefType == model.RefTypeLink {
			linkCount++
		}
	}
	if linkCount != 3 {
		t.Errorf("expected 3 external link refs, got %d", linkCount)
		for _, r := range refs {
			if r.RefType == model.RefTypeLink {
				t.Logf("  link: %s", r.Target)
			}
		}
	}
}

func TestScanFileWithOptions_ExternalLinksDisabled(t *testing.T) {
	content := `= Test Page

Visit https://example.com for more info.
`
	f := writeTempFile(t, "test.adoc", content)
	// Default options: external links disabled
	refs, err := ScanFile(f, "mycomp", "1.0", "mymodule", model.FamilyPages)
	if err != nil {
		t.Fatalf("ScanFile error: %v", err)
	}
	for _, r := range refs {
		if r.RefType == model.RefTypeLink {
			t.Errorf("external link should not be extracted when disabled, got: %s", r.Target)
		}
	}
}

func TestScanFileWithOptions_ExternalLinksInCodeBlock(t *testing.T) {
	content := `= Test Page

----
https://inside-block.example.com should be ignored
----

https://outside-block.example.com should be found
`
	f := writeTempFile(t, "test.adoc", content)
	opts := ScanOptions{ExtractExternalLinks: true}
	refs, err := ScanFileWithOptions(f, "mycomp", "1.0", "mymodule", model.FamilyPages, opts)
	if err != nil {
		t.Fatalf("ScanFileWithOptions error: %v", err)
	}
	linkCount := 0
	for _, r := range refs {
		if r.RefType == model.RefTypeLink {
			linkCount++
		}
	}
	if linkCount != 1 {
		t.Errorf("expected 1 external link (code block skipped), got %d", linkCount)
	}
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("could not write temp file: %v", err)
	}
	return path
}
