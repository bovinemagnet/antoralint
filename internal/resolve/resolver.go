package resolve

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/bovinemagnet/antoralint/internal/index"
	"github.com/bovinemagnet/antoralint/internal/model"
)

// Result represents the result of resolving a reference.
type Result struct {
	Ref               *model.Reference
	Resource          *model.Resource // nil if not found
	Found             bool
	CaseMismatch      bool
	HasUnresolvedAttr bool
}

// Resolver resolves references against an index.
type Resolver struct {
	idx *index.Index
}

// New creates a new Resolver.
func New(idx *index.Index) *Resolver {
	return &Resolver{idx: idx}
}

// Resolve attempts to resolve a reference and returns a Result.
func (r *Resolver) Resolve(ref *model.Reference) *Result {
	result := &Result{Ref: ref}

	// Check for unresolved attributes
	if strings.Contains(ref.Target, "{") && strings.Contains(ref.Target, "}") {
		result.HasUnresolvedAttr = true
		return result
	}

	switch ref.RefType {
	case model.RefTypeXref:
		return r.resolveXref(ref)
	case model.RefTypeInclude:
		return r.resolveInclude(ref)
	case model.RefTypeImage:
		return r.resolveImage(ref)
	case model.RefTypeAttachment:
		return r.resolveAttachment(ref)
	}
	return result
}

// resolveXref resolves an xref: reference.
// Antora xref format: [version@][component:][[module:]page[.adoc]][#fragment]
func (r *Resolver) resolveXref(ref *model.Reference) *Result {
	result := &Result{Ref: ref}
	target := ref.Target

	component := ref.SrcComponent
	version := ref.SrcVersion
	module := ref.SrcModule
	pagePath := target

	// Handle version@
	if idx := strings.Index(target, "@"); idx >= 0 {
		version = target[:idx]
		target = target[idx+1:]
	}

	// Handle component: and optional module:
	if colonIdx := strings.Index(target, ":"); colonIdx >= 0 {
		component = target[:colonIdx]
		target = target[colonIdx+1:]
		if colonIdx2 := strings.Index(target, ":"); colonIdx2 >= 0 {
			module = target[:colonIdx2]
			pagePath = target[colonIdx2+1:]
		} else {
			pagePath = target
		}
	} else {
		pagePath = target
	}

	// Normalize page path - add .adoc if no extension present
	if !strings.HasSuffix(pagePath, ".adoc") && !strings.Contains(pagePath, ".") {
		pagePath += ".adoc"
	}

	logicalID := version + "@" + component + ":" + module + ":pages$" + pagePath
	if res := r.idx.ByLogicalID[logicalID]; res != nil {
		result.Resource = res
		result.Found = true
		return result
	}

	// Try case-insensitive match
	if res := r.idx.ByLowerID[strings.ToLower(logicalID)]; res != nil {
		result.Resource = res
		result.Found = true
		result.CaseMismatch = true
		return result
	}

	return result
}

// resolveInclude resolves an include:: target.
func (r *Resolver) resolveInclude(ref *model.Reference) *Result {
	result := &Result{Ref: ref}
	target := ref.Target

	// Handle Antora resource ID form containing $ (partial$path, example$path, etc.)
	if strings.Contains(target, "$") {
		return r.resolveAntoraFamilyInclude(ref, target)
	}

	// Relative path include: resolve relative to source file dir
	srcDir := filepath.Dir(ref.SourceFile)
	absPath := filepath.Clean(filepath.Join(srcDir, filepath.FromSlash(target)))

	if res := r.idx.ByAbsPath[absPath]; res != nil {
		result.Resource = res
		result.Found = true
		return result
	}

	// Check file existence even if not indexed
	if fileExists(absPath) {
		result.Found = true
		return result
	}

	// Try case-insensitive match
	lowerAbs := strings.ToLower(absPath)
	for k, v := range r.idx.ByAbsPath {
		if strings.ToLower(k) == lowerAbs {
			result.Resource = v
			result.Found = true
			result.CaseMismatch = true
			return result
		}
	}

	return result
}

func (r *Resolver) resolveAntoraFamilyInclude(ref *model.Reference, target string) *Result {
	result := &Result{Ref: ref}

	component := ref.SrcComponent
	version := ref.SrcVersion
	module := ref.SrcModule

	if idx := strings.Index(target, "@"); idx >= 0 {
		version = target[:idx]
		target = target[idx+1:]
	}

	if idx := strings.Index(target, "$"); idx >= 0 {
		prefix := target[:idx]
		path := target[idx+1:]

		var family model.Family
		// prefix could be: family, module:family, component:module:family
		parts := strings.Split(prefix, ":")
		switch len(parts) {
		case 1:
			family = familyFromString(parts[0])
		case 2:
			module = parts[0]
			family = familyFromString(parts[1])
		case 3:
			component = parts[0]
			module = parts[1]
			family = familyFromString(parts[2])
		}

		logicalID := version + "@" + component + ":" + module + ":" + string(family) + "$" + path
		if res := r.idx.ByLogicalID[logicalID]; res != nil {
			result.Resource = res
			result.Found = true
			return result
		}

		// Try case-insensitive
		if res := r.idx.ByLowerID[strings.ToLower(logicalID)]; res != nil {
			result.Resource = res
			result.Found = true
			result.CaseMismatch = true
			return result
		}
	}

	return result
}

// resolveImage resolves an image:: or image: target.
func (r *Resolver) resolveImage(ref *model.Reference) *Result {
	result := &Result{Ref: ref}
	target := ref.Target

	// Antora image resource ID containing $ or cross-component with :
	if strings.Contains(target, "$") || strings.Contains(target, ":") {
		return r.resolveAntoraImageRef(ref, target)
	}

	// Simple lookup in source module's images directory
	component := ref.SrcComponent
	version := ref.SrcVersion
	module := ref.SrcModule

	logicalID := version + "@" + component + ":" + module + ":images$" + target
	if res := r.idx.ByLogicalID[logicalID]; res != nil {
		result.Resource = res
		result.Found = true
		return result
	}

	// Try case-insensitive
	if res := r.idx.ByLowerID[strings.ToLower(logicalID)]; res != nil {
		result.Resource = res
		result.Found = true
		result.CaseMismatch = true
		return result
	}

	return result
}

func (r *Resolver) resolveAntoraImageRef(ref *model.Reference, target string) *Result {
	result := &Result{Ref: ref}

	component := ref.SrcComponent
	version := ref.SrcVersion
	module := ref.SrcModule

	if idx := strings.Index(target, "@"); idx >= 0 {
		version = target[:idx]
		target = target[idx+1:]
	}

	if dollarIdx := strings.Index(target, "$"); dollarIdx >= 0 {
		prefix := target[:dollarIdx]
		path := target[dollarIdx+1:]
		parts := strings.Split(prefix, ":")
		switch len(parts) {
		case 2:
			module = parts[0]
		case 3:
			component = parts[0]
			module = parts[1]
		}
		logicalID := version + "@" + component + ":" + module + ":images$" + path
		if res := r.idx.ByLogicalID[logicalID]; res != nil {
			result.Resource = res
			result.Found = true
			return result
		}
		if res := r.idx.ByLowerID[strings.ToLower(logicalID)]; res != nil {
			result.Resource = res
			result.Found = true
			result.CaseMismatch = true
			return result
		}
	} else {
		// component:path form
		parts := strings.SplitN(target, ":", 2)
		if len(parts) == 2 {
			component = parts[0]
			target = parts[1]
		}
		logicalID := version + "@" + component + ":" + module + ":images$" + target
		if res := r.idx.ByLogicalID[logicalID]; res != nil {
			result.Resource = res
			result.Found = true
			return result
		}
	}

	return result
}

// resolveAttachment resolves an attachment reference.
func (r *Resolver) resolveAttachment(ref *model.Reference) *Result {
	result := &Result{Ref: ref}
	target := ref.Target
	component := ref.SrcComponent
	version := ref.SrcVersion
	module := ref.SrcModule

	logicalID := version + "@" + component + ":" + module + ":attachments$" + target
	if res := r.idx.ByLogicalID[logicalID]; res != nil {
		result.Resource = res
		result.Found = true
		return result
	}
	if res := r.idx.ByLowerID[strings.ToLower(logicalID)]; res != nil {
		result.Resource = res
		result.Found = true
		result.CaseMismatch = true
		return result
	}
	return result
}

func familyFromString(s string) model.Family {
	switch strings.ToLower(s) {
	case "pages":
		return model.FamilyPages
	case "partial", "partials":
		return model.FamilyPartials
	case "example", "examples":
		return model.FamilyExamples
	case "image", "images":
		return model.FamilyImages
	case "attachment", "attachments":
		return model.FamilyAttachments
	}
	return model.FamilyUnknown
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
