package application

import (
	"errors"
	"io"
)

const agentSkill = `---
name: gstyle
description: Search Go functions and quality findings, inspect complexity, check mutation and effect guardrails, and minimize bug reproducers.
---

# Gstyle

Install with ` + "`go install github.com/rstyled/gstyle@latest`" + `.
From the repository root, build with ` + "`go build -o gstyle .`" + `.

Use Gstyle to locate Go code whose control flow deserves inspection. Treat complexity as a signal, not a score to game.

## Enforce repository policy

Run ` + "`gstyle guard --config gstyle.toml --base REF .`" + ` to enforce configured file and Go-comment rules. Both rule types are disabled by default; a repository opts in through its TOML. The default config path is the repository's gstyle.toml. Quality rules remain under gstyle check.

The guard.comments mode added rejects new or expanded Go comment tokens relative to the Git base, while preserving unchanged comments. Mode all rejects every Go comment. Strings containing comment-like text are not comments. Directives, tests and generated Go files are included. guard.files entries match extensions case-insensitively or exact case-sensitive basenames through names. allow_directories and allow_paths exempt repository-relative directories and exact paths. File rules inspect tracked and nonignored untracked filenames without loading Go packages. These are filename checks, not interpreter or language detection.

File rules apply to every current path unless mode is added. Added rules check paths absent from the Git base; renames to new paths count as additions, and an unborn repository treats all paths as new. This repo enables nested_readme with names containing README.md, mode added and allow_paths containing README.md. It rejects new nested README.md files while allowing the root README and edits to existing nested READMEs. The rule is disabled unless configured.

Guard supports text, json and sarif. Exit 0 means no findings, 1 means findings and 2 means an operational/configuration failure. Guard findings are strict and do not use the quality baseline tolerance. These rules apply to all changes, independent of author identity.

## Report merge risk

Run ` + "`gstyle risk --base REF --head HEAD`" + ` for a compact Markdown report, or add --format json for structured evidence. This report applies to features, fixes and refactors. It reads the repository-relative --config path (default gstyle.toml) from the base commit, compares committed snapshots in temporary worktrees and excludes dirty files. It never merges or posts a comment.

Risk rules are off by default. Under [risk], pdg_change, dependency_change and api_breaking accept low, medium or high; [[risk.calls]] assigns a level to an exact direct callee identity. A configured added-call rule measures counts in native PDGs. Indirect call targets remain unknown. The source projection ignores positions and opaque IDs. Changed source outside that projection remains unknown. Repeated operations and lexical closures do not receive an invented node correspondence; their diagrams are omitted.

Markdown contains an optional collapsed Graph changes section with native Mermaid, at most three functions, eighteen nodes and twenty-four edges per function. Plus and minus labels identify additions and removals. The command compares changed production Go files in the root module; changed tests and nested-module source are explicitly unmeasured. Dependency evidence covers project modules, excluding testdata/vendor fixtures, and includes replacements and package imports.

The API diff row compares exported Go declarations in all importable root-module packages through the existing typed loader and golang.org/x/exp/apidiff. This includes internal packages, exported types/constants and APIs with no function PDG. Command packages, tests and nested modules are outside that scope. Package removals and declaration-name changes are detected. Changes to generic constraints require review; type loading failures are unavailable. The comparison follows apidiff source compatibility rules in the current build environment, not a platform matrix or a proof of runtime behavior. Entries show declarations and types or signatures; a recognized GitHub origin enables source links pinned to the compared commits. Dependency entries pair selected module versions and keep import counts separate. Analysis details retains full lists and limits. JSON preserves API changes, source locations, limits and scope.

Change status and severity are separate: measured changes remain visible without a severity policy. Missing policy is unrated; real static analysis gaps are unknown. Ambiguous matching omits a diagram without overriding a known static change rating. JSON retains added and removed dependency records. CLI contracts, check results and runtime coverage are unmeasured and appear in one scope note, not placeholder verdicts. The rating covers static evidence only. Exit 0 means report generated, including high/unknown/unrated risk; exit 2 means an operational/configuration/analysis failure. Never treat success or an unchanged projection as merge approval.

## Inspect

Run ` + "`gstyle .`" + ` for the current repository or ` + "`gstyle [paths...]`" + ` for selected Go files and directories.

The treemap combines both scores: area shows cyclomatic complexity, and color shows cognitive complexity from green to red. Files contain function tiles. Click a tile to select its source; the white outline marks the selected function. The map needs at least 100 columns and 30 rows.

- Files shows cyclomatic aggregates and a purple cognitive peak as space allows.
- Functions shows cyclomatic complexity as ` + "`CC`" + `, purple cognitive complexity as ` + "`COG`" + `, and physical size as ` + "`LINES`" + `.
- Source shows the selected function. Cyclomatic source uses amber text. Cognitive source gets a dark purple background. Shared lines show amber on purple. Line numbers stay neutral. Red connectors enclose guards that return errors. Red return values are errors. Green return values are successful results; a trailing nil error remains neutral.

The header shows ` + "`Q`" + ` with the quality finding count. Press ` + "`e`" + ` to switch the details pane between source and quality evidence, then j/k to scroll. Quality shows density, unknown coverage, rule findings, effects, and mutation events with source lines. Switching back preserves the source cursor. Use ` + "`gstyle --config PATH [paths...]`" + ` to apply the same TOML policy as the headless check. The TUI includes tests. Typed failures stay visible as quality failed; complexity inspection still works.

Inside a Git worktree, Source automatically shows the diff against ` + "`main`" + ` or ` + "`master`" + `. It falls back to their ` + "`origin/*`" + ` refs, then ` + "`HEAD`" + `. The Source title shows the chosen base and change counts. Green ` + "`+`" + ` gutters mark added lines. Red ` + "`−`" + ` rows preserve deleted lines beside the current function. Outside Git, Source stays unchanged.

Use ` + "`tab`" + ` and ` + "`shift+tab`" + ` to change panes, ` + "`j/k`" + ` to move, ` + "`,`" + ` and ` + "`.`" + ` to change files from any pane, ` + "`r`" + ` to refresh, and ` + "`q`" + ` to quit.

Press ` + "`[`" + ` and ` + "`]`" + ` from any pane to visit the previous or next note across the report. Navigation includes all saved notes and wraps at either end. The footer shows your position. Matched notes reveal their current source. Unmatched notes show their saved text in Source; ` + "`j/k`" + ` scrolls it and ` + "`d`" + ` removes the note. Files and Functions mark rows containing notes with amber diamonds.

In Source, ` + "`j/k`" + ` moves a line cursor and keeps it visible. Press ` + "`v`" + ` to start or clear a visual line selection, then ` + "`a`" + ` to attach a note. Use ` + "`d`" + ` to remove the note under the cursor. ` + "`esc`" + ` clears the line selection.

Annotations persist across restarts in SQLite at ` + "`$XDG_STATE_HOME/gstyle/annotations.db`" + `, or ` + "`~/.local/state/gstyle/annotations.db`" + ` when ` + "`XDG_STATE_HOME`" + ` is unset. They are isolated by repository.

## Control with JSON-RPC

Treat a running Gstyle TUI as a shared screen. Preserve unrelated focus and selections, and leave the view useful to the user.

Gstyle exposes its control API on ` + "`http://127.0.0.1:8197`" + ` by default. Probe that endpoint before starting another Gstyle process. A custom instance uses ` + "`gstyle --control-port PORT [paths...]`" + `. Pass ` + "`0`" + ` to choose a free port; Gstyle keeps the active port visible in its header.

Connect in this order:

1. Call ` + "`rpc.discover`" + `. Its runtime OpenRPC document is the authority for methods and parameters.
2. Call ` + "`getState`" + `. Record the revision, focus, and selected file and function before changing anything.
3. Inspect first. Mutate the shared view only when it helps the user.

Use any JSON-RPC client. This shell helper is enough:

` + "```bash" + `
CONTROL_URL=http://127.0.0.1:8197
rpc() {
  local method="$1" params="${2-}"
  if [ -z "$params" ]; then params='{}'; fi
  curl -fsS "$CONTROL_URL" \
    -H 'content-type: application/json' \
    --data "$(printf '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"%s\",\"params\":%s}' "$method" "$params")"
}

rpc rpc.discover
rpc getState
rpc setFocus '{"pane":"functions"}'
` + "```" + `

Treat a JSON-RPC ` + "`error`" + ` envelope as failure even when HTTP returns 200.

` + "`getReport`" + ` returns ranked complexity metadata. Each function includes ` + "`complexity`" + `, ` + "`cognitiveComplexity`" + `, ` + "`cyclomaticDiagnostics`" + `, and ` + "`cognitiveDiagnostics`" + `. The diagnostics identify each increment, its nesting cost, and its source position. The selected code lives at ` + "`result.selection.function.source`" + ` from ` + "`getState`" + `. Selections use zero-based indexes from the ranked file and function arrays. ` + "`setFocus`" + `, ` + "`selectFile`" + `, ` + "`selectFunction`" + `, and ` + "`scrollSource`" + ` return the updated state. ` + "`refresh`" + ` waits for analysis to finish before it replies.

Source review state is shared too. ` + "`revealLines`" + ` focuses and highlights a range, while ` + "`clearLineSelection`" + ` clears it. ` + "`annotateLines`" + ` adds a note without changing the visible selection; ` + "`removeAnnotation`" + ` deletes one by ID. Line parameters accept displayed source numbers or one-based lines relative to the selected function. Read ` + "`cursor`" + `, ` + "`lineSelection`" + `, ` + "`annotations`" + `, and ` + "`activeAnnotationId`" + ` from ` + "`getState`" + `. Use the runtime OpenRPC document for exact result shapes.

Use ` + "`waitForChange`" + ` instead of polling. Pass the last ` + "`revision`" + ` as ` + "`afterRevision`" + ` and an optional ` + "`timeoutMs`" + ` from 0 to 60000. It returns when the TUI advances or the timeout expires; compare the returned revision to tell which happened.

For inspection-only work, report the evidence without changing the TUI. Restore temporary focus or selection changes when they no longer help the user.

## Search before reading full reports

Run ` + "`" + `gstyle search --query "save" --path application --limit 20 .` + "`" + ` for compact JSON results without a TUI. Flags precede paths. Empty query browses functions. Use ` + "`" + `--findings-only` + "`" + `, ` + "`" + `--rule side_effect_density` + "`" + `, ` + "`" + `--min-complexity 4` + "`" + `, or ` + "`" + `--min-cognitive-complexity 8` + "`" + ` to narrow the result. Filters combine with AND. Query and path use case-insensitive literal substrings; rule uses an exact quality rule ID. Query searches path, package, function name, source, rule IDs, and finding messages. Pattern candidates stay in ` + "`" + `gstyle patterns` + "`" + `; suppressed quality findings are excluded.

Results contain matching functions once each, source locations, scores, compact findings, total, offset, limit, and nextOffset when more results exist. Use ` + "`" + `--offset` + "`" + ` with nextOffset to continue; limit defaults to 20 and accepts 1-200. No matches returns an empty matches array and exit 0. Analysis/config/IO failures exit 2. Inspect quality status and coverage: failed typed analysis still emits syntax results with quality.status=error, then exits 2. Unknown rule IDs match nothing. Full source is not repeated. Use ` + "`" + `--format text` + "`" + ` for readable output, ` + "`" + `--config PATH` + "`" + ` for policy, ` + "`" + `--tests=false` + "`" + ` to exclude tests from typed quality analysis, and ` + "`" + `--tags TAGS` + "`" + ` for typed build tags. The syntax scan includes test files and does not apply build tags, as in the TUI.

On a running TUI, call ` + "`" + `search` + "`" + ` with ` + "`" + `{"query":"save","findingsOnly":true,"limit":20}` + "`" + `. It uses the same filters in camelCase: minComplexity and minCognitiveComplexity. It returns revision, refreshing, error when present, and quality status with the results. Wait for refresh to finish before treating results as current. It preserves focus, selection, annotations, and revision. fileIndex and functionIndex address the ranked getReport arrays: call selectFile, then selectFunction to inspect a hit. Use indexes only while the revision is unchanged; restart pagination after a revision change. Read full quality evidence from getReport or the selected function.

## Pull requests

Scan only tracked Go files changed from ` + "`main`" + `:

` + "`git diff --name-only -z --diff-filter=ACMR main -- '*.go' | xargs -0 -r -o gstyle`" + `

The ` + "`-o`" + ` flag reconnects Gstyle to the terminal after ` + "`xargs`" + ` reads the pipe. Replace ` + "`main`" + ` with the pull request's base branch. This includes branch commits plus tracked staged and unstaged changes. Git omits untracked files, so pass those paths to Gstyle explicitly when they matter.

## Simplify

Use ` + "`CC`" + ` to find path-heavy functions and ` + "`COG`" + ` to find code that is hard to follow. A wide gap between them is useful evidence. Prefer guard clauses, early returns, and flatter control flow. Do not extract helpers solely to lower a metric or trade readable code for a smaller number.

After editing, run the relevant tests and Gstyle again. Report what became easier to follow, not only how the score changed.

Quality is shared too. ` + "`getReport`" + ` includes ` + "`quality.status`" + ` (ready or error) and, on success, ` + "`quality.report`" + ` with its summary, functions, and diagnostics. Function entries and ` + "`getState.selection.function.quality`" + ` include density_milli, complete, unclassified_calls, effects, mutation_events, and diagnostics. Nested quality fields retain the headless report's snake_case names. Missing function quality means unavailable; it is not a clean result.

Use ` + "`setDetailsView`" + ` with ` + "`{\"view\":\"quality\"}`" + ` or ` + "`{\"view\":\"source\"}`" + ` to change the shared details pane. State includes detailsView and qualityOffset. ` + "`revealLines`" + ` and ` + "`scrollSource`" + ` return to source view. ` + "`refresh`" + ` reruns complexity and typed quality analysis together.

## Fix patterns automatically

Run ` + "`gstyle patterns [paths...]`" + ` to find refactoring proposals and bug patterns: guard clauses, value objects, verified parameterizations, anemic models, primitive obsession, type switches, enum dispatch, entity identity, missing identity, factories, specifications, and domain services (detection-only). Supported fixes use deterministic source edits backed by extraction evidence. Identity and domain-design findings need review.

Run ` + "`gstyle fix --kind all [paths...]`" + ` to auto-fix them. Dry-run by default (shows a diff); add ` + "`--apply`" + ` to write. No LLM, no tokens — pure static analysis. Use ` + "`--kind guard_clause|value_object|parameterize|anemic_model|primitive_obsession|type_switch|enum_dispatch|trait_method|capability_set|generic_fn|factory|specification`" + ` to fix one kind.

Every fix run uses ordered phases. PDG facts are reused until edits change the source, then refreshed for the complete input scope (max 3 cycles). Preview simulates the same edits without writing. Phases run guard/value-object work first, structural patterns next, parameterize next, and declaration ordering last.

` + "`domain_service`" + ` is detection-only: it identifies stateless functions operating on 2+ domain types as legitimate Domain Services (Evans), and suppresses the corresponding ` + "`anemic_model`" + ` suggestions. ` + "`specification`" + ` extracts repeated boolean business rules into named predicates (a method on the type, or a function for external types). ` + "`trait_method`" + ` proposes interfaces from parallel methods on types in the same package.

The offset_fallback pattern detects guarded suffix slices that leave the
original collection intact when an offset exceeds its length. It shows an
empty-result diff for review. The missing-fallback case is advisory: fix --apply
never invents pagination intent. Automatic clamping requires a single slice
assignment and an existing else branch that selects the empty suffix, preserving
capacity and aliasing. Shadowed min, older Go versions, condition comments, and
extra body work block automatic edits. Re-mine after source changes.

The declaration_order pattern groups each type with its New/new constructors,
exported methods, then private methods, with other functions last. Constructors
must return a local named type as their first result. Use
` + "`gstyle fix --kind declaration_order [paths...]`" + ` to preview and add
` + "`--apply`" + ` to write. It preserves comments and bodies and skips files
with Go directives, cgo imports, or same-line declarations. Its candidate spans
the file, so ` + "`--changed`" + ` can reorder unchanged functions in that file.

Parameterization findings require compatible canonical PDG interfaces and callable types,
a full supported AST match, a source snapshot proof, and compatible PDG binding
identities. Auto-fixes substitute matched call occurrences without reordering the
body. Missing whole-function effect/order coverage alone does not block them.
Fresh names prevent capture; builtin substitutions and direct recover/stack
inspection do not produce findings. Graph similarity alone stays internal.
Size thresholds, signature/clone similarity and mined usage deviations do not produce findings.
Default fixing also skips unverified interface and generic proposals. Explicit
fix kinds remain available for manual review.
A proof targets only its selected pair; after source edits,
re-mine with phased fixing before parameterization.

**Workflow:** Run ` + "`gstyle patterns`" + ` first, then ` + "`gstyle fix --apply`" + ` to clear the mechanical issues. This saves tokens — don't hand-rewrite what the fixer handles. In an agent loop, use ` + "`gstyle fix --changed --apply`" + ` to fix only candidates in functions your diff touched (against ` + "`--base REF`" + `, defaulting to the merge-base with main/master).

**One at a time:** ` + "`gstyle next [paths...]`" + ` shows the single highest-impact candidate with fix instructions — the one thing to fix now. ` + "`gstyle score [paths...]`" + ` gives a 0-100 finding score (100 = no scored findings, weighted by severity; inspect dependency coverage). Suppressed items show separately, so you can't suppress your way to 100.

## Check quality guardrails (needs your judgment)

Run ` + "`gstyle check --format json [paths...]`" + ` for typed Go mutation and side-effect diagnostics without a TUI. Directories scan packages recursively; Go file arguments report only those files after loading their enclosing packages. Run from the repository root. Use ` + "`--config PATH`" + ` for TOML policy, ` + "`--tests`" + ` to include tests, and ` + "`--tags TAGS`" + ` for build tags. Use ` + "`--changed`" + ` to report only findings in functions the git diff touches (against ` + "`--base REF`" + `, defaulting to the merge-base with main/master), so an agent loop can gate on what its own edits introduced; untracked files count as fully changed.

The rules are fn_length, fn_params, mutation_per_target, mutated_targets, side_effect_density, invalid_suppression, and the DDD rules aggregate, repository, and mutable_identity. Findings carry actual, limit, rule_id, and source location. Density findings include effect kinds, labels, and lines; the message shows the finding's arithmetic (weight over statements with per-kind weights), and the diagnostic carries weight, statements, and kind_weights fields. The Go standard library is fully classified, so unknown now means a call gstyle cannot see into — third-party code or dynamic dispatch — rather than an ordinary stdlib call. Review that evidence before changing code. Preserve legitimate IO boundaries and do not extract helpers solely to lower a score.

Static calls to visible helpers use conservative body summaries. Density counts each source call once per effect kind; all underlying effects remain evidence, and helpers selected by the scan receive their own function reports. Saved facts without call identities retain independent effect counts. Proven read-only recursion and writes bound to caller-owned direct formal storage can be effect-free. Unproved cycles, dynamic dispatch, missing bodies, and summary limits remain unknown. Prefix policy overrides still apply, including when reevaluating saved facts. Intrinsic effects are syntactic; nested closure bodies contribute even if not called. Pointer-receiver calls are not automatically mutations. Zero density is not proof of purity.

Facts export uses schema_version 2 with helper summaries; version 1 remains readable and keeps absent helper information unknown. Save versioned facts with ` + "`gstyle check --format facts .`" + `, then use ` + "`gstyle check --facts-in PATH --config POLICY --format json`" + ` to reevaluate without loading Go packages. Exit 0 means no findings or successful export, 1 means guardrail findings, and 2 means an operational failure. Type errors must not be interpreted as a clean check.

Suppressions use ` + "`// gstyle-allow(rule_a, rule_b): reason`" + ` above a function, with intervening doc comments allowed. Reasons are required; unknown rules fail validation. Prefer documenting a deliberate exception over hiding evidence.

The advisory unchecked_output pattern flags standalone fmt.Fprint/Fprintf/Fprintln calls and assignments or declarations that discard their error with _. Typed stdlib identity is required; known *bytes.Buffer and *strings.Builder destinations are excluded. Best-effort diagnostics need review. This rule has no auto-fix and does not track indirect calls, deferred writes, or later assigned-error uses.

Bug-pattern warnings are advisory. ` + "`nilerr`" + ` requires explicit nil in every error result of the error branch; propagating an error in another result is valid. ` + "`typednil`" + ` proves non-nil interface comparisons from compiler value kinds and supported PDG binding/value/return dependencies. A concrete value has a dynamic type even when its pointer, map, or slice is nil. The bounded acyclic analysis follows local overwrites, branch joins, named returns, and direct same-analysis function summaries. Mixed nil paths and unknown interface inputs remain unknown. Loops, address escapes, closures, deferred writes, concurrency, and multiple results/assignments are excluded; absence of a warning does not prove nil safety. Interface suggestions describe normalized signature shapes, not Go declarations; preserve the concrete parameter and result types.

Taint and slicing use canonical PDG operations. Inspect analysis_coverage even
when no findings appear: partial coverage is not proof of clean flow. Each finding's
analysis_evidence records missing capabilities and canonical operation IDs within
its function. These are advisory paths and slices, not proof of a vulnerability.

Native conformance uses ` + "`gstyle match-graphs --source .`" + `.
It validates bounded scalar Go source and reports excluded functions before strict
matching. Add ` + "`--context-output /tmp/gstyle-context.jsonl`" + ` with a new path
to preserve source snapshots and paired PDG/CFG documents. The held-out quality
study remains pending; clone scores do not prove a safe auto-fix.

**Patterns vs quality:** ` + "`gstyle fix`" + ` handles mechanical patterns automatically. ` + "`gstyle check`" + ` findings (side_effect_density, mutated_targets, etc.) are design smells — they need your judgment to refactor. The check tells you *where* to look; you decide *how* to restructure. Don't try to auto-fix quality findings.

## Reduce a bug reproducer

Use the bug reducer once you have an input that reproduces a specific failure and a deterministic checker for that failure. Default mode works with inputs for any codebase and removes whole lines; it does not find or fix bugs itself. Go mode uses Tree-sitter to remove complete declarations, statements, and adjacent groups, largest first, reparsing each accepted reduction.

` + "```sh" + `
gstyle bug-reducer command.sh -- ./checker.sh
gstyle bug-reducer --timeout 30s --output reduced.sh command.sh -- ./checker.sh
gstyle bug-reducer --language go --go-parser /path/to/go.so --tui=false input.go -- ./checker.sh
` + "```" + `

Here, ` + "`command.sh`" + ` is the input file being minimized. ` + "`checker.sh`" + ` receives the absolute candidate file path as its last argument, after any checker arguments supplied on the command line. The reducer invokes the checker; the checker decides how to run or inspect the candidate. Input can also be Go source, JSON, or another file format.

Prefer --language go for Go reproducers. Use the default line mode for other formats or when deliberately experimenting with text reduction. Go mode requires the tree-sitter CLI and a configured source.go grammar, or an explicit Go parser dynamic library supplied with --go-parser. Omit --go-parser when the grammar is configured. If the parser is unavailable, report the setup error rather than silently switching modes.

Go syntax is validated before every candidate reaches the checker. The checker must still validate compilation and behavior; syntax alone does not establish the bug. Go mode has no line-deletion fallback. Adjacent groups allow related statements, such as a declaration and its only use, to disappear together. Structural reduction can use fewer checker runs while leaving a slightly larger reproducer than line reduction; compare both checker count and final size when evaluating it.

Write the checker to exit **0 only when the same bug still occurs**. An ordinary nonzero exit rejects the candidate. Reject unrelated syntax errors, build failures, or different crashes. Confirm that the original reproduces the failure before reducing it. For project-dependent checks, arrange the required build workspace or overlay in the checker.

The checker runs in the launch directory, inherits the environment, and receives no stdin. Terminals automatically show a live dashboard with statistics, size over time, accepted deletions, and checker output. Use tab to change panes, j/k to scroll, enter to expand, and q to stop and save the best accepted input. Completed runs stay open for inspection; q closes the dashboard. Use ` + "`--tui=false`" + ` for unattended runs, or ` + "`--tui`" + ` to force the dashboard. Redirected output uses the plain final summary and suppresses checker output. The dashboard retains the last 16 KiB of output per check.

Candidates are temporary single files. Input must be a regular file. The original stays intact; output defaults to ` + "`command.sh.reduced`" + ` and must not already exist. Flags go before the input; the default timeout is 10 seconds per check.

Timeout, Ctrl-C, Unix SIGTERM, checker signals, and launch errors stop reduction. After the original is accepted, the best accepted candidate is saved even when the run stops with an error. Unix runs clean up the checker's process group; detached descendants and child-process cleanup on other platforms need checker-managed cleanup.

Recheck the reduced file, use it as a regression test, and then fix the bug. Report the original and reduced sizes, the checker used, and validation results. The result is a local minimum for the selected deletion units, not a guarantee of the smallest possible reproducer. Use ` + "`gstyle bug-reducer --help`" + ` for the current CLI contract.
`

func WriteSkill(writer io.Writer) error {
	if writer == nil {
		return errors.New("skill writer is required")
	}

	_, err := io.WriteString(writer, agentSkill)
	return err
}
