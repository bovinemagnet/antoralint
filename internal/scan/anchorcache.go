package scan

import (
	"sync"
)

// AnchorCache provides cached anchor lookups for .adoc files.
// It lazily loads anchors from files on first access and caches the results.
type AnchorCache struct {
	mu          sync.Mutex
	cache       map[string]map[string]bool
	IDPrefix    string
	IDSeparator string
}

// NewAnchorCache creates a new AnchorCache with the given Asciidoctor
// ID generation settings.
func NewAnchorCache(idPrefix, idSeparator string) *AnchorCache {
	return &AnchorCache{
		cache:       make(map[string]map[string]bool),
		IDPrefix:    idPrefix,
		IDSeparator: idSeparator,
	}
}

// HasAnchor returns true if the given file contains the specified anchor ID.
// The file is scanned on first access and the result is cached.
func (ac *AnchorCache) HasAnchor(absPath string, fragment string) (bool, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	anchors, ok := ac.cache[absPath]
	if !ok {
		var err error
		anchors, err = ExtractAnchors(absPath, ac.IDPrefix, ac.IDSeparator)
		if err != nil {
			return false, err
		}
		ac.cache[absPath] = anchors
	}

	return anchors[fragment], nil
}
