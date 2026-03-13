package repo

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// AntoraYML represents the parsed content of an antora.yml file.
type AntoraYML struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Title   string `yaml:"title"`
}

// Component represents a discovered Antora component.
type Component struct {
	Name    string
	Version string
	RootDir string // directory containing antora.yml
}

// Discover finds all Antora components under rootDir.
func Discover(rootDir string) ([]*Component, error) {
	var components []*Component

	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if !d.IsDir() && d.Name() == "antora.yml" {
			comp, err := parseAntoraYML(path)
			if err == nil {
				components = append(components, comp)
			}
		}
		return nil
	})
	return components, err
}

func parseAntoraYML(path string) (*Component, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var yml AntoraYML
	if err := yaml.Unmarshal(data, &yml); err != nil {
		return nil, err
	}
	version := yml.Version
	if version == "" {
		version = "_"
	}
	return &Component{
		Name:    yml.Name,
		Version: version,
		RootDir: filepath.Dir(path),
	}, nil
}
