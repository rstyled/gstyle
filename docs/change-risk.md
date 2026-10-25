# Change risk report

## Settled direction

- Add a deterministic change risk report in Go, using gstyle's existing PDG and checks.
- Support features, bug fixes and refactors.
- Read configured rules from `gstyle.toml`; do not require a task manifest.
- Report a risk level and concrete reasons tied to affected files and functions.
- Determine severity from configured kinds of changes. A PDG difference alone does not imply high risk.
- Use a compact human report similar to merge-risk: an overall risk, consistent categories, emoji verdicts and short reasons, totaling 10–20 lines.
- Separate measured change status from configured severity. Missing policy is unrated; real static analysis gaps are unknown.
- Show only implemented measurements as verdict rows. List uncollected evidence in one scope note.
- Append an optional, collapsed Graph changes section after the compact report when meaningful graph changes are available.
- Emit Mermaid text from Go and use GitHub's native rendering. Do not add Pele or a JavaScript rendering dependency for PR descriptions or comments.
- Show a bounded view of changed nodes and edges with enough surrounding context, rather than the entire PDG. Label additions and removals explicitly.
- The report does not authorize or perform a merge.

An illustrative output shape is:

```text
Change risk: high

PDG changes: changed 👀
New filesystem write: os.WriteFile in store.go:42.

Dependencies: unchanged ✅
No module or package-import changes.

API diff: unchanged ✅
No exported API changes in the root module (current build).

Analysis limits: None detected in the compared scope.

Not assessed: CLI contract, check results, runtime coverage.
Compared base → head.
```

The rendered report uses Markdown emphasis for verdicts. This example establishes the output shape; its facts are illustrative. It does not establish that every PDG change is high risk.

## Comparison requirements

Source positions, graph-local references and other metadata must not be mistaken for semantic changes. Rule severities come from the base revision's policy. Missing severities remain unrated. Unavailable static evidence stays unknown; a measured high-risk reason takes precedence over unknown evidence. With no risk rules at all, the overall rating is unrated and analysis limits remain visible separately.

## First implementation

`gstyle risk --base REF --head REF` writes Markdown; `--format json` writes schema version 1 evidence. Both inputs resolve to full commit SHAs. The command creates and removes detached temporary worktrees, reads the base's tracked policy, and rejects symlinked policy paths or checkouts modified during analysis. Source, configuration and dependency errors never become a passing report.

`risk.pdg_change`, `risk.dependency_change` and `risk.api_breaking` assign low, medium or high severity. `risk.calls` entries assign severity to an exact direct compiler callee identity whose occurrence count increases. Changed functions with indirect or unresolved call targets carry explicit unknown evidence. These rules have no enabled defaults. This repository opts into medium PDG changes, high dependency and breaking API changes, and high added calls to `os.WriteFile` and `os.RemoveAll`.

The existing native PDG is the input. A report projection retains operation categories, types, symbols, operators, literals, call identities, writes, reads and typed dependency edges. It excludes source positions, opaque IDs and closure identity offsets. The projection is deliberately incomplete: source-token changes that it cannot distinguish are reported as unknown, not equivalent behavior. Literal text resembling a closure identity is not normalized away.

Changed production Go files in the root module are analyzed with their enclosing packages. Changed test files and nested-module Go source are explicitly outside this PDG comparison. Lexical closure calls contribute to call counts, but no correspondence across closure boundaries is invented. Repeated indistinguishable operations also make correspondence ambiguous; those diagrams are omitted.

Dependency collection compares the module graph, resolved module identities and replacement paths, plus production and test package imports. It includes nested project modules, excludes `testdata` and `vendor` fixture modules, and uses the current Go build environment. It is not a platform/build-tag matrix. Missing dependency evidence is marked unavailable. Go loader changes to a snapshot cause the report to fail.

The Go API comparison reuses the existing package loading path with compiler types and readonly module resolution. It inventories all importable root-module packages, including internal packages, rather than only changed function PDGs. Command packages, tests and nested modules are outside this API scope. It compares packages by import path using a pinned `golang.org/x/exp/apidiff` version compatible with the existing compiler libraries. Package additions, removals and declaration-name changes are explicit. Changes to exported functions, methods, struct fields, interfaces, aliases, variables and constants follow apidiff's source compatibility rules. Its documented exclusions, including unkeyed struct literal compatibility, remain exclusions. Successful type checking of both revisions does not establish compatibility for old callers.

Changed generic parameter constraints or constraint interface terms are flagged for review separately: the comparator does not establish their source compatibility. The supplemental check expands embedded constraint interfaces, including private constraints exposed by exported declarations. Such evidence is partial, not a compatibility pass. Renaming generic parameters alone is ignored when constraint fingerprints remain equal. Unsupported correspondences can conservatively require review; the check is not a complete generic compatibility proof. Known breaking changes still remain breaking when other limits exist. API type loading failures are unavailable and never become an unchanged verdict. JSON retains all API changes, limits, package counts and the `root_module_current_build` scope. CLI string flags and runtime behavior require separate contract checks.

The Markdown summary stays within twenty lines before the optional diagram section. PDG and dependency rows show changed, unchanged, partial or unavailable status independently of severity. Dependency counts separate import edges from module records, and JSON retains the added and removed records. Analysis limits identify unresolved targets, excluded scope, unrepresented source changes and omitted diagrams. Ambiguous operation correspondence affects the illustration, not a known static change classification. Diagrams show at most three functions, eighteen nodes and twenty-four changed edges per function. They use generated node IDs and escaped labels. The JSON retains the complete measured deltas. Diagram limits are presentation limits, not evidence that omitted operations are safe. Markdown abbreviates commit IDs to twelve characters; JSON retains full SHAs.

CLI contracts, check results and runtime test coverage are not collected. One scope note and JSON `unmeasured` fields list them; no placeholder verdict rows are emitted. These unimplemented collectors do not force a static change rating to unknown. A low or medium static rating covers only the measured projection and is not an overall merge-risk assessment. Exit 0 means successful report generation, including high, unknown and unrated results; operational/configuration/analysis failures exit 2. There is no merge permission, GitHub comment writer or automatic merge task.

## Verification

Tests cover a committed file-write addition, a candidate that disables its own policy, dirty source exclusion, deterministic Markdown, source-position and opaque-ID changes, closure calls and moves, cross-file separation, literal identity text, repeated callees, malformed policy and dependency output, temporary worktree cleanup, cancellation, snapshot mutation and escaping symlinks. Renderer controls check Markdown escaping, graph bounds and empty/ambiguous diagram omission.

API controls include type-only changes without any function PDG, changed signatures with updated in-repository callers, compatible additions, unchanged bodies and parameter names, methods, fields, interfaces, constants, aliases, whole-package removal, default import-name changes, generic constraints and unavailable type inventories. An end-to-end Git control proves that a candidate cannot disable its base's breaking API policy. CLI, coverage and CI evidence can be added later without changing the native graph representation or fabricating unavailable results now.
