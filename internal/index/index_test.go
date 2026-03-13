package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bovinemagnet/antoralint/internal/model"
	"github.com/bovinemagnet/antoralint/internal/repo"
)

func TestBuild_IndexesPages(t *testing.T) {
	root := t.TempDir()
	createFixture(t, root, "mycomp", "1.0", "ROOT", "pages", "index.adoc")
	createFixture(t, root, "mycomp", "1.0", "ROOT", "pages", "guide.adoc")

	components := []*repo.Component{{Name: "mycomp", Version: "1.0", RootDir: root}}
	idx, err := Build(root, components)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	if len(idx.Resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(idx.Resources))
	}

	res := idx.LookupPage("mycomp", "1.0", "ROOT", "index.adoc")
	if res == nil {
		t.Error("expected to find index.adoc page")
	}
	if res != nil && res.Family != model.FamilyPages {
		t.Errorf("expected family pages, got %s", res.Family)
	}
}

func TestBuild_MultipleModules(t *testing.T) {
	root := t.TempDir()
	createFixture(t, root, "mycomp", "1.0", "ROOT", "pages", "index.adoc")
	createFixture(t, root, "mycomp", "1.0", "admin", "pages", "settings.adoc")

	components := []*repo.Component{{Name: "mycomp", Version: "1.0", RootDir: root}}
	idx, err := Build(root, components)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	if len(idx.Resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(idx.Resources))
	}

	res := idx.LookupPage("mycomp", "1.0", "admin", "settings.adoc")
	if res == nil {
		t.Error("expected to find admin settings.adoc page")
	}
}

func TestBuild_CaseInsensitiveLookup(t *testing.T) {
	root := t.TempDir()
	createFixture(t, root, "mycomp", "1.0", "ROOT", "pages", "MyPage.adoc")

	components := []*repo.Component{{Name: "mycomp", Version: "1.0", RootDir: root}}
	idx, err := Build(root, components)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	// Exact match
	res := idx.LookupPage("mycomp", "1.0", "ROOT", "MyPage.adoc")
	if res == nil {
		t.Error("expected to find MyPage.adoc")
	}

	// Case-insensitive via ByLowerID (entire key is lowercased)
	id := "1.0@mycomp:root:pages$mypage.adoc"
	resLower := idx.ByLowerID[id]
	if resLower == nil {
		t.Error("expected case-insensitive lookup to find MyPage.adoc")
	}
}

func createFixture(t *testing.T, root, comp, ver, module, family, filename string) {
	t.Helper()
	dir := filepath.Join(root, "modules", module, family)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte("= "+filename), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
