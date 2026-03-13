package index

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/bovinemagnet/antoralint/internal/model"
	"github.com/bovinemagnet/antoralint/internal/repo"
)

// Index holds all discovered resources indexed by logical ID and path.
type Index struct {
	Resources   []*model.Resource
	ByLogicalID map[string]*model.Resource
	ByAbsPath   map[string]*model.Resource
	// For case-insensitive lookups: lowercase(logicalID) -> resource
	ByLowerID map[string]*model.Resource
}

// Build creates an Index from discovered components.
func Build(rootDir string, components []*repo.Component) (*Index, error) {
	idx := &Index{
		ByLogicalID: make(map[string]*model.Resource),
		ByAbsPath:   make(map[string]*model.Resource),
		ByLowerID:   make(map[string]*model.Resource),
	}

	for _, comp := range components {
		modulesDir := filepath.Join(comp.RootDir, "modules")
		entries, err := os.ReadDir(modulesDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			moduleName := entry.Name()
			moduleDir := filepath.Join(modulesDir, moduleName)
			families := []struct {
				name model.Family
				dir  string
			}{
				{model.FamilyPages, "pages"},
				{model.FamilyPartials, "partials"},
				{model.FamilyExamples, "examples"},
				{model.FamilyImages, "images"},
				{model.FamilyAttachments, "attachments"},
			}
			for _, fam := range families {
				famDir := filepath.Join(moduleDir, fam.dir)
				idx.indexDir(rootDir, famDir, comp, moduleName, fam.name)
			}
		}
	}
	return idx, nil
}

func (idx *Index) indexDir(rootDir, dir string, comp *repo.Component, module string, family model.Family) {
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		relPath, _ := filepath.Rel(rootDir, path)
		relPath = filepath.ToSlash(relPath)

		// Compute logical resource path within family
		withinFam, _ := filepath.Rel(dir, path)
		withinFam = filepath.ToSlash(withinFam)

		logicalID := buildLogicalID(comp.Name, comp.Version, module, family, withinFam)

		r := &model.Resource{
			AbsPath:   path,
			RelPath:   relPath,
			Component: comp.Name,
			Version:   comp.Version,
			Module:    module,
			Family:    family,
			LogicalID: logicalID,
		}
		idx.Resources = append(idx.Resources, r)
		idx.ByLogicalID[logicalID] = r
		idx.ByAbsPath[path] = r
		idx.ByLowerID[strings.ToLower(logicalID)] = r
		return nil
	})
}

// buildLogicalID constructs a canonical Antora resource ID.
// Format: version@component:module:family$path
func buildLogicalID(component, version, module string, family model.Family, path string) string {
	return version + "@" + component + ":" + module + ":" + string(family) + "$" + path
}

// LookupPage returns a resource for the given component:version:module:pages$path
func (idx *Index) LookupPage(component, version, module, path string) *model.Resource {
	id := buildLogicalID(component, version, module, model.FamilyPages, path)
	return idx.ByLogicalID[id]
}

// LookupByAbsPath returns a resource by absolute path.
func (idx *Index) LookupByAbsPath(absPath string) *model.Resource {
	return idx.ByAbsPath[absPath]
}
