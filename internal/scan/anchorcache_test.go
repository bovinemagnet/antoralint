package scan

import (
	"testing"
)

func TestAnchorCache_HasAnchor(t *testing.T) {
	content := `= Page Title

== My Section

[[custom-id]]
Paragraph.
`
	f := writeTempFile(t, "test.adoc", content)
	cache := NewAnchorCache("_", "_")

	// First call: loads from file
	found, err := cache.HasAnchor(f, "_my_section")
	if err != nil {
		t.Fatalf("HasAnchor error: %v", err)
	}
	if !found {
		t.Error("expected _my_section to be found")
	}

	// Verify custom-id also found
	found, err = cache.HasAnchor(f, "custom-id")
	if err != nil {
		t.Fatalf("HasAnchor error: %v", err)
	}
	if !found {
		t.Error("expected custom-id to be found")
	}

	// Missing anchor
	found, err = cache.HasAnchor(f, "nonexistent")
	if err != nil {
		t.Fatalf("HasAnchor error: %v", err)
	}
	if found {
		t.Error("expected nonexistent to not be found")
	}
}

func TestAnchorCache_CachesResults(t *testing.T) {
	content := `= Page

== Section
`
	f := writeTempFile(t, "test.adoc", content)
	cache := NewAnchorCache("_", "_")

	// Load the cache
	_, err := cache.HasAnchor(f, "_section")
	if err != nil {
		t.Fatalf("HasAnchor error: %v", err)
	}

	// Verify cache entry exists
	cache.mu.Lock()
	_, cached := cache.cache[f]
	cache.mu.Unlock()

	if !cached {
		t.Error("expected file to be cached after first access")
	}
}
