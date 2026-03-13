# Product Requirements Document: Antora/AsciiDoc Repository Linter

## 1. Overview

### 1.1 Product Name

**adoclint**  
Working name for a command-line tool written in **Go** that scans an **Antora-based AsciiDoc repository** for broken references and structural issues.

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

- recursive scanning of an Antora repository
- discovery of Antora components, versions, modules, and resource families
- parsing of `.adoc` files using a pragmatic scanner
- validation of:
  - `xref:`
  - `include::`
  - `image::`
  - attachment-style references where feasible
- optional validation of external HTTP/HTTPS links
- text and machine-readable output formats
- non-zero exit codes for CI failure conditions

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

### 7.1 Repository Discovery

The tool must:

- accept a root directory as input
- recursively scan the repository for Antora content
- detect `antora.yml` files
- identify components, versions, modules, and resource families
- index resources under standard Antora families such as:
  - `pages`
  - `partials`
  - `examples`
  - `images`
  - `attachments`

The tool should:

- handle mono-repos containing multiple Antora components
- normalize paths consistently across operating systems
- retain case-sensitive path information for validation

---

### 7.2 File Scanning

The tool must:

- scan `.adoc` files
- detect relevant macros and reference patterns
- capture file path, line number, and where practical column number
- avoid obvious false positives in comments and code/literal blocks where feasible

The tool should use a **structured scanner approach**, not just raw ad hoc regex matching across the whole file.

The scanner should be intentionally lightweight for v1:
- line-oriented where possible
- context-aware enough to reduce noise
- easy to extend with new reference extractors

---

### 7.3 Reference Validation

#### 7.3.1 `xref:` validation
The tool must validate `xref:` references for local Antora resources.

It must detect:

- missing target page
- invalid family or module references
- invalid component or version references where explicitly provided
- malformed xref syntax where supported by scanner rules

It should support:

- relative xrefs
- module-relative xrefs
- page references with fragments
- Antora-style resource ID forms commonly used in documentation repos

It may initially warn, rather than fail hard, on advanced unresolved cases involving complex attributes.

#### 7.3.2 `include::` validation
The tool must validate `include::` targets.

It must detect:

- missing include file
- invalid relative path
- invalid partial or example target where identifiable
- include cycles if cycle detection is implemented in v1

It should support:

- standard include targets
- Antora-family includes where used conventionally
- reporting of include chains for easier debugging

#### 7.3.3 `image::` validation
The tool must validate image references.

It must detect:

- missing image files
- invalid relative image paths
- case mismatch between referenced path and actual file path

#### 7.3.4 Attachment reference validation
Where Antora attachment references are detectable, the tool should validate:

- missing attachment files
- invalid attachment paths
- case mismatch

---

### 7.4 External Link Validation

This feature is optional and disabled by default.

When enabled, the tool should:

- validate HTTP and HTTPS links
- use bounded concurrency
- apply request timeouts
- follow a limited number of redirects
- record status code or failure reason

The tool must avoid making the entire scan unreliable due to flaky external sites.

The tool should:

- treat transient network failures differently from definite dead links
- support retry logic with conservative defaults
- allow users to skip specific domains in future versions

---

### 7.5 Diagnostics

The tool must produce diagnostics containing:

- severity
- rule ID
- message
- file path
- line number
- optional column number
- reference type
- target value
- optional resolution context

Example categories:

- `broken-xref`
- `broken-include`
- `broken-image`
- `broken-attachment`
- `external-link-dead`
- `external-link-timeout`
- `unresolved-attribute`
- `case-mismatch`

The tool should group and summarize diagnostics at the end of a run.

---

### 7.6 Output Formats

The tool must support:

- human-readable text output
- JSON output

The tool should support:

- SARIF output for CI and code scanning integrations

Text output should be concise and useful in terminal workflows.

JSON and SARIF output should preserve structured fields for downstream automation.

---

### 7.7 Exit Codes

The tool must provide CI-friendly exit codes.

Suggested behavior:

- `0` = no issues found
- `1` = issues found above configured threshold
- `2` = runtime/configuration error

The exact mapping may be finalized during implementation, but it must remain stable once published.

---

### 7.8 Configuration

The v1 tool should support basic configuration through command-line flags.

Later versions may add config file support.

Initial configuration options should include:

- root directory to scan
- output format
- enable or disable external link checking
- severity threshold for failure
- include or exclude path patterns
- maximum concurrency for external link checks
- request timeout for external link checks

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

### 9.1 Command Structure

Example command style:

```bash
adoclint scan ./docs
adoclint scan ./docs --format json
adoclint scan ./docs --format sarif
adoclint scan ./docs --external-links
adoclint scan ./docs --fail-on warning
```

### 9.2 Flags

Proposed flags:

```text
--format <text|json|sarif>
--external-links
--fail-on <error|warning|none>
--include <glob>
--exclude <glob>
--timeout <duration>
--concurrency <n>
--verbose
```

The final flag names may vary, but the interface should remain minimal and predictable.

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

### 10.2 Suggested Package Layout

```text
cmd/adoclint
internal/repo
internal/index
internal/scan
internal/resolve
internal/rules
internal/report
internal/model
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

### 11.1 Core Domain Types

Suggested conceptual types:

#### Resource
Represents a discovered repo resource.

Fields may include:

- absolute path
- repo-relative path
- component
- version
- module
- family
- logical resource ID

#### Reference
Represents a parsed reference from an AsciiDoc file.

Fields may include:

- source file
- line
- column
- raw text
- reference type
- target
- context module/component/version
- fragment if present

#### Diagnostic
Represents a reported issue.

Fields may include:

- severity
- rule ID
- message
- file
- line
- column
- raw target
- suggested fix if available

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

### 13.1 Required Rules

#### R001 Broken xref
Report when an `xref:` target cannot be resolved to an indexed page.

#### R002 Broken include
Report when an `include::` target cannot be resolved to an existing file.

#### R003 Broken image
Report when an `image::` target cannot be resolved to an existing image.

#### R004 Broken attachment
Report when an attachment reference target cannot be resolved.

#### R005 Case mismatch
Report when a referenced path differs from the actual file path only by case.

#### R006 Unresolved attribute in target
Warn when a target contains attributes that were not resolved and validation therefore cannot be completed with confidence.

### 13.2 Optional Rules for v1 or v1.x

#### R007 Include cycle
Detect simple include cycles and report the include chain.

#### R008 Dead external link
Report confirmed dead external URLs.

#### R009 External link timeout
Warn when an external URL cannot be validated due to timeout or transient failure.

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

### 15.1 Unit Tests
The project must include unit tests for:

- scanner extraction logic
- path normalization
- xref resolution
- include resolution
- image resolution
- diagnostic generation

### 15.2 Golden Tests
The project should include golden-file tests for:

- text output
- JSON output
- SARIF output if implemented

### 15.3 Fixture Repositories
The project should include realistic fixture repos covering:

- valid Antora structures
- broken xrefs
- broken includes
- broken images
- unresolved attributes
- case mismatch scenarios
- multi-component and multi-version layouts

### 15.4 Cross-Platform Testing
The tool should be tested for path and case behavior across major OS environments, especially where Windows and Linux behavior differs.

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

### Milestone 1: MVP
Deliver:

- repo discovery
- indexing of Antora resources
- `.adoc` scanning
- broken `xref`, `include`, and `image` detection
- text output
- CI exit codes

### Milestone 2: Automation Readiness
Deliver:

- JSON output
- better diagnostics
- path include/exclude controls
- improved handling of unresolved attributes

### Milestone 3: Advanced Validation
Deliver:

- SARIF output
- optional external link checking
- case mismatch detection
- include cycle detection

### Milestone 4: Hardening
Deliver:

- performance improvements
- broader fixture coverage
- better Windows path handling
- polished documentation and examples

---

## 19. Open Questions

These should be resolved during design and implementation:

1. How much attribute resolution should v1 support?
2. Should nav files be explicitly validated in v1?
3. Should anchors within pages be validated in v1 or deferred?
4. Should external link checking use `HEAD`, `GET`, or fallback logic?
5. How should the tool treat unresolved dynamic targets: warning, skip, or error?
6. How aggressively should literal blocks and comments be ignored?
7. Should the tool require an Antora root, or support partial repo scans?

---

## 20. Recommended v1 Decisions

The recommended product decisions for v1 are:

- support only **high-confidence checks**
- prefer **warning** over false-positive **error** for unresolved attribute-heavy cases
- keep external link checks **off by default**
- ship **text + JSON** first
- add **SARIF** in the next increment
- build a **clean resolver layer** early, because that will decide whether the project remains maintainable
- keep scanning logic simple and structured rather than aiming for full parser completeness

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

## 22. Acceptance Criteria

The MVP is accepted when all of the following are true:

- the tool can scan a valid Antora repo root
- the tool indexes pages, partials, and images
- the tool detects broken `xref`, `include`, and `image` references
- every issue includes file path and line number
- the tool exits non-zero when configured failure conditions are met
- the tool runs successfully on Linux, macOS, and Windows
- the tool can be distributed as a single Go binary
- unit and fixture tests cover the main resolution flows

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

`adoclint` is a Go-based CLI tool for validating Antora/AsciiDoc repositories by detecting broken internal references and related documentation issues. The first version should focus on the highest-value checks, deliver reliable diagnostics, and fit cleanly into both developer workflows and CI pipelines.

The defining product decision for v1 is to keep the implementation **pragmatic, fast, and maintainable** rather than trying to fully parse or emulate all AsciiDoc and Antora behavior up front. That is the most practical path to delivering value quickly while preserving a clean foundation for future growth.
