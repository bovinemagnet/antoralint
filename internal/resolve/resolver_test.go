package resolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bovinemagnet/antoralint/internal/index"
	"github.com/bovinemagnet/antoralint/internal/model"
	"github.com/bovinemagnet/antoralint/internal/repo"
)

func setupTestIndex(t *testing.T) (*index.Index, string) {
	t.Helper()
	root := t.TempDir()

	comp := &repo.Component{Name: "mycomp", Version: "1.0", RootDir: root}
	createFile(t, root, "modules/ROOT/pages/index.adoc")
	createFile(t, root, "modules/ROOT/pages/guide.adoc")
	createFile(t, root, "modules/ROOT/partials/snippet.adoc")
	createFile(t, root, "modules/ROOT/images/diagram.png")
	createFile(t, root, "modules/admin/pages/settings.adoc")
	createFile(t, root, "modules/ROOT/attachments/report.pdf")

	idx, err := index.Build(root, []*repo.Component{comp})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return idx, root
}

func createFile(t *testing.T, root string, rel string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestResolveXref_SamePage(t *testing.T) {
	idx, _ := setupTestIndex(t)
	r := New(idx)

	ref := &model.Reference{
		RefType:      model.RefTypeXref,
		Target:       "index.adoc",
		SrcComponent: "mycomp",
		SrcVersion:   "1.0",
		SrcModule:    "ROOT",
		SrcFamily:    model.FamilyPages,
	}
	result := r.Resolve(ref)
	if !result.Found {
		t.Error("expected to find index.adoc in same module")
	}
}

func TestResolveXref_CrossModule(t *testing.T) {
	idx, _ := setupTestIndex(t)
	r := New(idx)

	// module:page form — one colon = module qualifier
	ref := &model.Reference{
		RefType:      model.RefTypeXref,
		Target:       "admin:settings.adoc",
		SrcComponent: "mycomp",
		SrcVersion:   "1.0",
		SrcModule:    "ROOT",
		SrcFamily:    model.FamilyPages,
	}
	result := r.Resolve(ref)
	if !result.Found {
		t.Error("expected to resolve admin:settings.adoc as module:page")
	}
}

func TestResolveXref_Missing(t *testing.T) {
	idx, _ := setupTestIndex(t)
	r := New(idx)

	ref := &model.Reference{
		RefType:      model.RefTypeXref,
		Target:       "nonexistent.adoc",
		SrcComponent: "mycomp",
		SrcVersion:   "1.0",
		SrcModule:    "ROOT",
		SrcFamily:    model.FamilyPages,
	}
	result := r.Resolve(ref)
	if result.Found {
		t.Error("expected not to find nonexistent.adoc")
	}
}

func TestResolveXref_UnresolvedAttribute(t *testing.T) {
	idx, _ := setupTestIndex(t)
	r := New(idx)

	ref := &model.Reference{
		RefType:      model.RefTypeXref,
		Target:       "{product-page}",
		SrcComponent: "mycomp",
		SrcVersion:   "1.0",
		SrcModule:    "ROOT",
	}
	result := r.Resolve(ref)
	if !result.HasUnresolvedAttr {
		t.Error("expected HasUnresolvedAttr for target with braces")
	}
}

func TestResolveImage_Found(t *testing.T) {
	idx, _ := setupTestIndex(t)
	r := New(idx)

	ref := &model.Reference{
		RefType:      model.RefTypeImage,
		Target:       "diagram.png",
		SrcComponent: "mycomp",
		SrcVersion:   "1.0",
		SrcModule:    "ROOT",
		SrcFamily:    model.FamilyPages,
	}
	result := r.Resolve(ref)
	if !result.Found {
		t.Error("expected to find diagram.png in images")
	}
}

func TestResolveImage_Missing(t *testing.T) {
	idx, _ := setupTestIndex(t)
	r := New(idx)

	ref := &model.Reference{
		RefType:      model.RefTypeImage,
		Target:       "missing.png",
		SrcComponent: "mycomp",
		SrcVersion:   "1.0",
		SrcModule:    "ROOT",
		SrcFamily:    model.FamilyPages,
	}
	result := r.Resolve(ref)
	if result.Found {
		t.Error("expected not to find missing.png")
	}
}

func TestResolveXref_CrossComponentDefaultModule(t *testing.T) {
	// component::page form (empty module defaults to ROOT)
	root := t.TempDir()
	compA := &repo.Component{Name: "comp-a", Version: "1.0", RootDir: root + "/comp-a"}
	compB := &repo.Component{Name: "comp-b", Version: "1.0", RootDir: root + "/comp-b"}
	createFile(t, root+"/comp-a", "modules/ROOT/pages/index.adoc")
	createFile(t, root+"/comp-b", "modules/ROOT/pages/index.adoc")

	idx, err := index.Build(root, []*repo.Component{compA, compB})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	r := New(idx)

	ref := &model.Reference{
		RefType:      model.RefTypeXref,
		Target:       "comp-b::index.adoc",
		SrcComponent: "comp-a",
		SrcVersion:   "1.0",
		SrcModule:    "ROOT",
		SrcFamily:    model.FamilyPages,
	}
	result := r.Resolve(ref)
	if !result.Found {
		t.Error("expected to resolve comp-b::index.adoc (cross-component, default ROOT module)")
	}
}

func TestResolveAttachment_Found(t *testing.T) {
	idx, _ := setupTestIndex(t)
	r := New(idx)

	ref := &model.Reference{
		RefType:      model.RefTypeAttachment,
		Target:       "report.pdf",
		SrcComponent: "mycomp",
		SrcVersion:   "1.0",
		SrcModule:    "ROOT",
		SrcFamily:    model.FamilyPages,
	}
	result := r.Resolve(ref)
	if !result.Found {
		t.Error("expected to find report.pdf in attachments")
	}
}

func TestResolveAttachment_Missing(t *testing.T) {
	idx, _ := setupTestIndex(t)
	r := New(idx)

	ref := &model.Reference{
		RefType:      model.RefTypeAttachment,
		Target:       "nonexistent.pdf",
		SrcComponent: "mycomp",
		SrcVersion:   "1.0",
		SrcModule:    "ROOT",
		SrcFamily:    model.FamilyPages,
	}
	result := r.Resolve(ref)
	if result.Found {
		t.Error("expected not to find nonexistent.pdf")
	}
}

func TestResolveXref_FamilyPrefix(t *testing.T) {
	idx, _ := setupTestIndex(t)
	r := New(idx)

	// xref:attachment$report.pdf[Download] should resolve via attachments family
	ref := &model.Reference{
		RefType:      model.RefTypeXref,
		Target:       "attachment$report.pdf",
		SrcComponent: "mycomp",
		SrcVersion:   "1.0",
		SrcModule:    "ROOT",
		SrcFamily:    model.FamilyPages,
	}
	result := r.Resolve(ref)
	if !result.Found {
		t.Error("expected to resolve xref:attachment$report.pdf via attachments family")
	}
}

func TestResolveInclude_AntoraPartial(t *testing.T) {
	idx, _ := setupTestIndex(t)
	r := New(idx)

	ref := &model.Reference{
		RefType:      model.RefTypeInclude,
		Target:       "partial$snippet.adoc",
		SrcComponent: "mycomp",
		SrcVersion:   "1.0",
		SrcModule:    "ROOT",
		SrcFamily:    model.FamilyPages,
	}
	result := r.Resolve(ref)
	if !result.Found {
		t.Error("expected to find partial$snippet.adoc")
	}
}
