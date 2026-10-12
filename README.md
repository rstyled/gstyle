# gstyle

See cyclomatic paths and cognitive load in Go code in a TUI or hand it to an agent

## Install

Install `gstyle` globally with Go:

```sh
go install github.com/rstyled/gstyle@latest
```

The executable is `gstyle`. RPC methods use plain names such as `getState` and
`getReport`; standard discovery remains `rpc.discover`. Suppression comments use
`gstyle-allow`, and annotation storage and analysis profiles use the `gstyle` name.

## Run

Scan the current directory:

```sh
gstyle .
```

Pass one or more files or directories to scan them instead:

```sh
gstyle ./domain ./adapters
```

The treemap combines both complexity scores: tile area shows cyclomatic complexity,
and color shows cognitive complexity from green to red. Files group their function
tiles. Click a tile to inspect its source and both scores. The map appears in
terminals at least 100 columns wide and 30 rows tall.

Inside Git, Gstyle automatically shows added and deleted source lines against `main` or `master`.

Gstyle starts a localhost JSON-RPC control API on port `8197`. Pick another port with `--control-port`:

```sh
gstyle --control-port 9000 .
```

Ask the running app for its OpenRPC document:

```sh
curl -s http://127.0.0.1:9000 \
  -H 'content-type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"rpc.discover"}'
```

The control API reads and changes the live TUI model. `getReport` returns
complexity scores and a typed `quality` report with deterministic diagnostics and
effect counts. Each analyzed function includes `quality` metrics, diagnostics,
effects, and mutation events. `getState` includes this evidence alongside
the selected function's source. Agents can focus panes, select files and functions,
scroll source, refresh analysis, and wait for a new revision without polling.
See [application/openrpc.json](application/openrpc.json) for the full contract.

Print the operating skill for coding agents:

```sh
gstyle --skill
```

## Build

```sh
go build -o gstyle .
./gstyle
```

## Check Go code quality

Run typed mutation and side-effect analysis without starting the TUI:

```sh
gstyle check .
gstyle check --format json ./domain ./adapters
gstyle check --config examples/quality/gstyle.toml .
```

Directories include Go packages recursively. File arguments load their enclosing
packages for type information, then report only the selected files. Run from the
repository root; inputs must be inside that root. Analysis respects Go's active
build configuration. Use `--tests` to include tests and `--tags` for build tags.
Generated files are excluded. Package/type errors fail the check.

The TUI also runs quality analysis. Its header shows `Q` with the report's finding
count; the selected source title shows its function's count. Press `e` to switch
the details pane between source and quality evidence, then `j/k` to scroll.
Switching back preserves the source cursor. Use a policy in the TUI with:

```sh
gstyle --config examples/quality/gstyle.toml .
```

The TUI includes tests to match its existing complexity scan. Generated functions,
inactive build files, and nested literals that lack separate typed records show
quality as unavailable. Type-check failures appear as `quality failed`, with the
error in the quality pane; the syntax-based complexity view remains usable.

RPC exposes the same data through `getReport.quality` and
`getState.selection.function.quality`. Report quality has `status` (`ready`
or `error`), an optional error, and the successful report. Nested quality fields
use the same snake_case names as `gstyle check --format json`. Switch the shared
details pane with `setDetailsView {"view":"quality"}` or `{"view":"source"}`.
`getState.report.quality` contains status, aggregate statistics, and a
finding count; the full report comes from `getReport`.
State includes `detailsView` and `qualityOffset`. `revealLines` and
`scrollSource` return to source view. Refresh reruns both analyses.

| Rule | Default |
| --- | --- |
| `fn_length` | More than 200 code lines, excluding comments and blank lines |
| `fn_params` | More than 4 parameters, excluding the receiver |
| `mutation_per_target` | More than 3 writes to a target |
| `mutated_targets` | More than 3 distinct targets |
| `side_effect_density` | More than 500 milli, with at least 3 statements |
| `invalid_suppression` | Malformed suppression, unknown rule, or missing reason |

Mutation includes assignments to existing bindings, increments, field/index/
pointer writes, and `append`, `copy`, `delete`, and `clear`. Initial bindings do
not count. Shadowed variables have separate identities. Global reads/writes are
global effects instead of budgeted mutations. Nested closure bodies contribute
to their enclosing function even if not invoked; package-level function literals
get their own records.

Mutation provenance is `local`, `external`, or `unknown`. Owned local writes
count toward mutation budgets but do not add side effects. Writes through shared
parameters and their aliases add mutation effects. Conflicting alias assignments,
reference fields with unresolved ownership, and unresolved calls remain unknown.
Pointer-receiver calls are not automatically treated as writes.

Effects use resolved import/type/method paths and the longest matching config
prefix. `os/exec.Cmd.Output` is IO; `os/exec.Command` is a builder. Unclassified
calls add unknown effects unless a same-package helper body establishes their
effects. Local body summaries propagate effects; there is no cross-package effect
inference. Goroutines and channel operations also report unknown effects. Zero
density means no effects were detected under the policy, not proof of purity.
`complete` means no unknown effects remain under the policy; it is not a
whole-program correctness claim.

Density is `1000 * sum(effect weights) / max(statements, 1)` with integer division.
Weights default to mutation 1, IO/network 3, global 2, unsafe 4, time/random 1,
panic 0, unknown 1. Go statements count recursively; blocks, labels, and case
clauses do not add extra statements. Pure arithmetic scores zero.

TOML config retains unspecified defaults and rejects unknown keys. Project
prefixes extend defaults; later entries win equal-length ties. Kind `none`
explicitly excludes a call from scoring. See [the example policy](examples/quality/gstyle.toml).

Suppress selected rules with a reason:

```go
// gstyle-allow(fn_params): stable public interface
func Export(a, b, c, d, e int) {}
```

Doc comment lines may appear between the suppression and function. Invalid
suppressions never suppress findings.

Save facts and change policy without reloading or type-checking the code:

```sh
gstyle check --format facts . > /tmp/gstyle-facts.json
gstyle check --facts-in /tmp/gstyle-facts.json --config examples/quality/gstyle.toml --format json
```

Reports contain relative paths and no timestamps. Exit codes are **0** for no
findings or facts export, **1** for findings, and **2** for configuration, analysis,
or IO failure. Start with advisory reporting and review the source evidence before
using density as a CI gate.

Static calls to named helpers in the same package now use their body summaries.
Effect-free helpers contribute no effects; effectful helpers contribute evidence
at the caller line. Recursive cycles, dynamic dispatch, unavailable bodies, and
summary limits remain unknown. Explicit prefix rules still take precedence.
Parameter writes are propagated conservatively without argument ownership
remapping. Summaries stop at 32 active helpers or 4,096 summarized effects and
retain unknown coverage when a limit is reached.

Facts exports use `schema_version: 2` and retain a flat graph of helper bodies for
policy reevaluation, including helpers outside a selected file. Version 1 inputs
remain readable; missing helper evidence stays unknown. Report JSON retains
schema version 1. Older Gstyle builds do not accept the new facts format.

Run the [generated quality harness](examples/quality-hunt/README.md) with
`go run ./examples/quality-hunt` to check ownership behavior and equivalent
syntax across a deterministic corpus. Mismatches can be automatically reduced
with Tree-sitter. The generated checks also run in normal CI.

## Reduce a bug reproducer

Try the dashboard with a bundled input and checker from the repository root:

```sh
sh examples/bug-reducer/demo.sh
```

Each demo run saves to a fresh temporary directory, so the command is repeatable.
See [the demo guide](examples/bug-reducer/README.md) for details.

`bug-reducer` works with inputs for any codebase. Supply a command that confirms
one specific bug; the checker decides what counts as a bug. Default reduction
removes whole lines. Go mode uses Tree-sitter to remove complete declarations
and statements (including adjacent groups), reparsing each candidate before invoking the checker.
The filenames below are placeholders: supply an existing input and checker.

```sh
gstyle bug-reducer failing-input -- ./check-bug.sh
# Run directly from this repository:
go run . bug-reducer failing-input -- ./check-bug.sh
```

For Go source, install the Tree-sitter CLI and configure a Go grammar, or supply
an existing Go parser dynamic library explicitly:

```sh
gstyle bug-reducer --language go --go-parser /path/to/go.so --tui=false failing.go -- ./check-bug.sh
```

Omit `--go-parser` when Tree-sitter resolves the configured `source.go` grammar.
The Go reducer tries larger syntax units first and reparses after each accepted
deletion. Syntax errors skip the checker; build errors and the specific bug still
need to be checked by your command. This mode has no line-deletion fallback.
Tree-sitter is a runtime dependency for Go reduction; Gstyle builds still support
`CGO_ENABLED=0`. See the [Tree-sitter CLI documentation](https://tree-sitter.github.io/tree-sitter/cli/parse.html).

The checker receives an absolute candidate file path as its **last argument**,
following any arguments you supply. It runs in the directory you launched Gstyle
from, inherits your environment, and receives no stdin. Its output appears in the
live dashboard; plain mode suppresses it.
Exit **0** means the same bug remains; an ordinary nonzero exit rejects the
candidate. The checker must reject unrelated syntax errors or other failures.
For project-dependent tests, the checker arranges its own build/test workspace
or overlay using that candidate path. Candidates are temporary single files,
not copies of the entire project.

The command saves to `failing-input.reduced`, preserves the original, and refuses
to overwrite an existing output. Flags go before the input:

```sh
gstyle bug-reducer --timeout 30s --output smaller.json failing.json -- ./check-bug.sh --strict
gstyle bug-reducer --help
```

In a terminal, the reducer opens a live dashboard inspired by Shrink Ray:

- **Statistics:** best size, bytes removed, elapsed time, checks, and current chunk size.
- **Size over time:** the size of the best accepted input throughout the run.
- **Recent reductions:** accepted deletions with the removed lines.
- **Checker output:** live stdout and stderr from the current check (last 16 KiB).

Use `tab` / `shift+tab` to select a pane, `j/k` or arrow keys to scroll, and
`enter` to expand the selected pane. Small terminals show one pane at a time.
Press `q` or Ctrl-C to stop, clean up the checker, and save the best accepted
input. Completed runs stay open for inspection; `q` closes the dashboard.
Redirected output stays plain. Use `--tui` to force the dashboard or
`--tui=false` for the original summary-only behavior:

```sh
gstyle bug-reducer --tui=false failing-input -- ./check-bug.sh
```

Input must be a regular file; FIFOs, devices, and directories are rejected before
reading. Symlinks to regular files are supported.

The timeout applies to each checker invocation. Timeout, Ctrl-C, Unix SIGTERM, checker signal,
or launch failure stops reduction; once the seed is accepted, the best accepted
candidate is saved before the command reports the error. If the seed is rejected,
no output file remains. On Unix, cancellation and normal checker completion clean up its process group,
including ordinary child processes; descendants that detach into another group are outside that
scope. Other platforms currently cancel only the direct checker process. Checkers
should wait for their own subprocesses and avoid leaving background jobs running.

Default reduction deletes chunks of whole lines until no single remaining line
can be removed while preserving the check. Go reduction tries declarations,
statements, and groups of adjacent units; it also handles statements sharing a
line. Neither mode promises a global minimum. Use deterministic checks. A
permissive line-mode check can accept an empty file.

For a quick non-Go smoke test (the marker stands in for a bug):

```sh
printf 'noise\nspecific failure\nmore noise\n' > /tmp/reducer-demo.txt
gstyle bug-reducer /tmp/reducer-demo.txt -- grep -Fq -- 'specific failure'
cat /tmp/reducer-demo.txt.reduced
# specific failure
```

Finding bugs still needs tests, fuzzing, or inspection. For Gstyle, a checker can
compare the analyzer's reports for equivalent Go forms, or assert a known wrong
score, source location, or panic. Keep each check focused on the specific failure;
use the reduced input as the regression test after fixing it.

To check a specific Go input for initializer inconsistencies and reduce failures:

```sh
go run ./examples/gstyle-hunt path/to/input.go
```

See [the Gstyle checker guide](examples/gstyle-hunt/README.md) for its supported inputs
and checks. The exploratory fixture generators have been removed.

## Match imported graphs against compat's strict profile

```sh
gstyle match-graphs --input domain/ccgraph/testdata/imported-graphs.json
go test ./domain/ccgraph -run TestCompat -count=1
```

The command accepts the separate projected-stage input format shown in that
example. It streams JSON Lines with exact pair-stage traces and a final summary.
Matching uses `compat.ccgraph.strict/0.1.0`, pinned to [compat revision 04d9aa0](https://github.com/rstyled/compat/tree/04d9aa0db8b1f8e4a3fcad16622e2a4e3bc3e53f).
The 39 concrete upstream stage cases run offline from a hash-checked fixture copy.
Configuration overrides use exact decimal strings, for example
`--config thresholds.json` with `{"final_threshold":"0.95"}`. An optional
`--timeout 30s` stops resource use with an incomplete-run diagnostic; it never
shortens WL depth or silently truncates pairs.

This is imported-graph matching, not certification of native Go extraction.
Native PDG exports require a separate CFG/source/evidence validator before strict
matching. The existing `patterns` matcher remains the Gstyle adaptation. Pair
results do not establish semantic equivalence or safe refactoring. Required
unknown facts are diagnosed; cancellation or output failure leaves an incomplete run.

## Export the compat PDG IR

```sh
gstyle patterns --format pdg-json . > pdg.json
```

Writes a JSON array with one compat draft 0.1.0 document per function. Export skips
pattern mining and preserves extraction limits as explicit evidence. The existing
`patterns --format json` option still writes the pattern report.
The schema lives in [rstyled/compat](https://github.com/rstyled/compat/blob/56edd6bb804b10edaf9bbc2343d69e5e298d8c1a/schemas/pdg.schema.json).
The root extraction profile governs counts, dependencies, capabilities and
provenance. Records have no separate policy fields; conformance is one assessment
of the root profile. Gstyle's native reference count still counts recorded
reference-bearing bindings; it is not the strict matcher's retained-read count.
To check the export fixture against a compat checkout, install Python `jsonschema`
and run:

```sh
COMPAT_PDG_SCHEMA=/path/to/compat/schemas/pdg.schema.json go test ./adapters/pdgjson -run TestExportMatchesCompatSchema -count=1
```

The schema test skips when no path is set. Tests and export do not fetch schemas.
Scope IDs use relative paths and lexical byte ranges; they stay deterministic for
unchanged source and extraction root. Dependencies remain approximate; structural
schema validity does not establish compat extraction or matching conformance.
