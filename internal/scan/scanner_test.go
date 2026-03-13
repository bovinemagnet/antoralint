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

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("could not write temp file: %v", err)
	}
	return path
}
