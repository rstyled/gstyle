# Change risk report

## Settled direction

- Add a deterministic change risk report in Go, using gstyle's existing PDG and checks.
- Support features, bug fixes and refactors.
- Read configured rules from `gstyle.toml`; do not require a task manifest.
- Report a risk level and concrete reasons tied to affected files and functions.
- Determine severity from configured kinds of changes. A PDG difference alone does not imply high risk.
- Use a compact human report similar to merge-risk: an overall risk, consistent categories, emoji verdicts and short reasons, totaling 10–20 lines.
- Distinguish unavailable evidence from a passing check in the compact report.
- Append an optional, collapsed Graph changes section after the compact report when meaningful graph changes are available.
- Emit Mermaid text from Go and use GitHub's native rendering. Do not add Pele or a JavaScript rendering dependency for PR descriptions or comments.
- Show a bounded view of changed nodes and edges with enough surrounding context, rather than the entire PDG. Label additions and removals explicitly.
- The report does not authorize or perform a merge.

An illustrative output shape is:

```text
Change risk: high

PDG effects: high ⚠️
New filesystem write: os.WriteFile in store.go:42.

Dependencies: unchanged ✅
No module or package-import changes.

Compatibility: unchanged ✅
No public API or CLI contract changes.

Checks: passed ✅
Build, tests, race, guards and quality checks pass.

Coverage: unknown 👀
Base-to-head coverage comparison is unavailable.
```

The rendered report uses Markdown emphasis for verdicts. This example establishes the output shape; its facts are illustrative. It does not establish that every PDG change is high risk.

## Comparison requirements

Source positions, graph-local references and other metadata must not be mistaken for semantic changes. Rule severities come from the base revision's policy. Unavailable evidence stays unknown; a measured high-risk reason takes precedence over unknown evidence.

## First implementation

`gstyle risk --base REF --head REF` writes Markdown; `--format json` writes schema version 1 evidence. Both inputs resolve to full commit SHAs. The command creates and removes detached temporary worktrees, reads the base's tracked policy, and rejects symlinked policy paths or checkouts modified during analysis. Source, configuration and dependency errors never become a passing report.

`risk.pdg_change` and `risk.dependency_change` assign low, medium or high severity. `risk.calls` entries assign severity to an exact direct compiler callee identity whose occurrence count increases. Changed functions with indirect or unresolved call targets carry explicit unknown evidence. These rules have no enabled defaults. This repository opts into medium PDG changes, high dependency changes, and high added calls to `os.WriteFile` and `os.RemoveAll`.

The existing native PDG is the input. A report projection retains operation categories, types, symbols, operators, literals, call identities, writes, reads and typed dependency edges. It excludes source positions, opaque IDs and closure identity offsets. The projection is deliberately incomplete: source-token changes that it cannot distinguish are reported as unknown, not equivalent behavior. Literal text resembling a closure identity is not normalized away.

Changed production Go files in the root module are analyzed with their enclosing packages. Changed test files and nested-module Go source are explicitly outside this PDG comparison. Lexical closure calls contribute to call counts, but no correspondence across closure boundaries is invented. Repeated indistinguishable operations also make correspondence ambiguous; those diagrams are omitted.

Dependency collection compares the module graph, resolved module identities and replacement paths, plus production and test package imports. It includes nested project modules, excludes `testdata` and `vendor` fixture modules, and uses the current Go build environment. It is not a platform/build-tag matrix. Missing dependency evidence is marked unavailable. Go loader changes to a snapshot cause the report to fail.

The Markdown summary is eighteen lines before the optional diagram section. Diagrams show at most three functions, eighteen nodes and twenty-four changed edges per function. They use generated node IDs and escaped labels. The JSON retains the complete measured deltas. Diagram limits are presentation limits, not evidence that omitted operations are safe.

API/CLI compatibility, CI status and runtime test coverage are not collected in this slice. Their categories and JSON `unmeasured` fields remain explicit. Consequently the overall risk is unknown unless a measured, configured high-risk reason takes precedence. Exit 0 means successful report generation, including high and unknown results; operational/configuration/analysis failures exit 2. There is no merge permission, GitHub comment writer or automatic merge task.

## Verification

Tests cover a committed file-write addition, a candidate that disables its own policy, dirty source exclusion, deterministic Markdown, source-position and opaque-ID changes, closure calls and moves, cross-file separation, literal identity text, repeated callees, malformed policy and dependency output, temporary worktree cleanup, cancellation, snapshot mutation and escaping symlinks. Renderer controls check Markdown escaping, graph bounds and empty/ambiguous diagram omission.

Coverage, compatibility and CI evidence can be added later without changing the native graph representation or fabricating unavailable results now.
