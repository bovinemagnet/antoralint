# Product Requirements Document: Antora/AsciiDoc Repository Linter

> **Implementation Status as of 2026-03-26:** The MVP and Milestones 1–3 are complete. The tool is functional with a 6-stage pipeline, 9 implemented rules, 3 output formats (text, JSON, SARIF), 8 CLI flags, and 5 fixture repositories. See the [Implementation Status](#25-implementation-status) section for full details.

## 1. Overview

### 1.1 Product Name

**adoclint**
A command-line tool written in **Go** that scans an **Antora-based AsciiDoc repository** for broken references and structural issues.

**Module path:** `github.com/bovinemagnet/antoralint`
**Go version:** 1.24.13
**External dependencies:** `github.com/spf13/cobra` (CLI), `gopkg.in/yaml.v3` (Antora config parsing)

### 1.2 Purpose
Documentation repositories often accumulate broken `xref`, `include`, image, and attachment references over time, especially in multi-module or multi-version Antora sites. These errors are easy to miss during manual review and often only surface during site generation or after publishing.

The purpose of this tool is to provide a **fast, CI-friendly static analysis utility** that detects these issues early and reports them clearly, with actionable diagnostics.

### 1.3 Product Principle for v1
The first version will be intentionally **pragmatic, fast, and maintainable**. It will **not** try to fully parse or emulate all AsciiDoc and Antora behavior up front.

The design goal for v1 is to deliver high-value validation quickly by focusing on common, high-confidence cases:
- broken `xref`
- broken `include`
- broken `image`
- missing attachments where detectable
- clear file and line diagnostics
- CI-friendly execution and exit codes

This principle should guide product and engineering decisions whenever scope is unclear.

### 1.4 Goals
The product should:

- scan an Antora-based AsciiDoc repository
- detect broken internal references and missing resources
- produce clear diagnostics with file and line information
- run locally and in CI
- support machine-readable output for automation
- be simple to install and distribute as a single Go binary

### 1.5 Non-Goals
The first version will **not** attempt to:

- fully implement the complete AsciiDoc parser specification
- fully emulate Antora site generation
- render or preview pages
- automatically fix all issues
- validate content semantics beyond reference integrity
- resolve every possible dynamic attribute expansion scenario
- act as a language server or IDE plugin

---

## 2. Problem Statement

Teams using Antora and AsciiDoc often face these recurring issues:

- `xref:` targets point to pages that no longer exist
- `include::` directives reference missing files
- `image::` macros reference missing images
- attachment links refer to absent assets
- references work on one platform but fail on another because of case mismatches
- broken links are introduced in pull requests and detected too late
- large documentation repositories are difficult to validate manually

There is a need for a dedicated repo-aware linter that understands enough of Antora’s structure and AsciiDoc reference patterns to find these problems quickly and reliably.

---

## 3. Users and Use Cases

### 3.1 Primary Users
- technical writers maintaining Antora documentation
- developers contributing to docs alongside code
- DevOps and platform teams integrating documentation validation into CI/CD
- documentation architects responsible for repo health

### 3.2 Key Use Cases

#### Local developer validation
A developer runs the tool before committing changes and sees broken `xref` and `include` diagnostics.

#### Pull request validation
The CI pipeline runs the tool and fails the build if broken documentation references are introduced.

#### Repository health audit
A documentation team runs the tool across an entire docs mono-repo to identify legacy issues.

#### Machine-readable reporting
An engineering system consumes JSON or SARIF output to annotate pull requests or dashboards.

---

## 4. Scope

### 4.1 In Scope for v1
The v1 product will support:

- recursive scanning of an Antora repository — **implemented**
- discovery of Antora components, versions, modules, and resource families — **implemented**
- parsing of `.adoc` files using a pragmatic scanner — **implemented**
- validation of:
  - `xref:` — **implemented** (same-module, cross-module, cross-component, with version/fragment support)
  - `include::` — **implemented** (relative paths, Antora family form with `$` prefix)
  - `image::` — **implemented** (block and inline forms, cross-module support)
  - attachment-style references where feasible — **implemented** (scanner extracts `link:{attachmentsdir}/...` patterns, resolver validates against index)
- optional validation of external HTTP/HTTPS links — **implemented** (opt-in via `--external-links` flag with `--timeout` and `--concurrency` controls)
- text and machine-readable output formats — **implemented** (text, JSON, and SARIF 2.1.0)
- non-zero exit codes for CI failure conditions — **implemented**

### 4.2 Out of Scope for v1
The v1 product will not support:

- full Antora playbook execution
- full AsciiDoc substitution engine
- live preview
- automatic rewriting of references
- editor plugins
- checking every anchor in every possible inline syntax variant unless explicitly supported
- remote component catalog resolution beyond local repo structures unless added later

---

## 5. Product Vision

Build a **fast, practical, Antora-aware documentation linter** that delivers immediate value in local development and CI, while keeping the architecture clean enough to evolve into a more capable static analysis platform over time.

The product should favor:

- correctness of high-value checks
- fast feedback
- straightforward installation
- understandable diagnostics
- simple extensibility for new rules

---

## 6. Product Strategy

### 6.1 v1 Strategy
The first release should optimize for:
- high signal over completeness
- low false positives
- fast execution
- simple code structure
- easy CI adoption

The product should deliberately avoid getting blocked by edge-case completeness in AsciiDoc parsing. When faced with an ambiguous or highly dynamic construct, the preferred v1 behavior is:
1. resolve it if it can be resolved with high confidence
2. emit a warning if it cannot be validated confidently
3. avoid pretending to fully understand constructs the tool does not yet support

### 6.2 Design Tradeoff
This product is not a general-purpose AsciiDoc engine in v1. It is a repository linter.

That means:
- repository indexing matters more than full document rendering semantics
- accurate resolution of common references matters more than exhaustive grammar support
- maintainability matters more than theoretical completeness

---

## 7. Functional Requirements

### 7.1 Repository Discovery — **Implemented**

The tool must:

- accept a root directory as input — **done** (`internal/repo/discover.go`)
- recursively scan the repository for Antora content — **done** (skips hidden directories)
- detect `antora.yml` files — **done** (parses `name`, `version`, `title`)
- identify components, versions, modules, and resource families — **done** (defaults version to `"_"` when unspecified)
- index resources under standard Antora families such as:
  - `pages` — **done**
  - `partials` — **done**
  - `examples` — **done**
  - `images` — **done**
  - `attachments` — **done**

The tool should:

- handle mono-repos containing multiple Antora components — **done**
- normalise paths consistently across operating systems — **done** (forward-slash normalisation)
- retain case-sensitive path information for validation — **done** (three lookup maps: exact, absolute path, and case-insensitive)

---

### 7.2 File Scanning — **Implemented**

The tool must:

- scan `.adoc` files — **done** (`internal/scan/scanner.go`)
- detect relevant macros and reference patterns — **done** (regex patterns: `xref:`, `include::`, `image::`, `image:`, `link:{attachmentsdir}/...`, `link:https://...`, bare `https://...` URLs)
- capture file path, line number, and where practical column number — **done** (line and column captured)
- avoid obvious false positives in comments and code/literal blocks where feasible — **done** (skips `//` comment lines; tracks block context for `----`, `....`, and backtick-fenced blocks; `include::` always scanned as AsciiDoc processes includes before other substitutions)

The tool uses a **structured scanner approach** (`internal/scan/scanner.go`):

- line-oriented processing — **done**
- context-aware block tracking (code blocks, literal blocks) — **done**
- easy to extend with new reference extractors — **done** (add new regex pattern and extraction logic)

---

### 7.3 Reference Validation

#### 7.3.1 `xref:` validation — **Implemented**
The tool validates `xref:` references for local Antora resources (`internal/resolve/resolver.go`).

It detects:

- missing target page — **done** (emits `broken-xref` error)
- invalid family or module references — **done** (colon-count parsing: 0 = same module, 1 = cross-module, 2+ = cross-component)
- invalid component or version references where explicitly provided — **done** (supports `version@component:module:page` form)
- malformed xref syntax where supported by scanner rules — **done** (via regex extraction)

It supports:

- relative xrefs — **done**
- module-relative xrefs — **done** (`module:page` form)
- page references with fragments — **done** (fragments extracted after `#`, not yet validated against page content)
- Antora-style resource ID forms commonly used in documentation repos — **done** (logical ID format: `version@component:module:pages$path`)

It warns, rather than fails hard, on unresolved cases involving attributes (`{...}` in target).

#### 7.3.2 `include::` validation — **Implemented**
The tool validates `include::` targets (`internal/resolve/resolver.go`).

It detects:

- missing include file — **done** (emits `broken-include` error)
- invalid relative path — **done** (resolved relative to source file directory, checked on filesystem)
- invalid partial or example target where identifiable — **done** (Antora family form with `$` prefix, e.g. `partial$snippet.adoc`)
- include cycles — **implemented** (`internal/cycles/cycles.go`, emits `include-cycle` error)

It supports:

- standard include targets — **done**
- Antora-family includes where used conventionally — **done** (supports `family$path`, `module:family$path`, `component:module:family$path`, and `version@` prefix)
- reporting of include chains for easier debugging — **not yet implemented**

#### 7.3.3 `image::` validation — **Implemented**
The tool validates image references (`internal/resolve/resolver.go`).

It detects:

- missing image files — **done** (emits `broken-image` error; supports both block `image::` and inline `image:`)
- invalid relative image paths — **done** (looks in source module's images directory)
- case mismatch between referenced path and actual file path — **done** (emits `case-mismatch` warning via case-insensitive fallback)

#### 7.3.4 Attachment reference validation — **Implemented**
The attachment scanner, resolver, and rule are all implemented. The scanner pattern `link:{attachmentsdir}/target[label]` extracts attachment references from `.adoc` files, and the resolver validates them against the index.

Where Antora attachment references are detectable, the tool validates:

- missing attachment files — **implemented** (emits `broken-attachment` error)
- invalid attachment paths — **implemented**
- case mismatch — **implemented** (case-insensitive fallback)

---

### 7.4 External Link Validation — **Implemented**

This feature is optional and disabled by default. When enabled via the `--external-links` flag, the scanner extracts `link:https://...[label]` macros and bare `https://...` URLs from `.adoc` files. The link checker (`internal/linkcheck/`) validates each URL using HEAD first with GET fallback on HTTP 405 Method Not Allowed, bounded concurrency via semaphore (default 5, configurable via `--concurrency`), and configurable per-request timeout (default 10s, configurable via `--timeout`).

The tool:

- validates HTTP and HTTPS links — **done**
- uses bounded concurrency — **done** (semaphore-based, default 5)
- applies request timeouts — **done** (default 10s)
- follows a limited number of redirects — **done**
- records status code or failure reason — **done**

The tool avoids making the entire scan unreliable due to flaky external sites — **addressed** (transient failures emit warnings via `external-link-timeout`, confirmed dead links emit errors via `external-link-dead`).

The tool:

- treats transient network failures differently from definite dead links — **done** (timeout = WARNING, dead = ERROR)
- support retry logic with conservative defaults — **done**
- allow users to skip specific domains in future versions — **not yet implemented**

---

### 7.5 Diagnostics — **Implemented**

The tool produces diagnostics (`internal/model/types.go` `Diagnostic` struct) containing:

- severity — **done** (`error`, `warning`, `info`)
- rule ID — **done**
- message — **done**
- file path — **done**
- line number — **done**
- optional column number — **done**
- reference type — **done** (via rule ID)
- target value — **done**
- optional resolution context — **done** (`Fix` field for suggested corrections)

Implemented diagnostic categories:

- `broken-xref` — **implemented** (ERROR)
- `broken-include` — **implemented** (ERROR)
- `broken-image` — **implemented** (ERROR)
- `broken-attachment` — **implemented** (ERROR)
- `external-link-dead` — **implemented** (ERROR)
- `external-link-timeout` — **implemented** (WARNING)
- `unresolved-attribute` — **implemented** (WARNING)
- `case-mismatch` — **implemented** (WARNING)

The tool summarises diagnostics (error/warning counts) at the end of a run — **done** (`report.Summary()` writes to stderr).

---

### 7.6 Output Formats — **Fully Implemented**

All three output formats are implemented in `internal/report/report.go`:

- human-readable text output — **done** (format: `SEVERITY rule-id file:line message`)
- JSON output — **done** (JSON array with `severity`, `ruleId`, `file`, `line`, `column`, `message`, `target` fields)
- SARIF 2.1.0 output — **done** (full SARIF compliance with tool metadata, physical locations, and severity mapping)

Text output is concise and useful in terminal workflows.

JSON and SARIF output preserve structured fields for downstream automation.

---

### 7.7 Exit Codes — **Implemented**

The tool provides CI-friendly exit codes (`cmd/adoclint/main.go`):

- `0` = no issues found (or all issues below configured threshold)
- `1` = issues found above configured `--fail-on` threshold
- `2` = runtime/configuration error

The `--fail-on` flag controls the threshold: `error` (default), `warning`, or `none`.

---

### 7.8 Configuration — **Partially Implemented**

The v1 tool supports basic configuration through command-line flags (`cmd/adoclint/main.go`).

Later versions may add config file support.

Implemented configuration options:

- root directory to scan — **done** (positional argument)
- output format — **done** (`--format text|json|sarif`)
- severity threshold for failure — **done** (`--fail-on error|warning|none`)
- include or exclude path patterns — **done** (`--include`, `--exclude` flags)
- verbose logging — **done** (`--verbose` flag)

Also implemented:

- enable or disable external link checking — **done** (`--external-links` flag, opt-in)
- maximum concurrency for external link checks — **done** (`--concurrency` flag, default 5)
- request timeout for external link checks — **done** (`--timeout` flag, default 10s)

---

## 8. Non-Functional Requirements

### 8.1 Performance
The tool should be fast enough to run in normal developer workflows.

Targets:

- small repo: complete in under a few seconds
- medium repo: complete comfortably in CI without noticeable friction
- large repo: scale with concurrency for I/O-bound work

Performance matters, but correctness and diagnostic quality matter more than micro-optimization.

### 8.2 Portability
The tool must run on:

- Linux
- macOS
- Windows

It should compile to a single static or near-static binary where practical.

### 8.3 Reliability
The tool must:

- fail clearly on invalid usage
- avoid panics in normal malformed-content cases
- continue scanning after encountering individual file issues
- distinguish content errors from runtime errors

### 8.4 Maintainability
The codebase should be structured into clear packages and rules, making it easy to add:

- new reference types
- new validation rules
- new output formats
- better Antora resolution behavior

### 8.5 Observability
The tool should provide optional verbose logging for debugging resolution logic, without cluttering standard output by default.

---

## 9. CLI Requirements

### 9.1 Command Structure — **Implemented**

The tool uses the Cobra CLI framework. Implemented commands:

```bash
adoclint scan ./docs                    # text output (default)
adoclint scan ./docs --format json      # JSON output
adoclint scan ./docs --format sarif     # SARIF 2.1.0 output
adoclint scan ./docs --fail-on warning  # fail on warnings too
adoclint scan ./docs --verbose          # verbose logging
```

Also available:
```bash
adoclint scan ./docs --external-links   # enable external link checking
adoclint scan ./docs --external-links --timeout 15s --concurrency 10
```

### 9.2 Flags

Implemented flags:

```text
--format <text|json|sarif>       ✓ implemented (default: text)
--fail-on <error|warning|none>   ✓ implemented (default: error)
--include <glob>                 ✓ implemented
--exclude <glob>                 ✓ implemented
--verbose                        ✓ implemented
```

Also implemented:

```text
--external-links                 ✓ implemented (opt-in, default: disabled)
--timeout <duration>             ✓ implemented (default: 10s)
--concurrency <n>                ✓ implemented (default: 5)
```

---

## 10. Architecture

### 10.1 High-Level Design

The system should use a staged pipeline:

1. **Repository discovery**
2. **Resource indexing**
3. **File scanning**
4. **Reference resolution**
5. **Rule evaluation**
6. **Reporting**

This approach keeps concerns separated and makes testing easier.

### 10.2 Package Layout — **Implemented**

The suggested layout has been implemented exactly as designed:

```text
cmd/adoclint/main.go         — CLI entry point, pipeline orchestration
internal/model/types.go      — Domain types (Resource, Reference, Diagnostic, enums)
internal/repo/discover.go    — Antora component discovery
internal/index/index.go      — Resource indexing with 3 lookup maps
internal/scan/scanner.go     — Reference extraction from .adoc files
internal/resolve/resolver.go — Antora-aware reference resolution
internal/rules/rules.go      — Rule evaluation and diagnostic generation
internal/report/report.go    — Output formatting (text, JSON, SARIF)
```

### 10.3 Package Responsibilities

#### `repo`
Discovers Antora structures and filesystem resources.

#### `index`
Builds a normalized in-memory representation of components, modules, pages, partials, images, and attachments.

#### `scan`
Scans `.adoc` files and extracts candidate references with location metadata.

#### `resolve`
Resolves extracted references against the indexed repository model.

#### `rules`
Converts unresolved or suspicious references into diagnostics.

#### `report`
Formats diagnostics as text, JSON, or SARIF.

#### `model`
Contains shared domain types such as resources, references, diagnostics, and enums.

---

## 11. Data Model

### 11.1 Core Domain Types — **Implemented**

All domain types are defined in `internal/model/types.go`.

#### Resource — **Implemented**
Represents a discovered repo resource.

Fields:

- `AbsPath` — absolute filesystem path
- `RelPath` — repo-relative path
- `Component` — Antora component name
- `Version` — component version (defaults to `"_"`)
- `Module` — module name (e.g. `ROOT`, `admin`)
- `Family` — resource family (`pages`, `partials`, `examples`, `images`, `attachments`)
- `LogicalID` — Antora resource ID (`version@component:module:family$path`)

#### Reference — **Implemented**
Represents a parsed reference from an AsciiDoc file.

Fields:

- `SourceFile` — file containing the reference
- `Line` — line number
- `Column` — column number
- `RawText` — original matched text
- `RefType` — reference type (`xref`, `include`, `image`, `attachment`)
- `Target` — resolved target string
- `Fragment` — fragment/anchor after `#` (if present)
- `SrcComponent`, `SrcVersion`, `SrcModule`, `SrcFamily` — source context

#### Diagnostic — **Implemented**
Represents a reported issue.

Fields:

- `Severity` — `error`, `warning`, or `info`
- `RuleID` — rule identifier (e.g. `broken-xref`)
- `Message` — human-readable description
- `File` — file path
- `Line` — line number
- `Column` — column number
- `Target` — raw target value
- `Fix` — suggested fix (if available)

#### Supporting Enums — **Implemented**

- `Severity` — `SeverityError`, `SeverityWarning`, `SeverityInfo`
- `Family` — `FamilyPages`, `FamilyPartials`, `FamilyExamples`, `FamilyImages`, `FamilyAttachments`, `FamilyUnknown`
- `RefType` — `RefTypeXref`, `RefTypeInclude`, `RefTypeImage`, `RefTypeAttachment`

---

## 12. Parsing and Resolution Strategy

### 12.1 Parsing Strategy
The tool should not begin with a full AsciiDoc parser.

Instead, it should use a pragmatic scanner that:

- reads files line by line
- detects known macro forms
- tracks simple block context
- extracts target strings and source locations

This is the right balance between implementation cost and useful correctness.

### 12.2 Resolution Strategy
Resolution should be Antora-aware.

For each reference, the resolver should consider:

- current file family
- current module
- current component and version
- target family
- relative path semantics
- explicit component/version/module prefixes where present

Resolution logic should be isolated so rules can evolve without rewriting scanning code.

### 12.3 v1 Resolution Policy
The resolver should prefer **high-confidence outcomes**:
- resolve when the target can be matched deterministically
- warn when attributes or uncommon syntax prevent confident validation
- avoid inventing semantics the tool does not truly support

This policy is critical to keeping v1 maintainable and trustworthy.

---

## 13. Rules for v1

### 13.1 Required Rules — **All Implemented**

All six required rules are implemented in `internal/rules/rules.go`:

#### R001 Broken xref — **Implemented**
Report when an `xref:` target cannot be resolved to an indexed page.
Rule ID: `broken-xref`, Severity: ERROR

#### R002 Broken include — **Implemented**
Report when an `include::` target cannot be resolved to an existing file.
Rule ID: `broken-include`, Severity: ERROR

#### R003 Broken image — **Implemented**
Report when an `image::` target cannot be resolved to an existing image.
Rule ID: `broken-image`, Severity: ERROR

#### R004 Broken attachment — **Implemented**
Report when an attachment reference target cannot be resolved. The scanner extracts `link:{attachmentsdir}/...` patterns, and the resolver validates them against the index.
Rule ID: `broken-attachment`, Severity: ERROR

#### R005 Case mismatch — **Implemented**
Report when a referenced path differs from the actual file path only by case. Uses the case-insensitive lookup map as fallback.
Rule ID: `case-mismatch`, Severity: WARNING

#### R006 Unresolved attribute in target — **Implemented**
Warn when a target contains `{...}` attributes that were not resolved. Early-return before resolution to avoid false positives.
Rule ID: `unresolved-attribute`, Severity: WARNING

### 13.2 Optional Rules for v1 or v1.x — **Implemented**

#### R007 Include cycle — **Implemented**
Detect simple include cycles and report the include chain. Implemented in `internal/cycles/cycles.go` and `internal/rules/rules.go`.
Rule ID: `include-cycle`, Severity: ERROR

#### R008 Dead external link — **Implemented**
Report confirmed dead external URLs. Implemented as the `external-link-dead` rule.
Rule ID: `external-link-dead`, Severity: ERROR

#### R009 External link timeout — **Implemented**
Warn when an external URL cannot be validated due to timeout or transient failure. Implemented as the `external-link-timeout` rule.
Rule ID: `external-link-timeout`, Severity: WARNING

---

## 14. Error Handling

The tool must clearly distinguish:

- content validation issues
- configuration errors
- filesystem errors
- network errors
- unexpected internal errors

It should continue scanning after recoverable failures and produce a summary at the end.

---

## 15. Testing Requirements

### 15.1 Unit Tests — **Implemented** (14 tests across 4 packages)
The project includes unit tests for:

- scanner extraction logic — **done** (5 tests in `internal/scan/scanner_test.go`: xref detection, include detection, image detection, comment skipping, code block skipping)
- xref resolution — **done** (4 tests in `internal/resolve/resolver_test.go`: same-page, cross-module, missing, unresolved attribute)
- include resolution — **done** (1 test: Antora partial resolution)
- image resolution — **done** (2 tests: found and missing)
- diagnostic generation — **done** (4 tests in `internal/rules/rules_test.go`: broken-xref, unresolved-attribute, found-no-issue, case-mismatch)
- index building — **done** (3 tests in `internal/index/index_test.go`: pages indexing, multiple modules, case-insensitive lookup)
- path normalisation — **not yet tested directly**

### 15.2 Golden Tests — **Implemented**
The project includes 6 golden files in `testdata/golden/`:

- `broken-text` — text output for broken references
- `broken-json` — JSON output for broken references
- `broken-sarif` — SARIF output for broken references
- `casemismatch-text` — text output for case mismatch scenarios
- `cycles-text` — text output for include cycle detection
- `multicomponent-text` — text output for multi-component layouts

### 15.3 Fixture Repositories — **Implemented**
The project includes five fixture repos under `testdata/fixtures/`:

- `simple/` — valid Antora structure with working references (multi-module: ROOT + admin)
- `broken/` — broken xrefs, broken includes, broken images, unresolved attributes
- `casemismatch/` — case mismatch scenarios for cross-platform compatibility testing
- `cycles/` — include cycle detection scenarios
- `multicomponent/` — multi-component layouts

Not yet covered:

- multi-version layouts

### 15.4 Cross-Platform Testing — **Not Yet Implemented**
The tool has not yet been tested for path and case behaviour across major OS environments.

---

## 16. Security and Safety Considerations

The tool must:

- avoid arbitrary code execution
- treat repository content as untrusted input
- avoid following unsafe paths outside the repo unless explicitly allowed
- handle symlinks conservatively
- impose timeouts and concurrency limits on external link checks

If future config files are added, they should be parsed safely without executing embedded scripts.

---

## 17. Success Metrics

The product will be considered successful if it achieves the following:

- detects the majority of high-value broken-reference issues in target Antora repos
- produces diagnostics clear enough for users to fix issues without confusion
- runs fast enough to be adopted in local development and CI
- is easy to install and run as a single binary
- can be extended with new rules without major refactoring

Practical adoption metrics may include:

- CI integration in at least one target repo
- local use by contributors before pull requests
- measurable reduction in broken docs links over time

---

## 18. Milestones

### Milestone 1: MVP — **Complete**
Delivered:

- repo discovery — **done**
- indexing of Antora resources — **done**
- `.adoc` scanning — **done**
- broken `xref`, `include`, and `image` detection — **done**
- text output — **done**
- CI exit codes — **done**

### Milestone 2: Automation Readiness — **Complete**
Delivered:

- JSON output — **done**
- better diagnostics — **done** (file, line, column, target, fix fields)
- path include/exclude controls — **done** (`--include`, `--exclude` flags)
- improved handling of unresolved attributes — **done** (early-return with warning for `{...}` targets)

### Milestone 3: Advanced Validation — **Complete**
Delivered:

- SARIF output — **done** (SARIF 2.1.0 with full compliance)
- case mismatch detection — **done** (case-insensitive fallback with warning)
- optional external link checking — **done** (opt-in via `--external-links`, HEAD with GET fallback, bounded concurrency)
- include cycle detection — **done** (`internal/cycles/cycles.go`, emits `include-cycle` error)

### Milestone 4: Hardening — **Not Yet Started**
Not yet delivered:

- performance improvements
- broader fixture coverage
- better Windows path handling
- polished documentation and examples

---

## 19. Open Questions

Resolved during implementation:

1. **How much attribute resolution should v1 support?** — **Resolved:** None. Targets containing `{...}` emit a warning and are skipped (no attribute expansion).
2. Should nav files be explicitly validated in v1? — **Still open.**
3. **Should anchors within pages be validated in v1 or deferred?** — **Resolved:** Deferred. Fragments are extracted from xrefs but not validated against page content.
4. Should external link checking use `HEAD`, `GET`, or fallback logic? — **Resolved:** HEAD request first, GET fallback on HTTP 405 Method Not Allowed. Bounded concurrency (default 5) with configurable timeout (default 10s).
5. **How should the tool treat unresolved dynamic targets: warning, skip, or error?** — **Resolved:** Warning. The `unresolved-attribute` rule emits `SeverityWarning`.
6. **How aggressively should literal blocks and comments be ignored?** — **Resolved:** Comments (`//`) are skipped; code/literal blocks (`----`, `....`, backtick fences) are tracked. `xref` and `image` are skipped inside blocks, but `include::` is always scanned (AsciiDoc processes includes first).
7. **Should the tool require an Antora root, or support partial repo scans?** — **Resolved:** The tool scans from a given root directory and discovers all `antora.yml` files beneath it. It does not require a single Antora root.

---

## 20. Recommended v1 Decisions — **All Adopted**

The recommended product decisions for v1 have all been implemented:

- support only **high-confidence checks** — **adopted** (9 rules with clear semantics)
- prefer **warning** over false-positive **error** for unresolved attribute-heavy cases — **adopted** (`unresolved-attribute` is WARNING)
- keep external link checks **off by default** — **adopted** (implemented as opt-in via `--external-links` flag)
- ship **text + JSON** first — **adopted** (both implemented)
- add **SARIF** in the next increment — **adopted** (SARIF 2.1.0 also implemented ahead of schedule)
- build a **clean resolver layer** early — **adopted** (isolated `internal/resolve` package with per-type resolvers)
- keep scanning logic simple and structured — **adopted** (line-oriented regex scanner with block tracking)

These decisions are consistent with the core v1 principle: **pragmatic, fast, and maintainable**.

---

## 21. Example User Stories

### Story 1
As a documentation contributor, I want to run a single command locally so that I can detect broken references before I commit changes.

### Story 2
As a CI engineer, I want the tool to fail the build when broken docs references are introduced so that bad documentation does not get merged.

### Story 3
As a technical writer, I want file and line diagnostics so that I can fix broken references quickly.

### Story 4
As a platform engineer, I want JSON or SARIF output so that I can integrate results into automated tooling.

---

## 22. Acceptance Criteria — **MVP Substantially Met**

The MVP acceptance criteria and their status:

- the tool can scan a valid Antora repo root — **met**
- the tool indexes pages, partials, and images — **met** (also indexes examples and attachments)
- the tool detects broken `xref`, `include`, and `image` references — **met**
- every issue includes file path and line number — **met** (also includes column)
- the tool exits non-zero when configured failure conditions are met — **met**
- the tool runs successfully on Linux, macOS, and Windows — **not yet verified** (cross-platform testing pending)
- the tool can be distributed as a single Go binary — **met** (`go build -o adoclint ./cmd/adoclint/main.go`)
- unit and fixture tests cover the main resolution flows — **met** (14+ tests, 5 fixture repos, 6 golden files)

---

## 23. Example Output

### Text Output
```text
ERROR broken-xref docs/modules/user-guide/pages/index.adoc:42 xref target not found: admin:settings.adoc
ERROR broken-include docs/modules/common/pages/setup.adoc:18 include target not found: ../partials/missing-snippet.adoc
WARN  unresolved-attribute docs/modules/user-guide/pages/intro.adoc:10 target contains unresolved attribute: xref:{product-page}[Product Page]
```

### JSON Output
```json
[
  {
    "severity": "error",
    "ruleId": "broken-xref",
    "file": "docs/modules/user-guide/pages/index.adoc",
    "line": 42,
    "message": "xref target not found",
    "target": "admin:settings.adoc"
  }
]
```

---

## 24. Summary

`adoclint` is a Go-based CLI tool for validating Antora/AsciiDoc repositories by detecting broken internal references and related documentation issues. The first version focuses on the highest-value checks, delivers reliable diagnostics, and fits cleanly into both developer workflows and CI pipelines.

The defining product decision for v1 is to keep the implementation **pragmatic, fast, and maintainable** rather than trying to fully parse or emulate all AsciiDoc and Antora behaviour up front. That is the most practical path to delivering value quickly while preserving a clean foundation for future growth.

---

## 25. Implementation Status

### 25.1 Overall Progress

| Area | Status |
|------|--------|
| Repository discovery | Complete |
| Resource indexing | Complete |
| File scanning | Complete |
| Reference resolution | Complete (4 reference types) |
| Rule evaluation | Complete (9 rules) |
| Reporting | Complete (3 formats) |
| CLI | Complete (8 flags) |
| Unit tests | 14+ tests across 4 packages |
| Fixture repos | 5 (simple, broken, casemismatch, cycles, multicomponent) |

### 25.2 What Is Implemented

- **6-stage pipeline**: repo → index → scan → resolve → rules → report
- **Scanner**: Line-oriented regex scanner detecting `xref:`, `include::`, `image::`, `image:`, `link:{attachmentsdir}/...`, `link:https://...`, and bare `https://...` URLs with comment and code block awareness
- **Resolver**: Antora-aware resolution supporting same-module, cross-module, cross-component, and versioned references; relative paths; Antora family form (`$` prefix); case-insensitive fallback
- **9 rules**: `broken-xref`, `broken-include`, `broken-image`, `broken-attachment`, `case-mismatch`, `unresolved-attribute`, `include-cycle`, `external-link-dead`, `external-link-timeout`
- **3 output formats**: text, JSON, SARIF 2.1.0
- **CLI flags**: `--format`, `--fail-on`, `--verbose`, `--include`, `--exclude`, `--external-links`, `--timeout`, `--concurrency`
- **Exit codes**: 0 (clean), 1 (issues found), 2 (runtime error)

### 25.3 What Remains

| Feature | PRD Section | Priority |
|---------|-------------|----------|
| Fragment/anchor validation | 19.3 | Deferred |
| Cross-platform testing | 15.4 | Medium |
| Nav file validation | 19.2 | Open |
| Config file support | 7.8 | Future |
| Performance optimisation | Milestone 4 | Future |
