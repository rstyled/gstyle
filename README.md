# gstyle

Find complex code, heavy mutation, and repeated patterns in Go. Inspect the results
in a terminal UI, get JSON reports, or hand the tool to a agent.

## Install

```sh
go install github.com/rstyled/gstyle@latest
```

## Start with your code

Run `gstyle --help` (or `gstyle help`) for commands and UI options.
Use `gstyle help check` or `gstyle check --help` for a command's flags.

Run these from your repository root:

```sh
gstyle .                    # explore complexity and quality in the TUI
gstyle check .              # report mutation and side-effect guardrails
gstyle patterns .           # find repeated code, refactoring candidates, and bug patterns
```

Pass files or directories to narrow the scan:

```sh
gstyle ./domain ./adapters
gstyle check ./domain/service.go
```

Quality and pattern analysis use Go type information. Fix package or type errors
before relying on their results. A finding is a reason to inspect the code, not
an instruction to rewrite it.

## Explore in the terminal

The treemap groups functions by file. Tile area shows cyclomatic complexity;
color shows cognitive complexity from green to red. Click a tile to inspect its
source and scores. The map appears at 100 columns by 30 rows or larger.

Inside Git, gstyle shows added and deleted source lines against `main` or `master`.
Press `e` to switch between source and quality evidence, and `j/k` to scroll the
quality pane. Type errors leave the syntax-based complexity view available.

Use your quality policy in the TUI:

```sh
gstyle --config examples/quality/gstyle.toml .
```

## Check quality

```sh
gstyle check --format json .
gstyle check --config examples/quality/gstyle.toml .
gstyle check --changed --base master .
```

`--changed` limits findings to functions touched by the Git diff. Directories scan
packages recursively. File arguments load the enclosing package, then report only
those files. Generated files are excluded. Use `--tests` to include tests and
`--tags` for build tags.

| Rule | Default limit |
| --- | --- |
| `fn_length` | 200 code lines, excluding comments and blank lines |
| `fn_params` | 4 parameters, excluding the receiver |
| `mutation_per_target` | 3 writes to one target |
| `mutated_targets` | 3 distinct mutation targets |
| `side_effect_density` | 500 milli, with at least 3 statements |
| `invalid_suppression` | Suppressions need a known rule and a reason |

Quality checks also report aggregate, repository, and mutable-identity concerns.
Findings include locations and evidence so you can decide whether the code needs
a change or has a valid reason for the tradeoff.

Local owned writes count toward mutation budgets. Writes through shared values,
IO, global access, and other configured effects contribute to side-effect density.
Unknown calls remain unknown; a low score does not establish purity.

See [the example TOML policy](examples/quality/gstyle.toml) for configuration.
Unspecified settings retain their defaults; unknown keys are rejected.

Give parts of the repository their own policy with `[[overrides]]`:

```toml
[[overrides]]
paths = ["adapters"]
weights = { io = 0, network = 0 }
side_effect_density = { max = 800 }
```

An override takes the same rule, weight, `min_statements`, `count_self` and
`granularity` keys as the top level. Keys it leaves out keep their top-level
values. `paths` lists repository-relative files or directories; a directory
covers everything below it. When several overrides match a file, they apply in
file order, each changing only the keys it sets:

```toml
[[overrides]]
paths = ["domain"]
weights = { io = 1000, global = 1000 }

[[overrides]]
paths = ["domain/legacy"]
fn_length = { max = 400 }   # domain/legacy keeps the domain weights
```

Overrides cannot set `guard`, `risk`, `prefixes` or `grouping`.

## Enforce repository policy

`gstyle guard` checks repository files without loading Go packages. Both rule
types are disabled by default. This repository enables them in `gstyle.toml`:

```toml
[guard.comments]
mode = "added"

[[guard.files]]
id = "python_outside_lean"
extensions = [".py", ".pyi", ".pyw", ".pyc", ".pyo", ".ipynb"]
allow_directories = ["verification/lean"]

[[guard.files]]
id = "nested_readme"
mode = "added"
names = ["README.md"]
allow_paths = ["README.md"]
```

```sh
gstyle guard --config gstyle.toml --base origin/master .
gstyle guard --format json .
make guard GUARD_BASE=origin/master
```

The default config path is the repository's `gstyle.toml`. `added` compares Go
comment tokens with the base commit, preserving unchanged comments in edited or
Git-recognized renamed files. New or expanded comments, including directives,
are findings. Comment-like text in strings is ignored. `all` reports every Go
comment. Tests, generated Go files and files in nested modules are included.

File rules check tracked and nonignored untracked paths. Extensions match
case-insensitively; `names` matches exact, case-sensitive basenames. Allowed
directories use repository-relative paths and exact directory boundaries;
`allow_paths` permits exact repository-relative paths. Either allowance exempts
the path. These are filename rules, not language detection or a scan for
interpreter commands. Deleted files are excluded. Neither rule depends on the
author's identity.

File rules normally apply to every current path. With `mode = "added"`, a rule
applies only to paths absent from the base commit, including staged, untracked
and branch-committed files. Renaming an existing file to a new path counts as
an addition. In a repository with no commits, all current paths are new. The
`nested_readme` rule above blocks new nested `README.md` files while allowing
the root `README.md` and edits to existing nested READMEs. Removing the rule
disables this restriction; it is not enabled by default.

The same TOML file can hold quality and repository rules. Quality rules run with
`gstyle check`; repository rules run with `gstyle guard`. Guard supports text,
JSON and SARIF. Exit 0 means no findings, 1 means policy findings, and 2 means an
operational or configuration failure. Guard findings are strict and do not use
the quality baseline's tolerance. CI runs `make guard` against the event's base
commit; local added-comment checks use the existing Git base selection unless
`--base` is supplied. `make guard` returns a nonzero status on any finding.

Suppress a quality rule with a reason when the tradeoff is intentional:

```go
// gstyle-allow(fn_params): stable public interface
func Export(a, b, c, d, e int) {}
```

`check` exits **0** with no findings, **1** with findings, and **2** on an analysis,
configuration, or IO error. Start with advisory reports before using it as a gate.

## Report merge risk

```sh
gstyle risk --base origin/master --head HEAD
gstyle risk --base origin/master --head HEAD --format json
gstyle risk --base origin/master --head HEAD --check
```

`risk` compares committed revisions in temporary detached checkouts. It emits a
compact Markdown report with emoji verdicts and, when the correspondence is
unambiguous, a collapsed Mermaid section for changed PDG operations and edges.
GitHub renders these diagrams directly in PR descriptions and comments.

Risk rules are disabled by default. A repository opts in through `gstyle.toml`:

```toml
[risk]
pdg_change = "medium"
pdg_low_risk_max_functions = 3
dependency_change = "high"
api_breaking = "high"

[[risk.calls]]
callee = "os.WriteFile"
level = "high"
```

The optional `pdg_low_risk_max_functions` lowers medium PDG-change severity to
low for changes within the function-count limit. Higher-risk calls, API and
dependency changes, and incomplete analysis still require review. Zero or
omission disables the limit.

The policy comes from the base revision, so editing the candidate's policy
cannot downgrade its own report. `--config` selects a tracked repository-relative
policy path in that base; symlinks are rejected. Both revisions must exist.
Uncommitted edits are excluded. The default head is `HEAD`; the default base is
its merge-base with `origin/main` or `origin/master`, then local `main` or
`master`, falling back to the head itself.

The report measures native PDG operation/dependency changes, exact configured
direct call counts, exported Go API changes, and project module requirements,
replacements and package imports. Position and opaque-ID changes are excluded; source changes
that the projection cannot distinguish remain unknown. Indirect call targets
remain unknown. Changed tests and nested
module source are explicitly outside the PDG comparison. Dependency collection
includes nested project modules and excludes `testdata` and `vendor` fixtures.

Uncovered changes require review: package-level declarations or comments, Go
files excluded from the current build, test or nested-module source, controls,
scripts, assets and unsafe file modes. Regular root README and documentation
Markdown are eligible only when they are not embedded program resources.

The API diff section reuses the typed package loader and applies
`golang.org/x/exp/apidiff` to every importable package in the root module,
including internal packages. It compares exported functions, types, methods,
fields, constants and variables, and detects package removal or renaming.
Body-only edits leave this API unchanged. `api_breaking` assigns severity to
detected source-incompatible changes; it has no enabled default. Changes to
generic constraints require review and remain partial unless another detected
break already establishes incompatibility. The check uses the current build
environment, excludes command packages and nested modules, and follows
apidiff's source compatibility rules; it does not prove behavior preservation.

API entries show the changed declaration and its type or signature. A GitHub
`origin` remote enables source links pinned to the compared commit, with old
declarations linked to the base. Added packages link to their directory and show
the number of exported declarations. Dependency entries pair old and new module
versions and show updated, added and removed module counts. Every module change
appears directly in the report. Every API change appears once. JSON retains import
links, analysis limits, source locations and unmeasured scopes.

Change status is separate from severity: measured PDG and dependency changes
remain visible when no severity policy is configured. `unrated` means policy
does not assign a severity; `unknown` means static analysis has a real gap.
Ambiguous operation matching omits a diagram without erasing a known change.
The JSON retains added and removed dependency records as well as PDG deltas.

The rating covers compared static evidence. CLI contracts, check results
and runtime test coverage are not collected; JSON lists these unmeasured fields.
Missing collectors do not produce placeholder verdicts or force the static rating to
unknown. This report does not authorize a merge or prove behavior preservation.
Without `--check`, exit 0 means the report was generated at any risk level.
With `--check`, only low risk exits 0; medium, high, unknown and unrated exit 1
and require human review. The report is still written. This mode requires an
explicit `--base`. Operational, configuration and analysis errors exit 2.
See [the comparison scope](docs/change-risk.md) for diagram limits and evidence.

## Find patterns and apply fixes

```sh
gstyle patterns .
gstyle patterns --format json . > patterns.json
gstyle fix .                         # preview supported rewrites
gstyle fix --kind declaration_order . # preview one kind
```

Pattern analysis uses program dependence graphs to find repeated regions and
track values and dependencies. It also reports opportunities such as guard
clauses, parameter groups, declaration ordering, and unchecked type assertions.
Discarded output errors are retained as analysis facts. They do not produce
`unchecked_output` findings: a failed write followed by a nil return does not
establish that output delivery is required. Status messages can fail after the
requested operation succeeds.

The `nilerr` rule checks bounded local paths from a non-nil error guard to an
empty result and a nil error return. It keeps boolean branch facts and compiler
field identities. Built-in print calls do not count as recovery. Standard-library
logging calls can invoke user callbacks and stay unknown. Deferred or concurrent code,
opaque calls, escaped storage, unsupported control flow, and exhausted path
budgets stay quiet. Generic zero values and boxed nil pointers do not establish
a nil error. This is not an interprocedural proof of correct error handling.

Size thresholds, signature/clone similarity and mined usage deviations do not produce
findings. Parameterization requires a complete supported source match with
compatible callable types; an aligned graph region alone is not enough.
Signature groups remain internal matching inputs. Default fixing also skips
unverified interface and generic proposals. Explicit fix kinds still preview
those transformations for manual review.

Coverage can be partial; no finding does not establish that a dependency or bug
is absent.

`fix` previews changes by default. Apply a specific kind after reviewing it:

```sh
gstyle fix --kind declaration_order --apply .
gstyle fix --changed --base master --apply .
```

Some candidates have a supported rewrite; others stay advisory. Review the diff
and run your project's checks after applying fixes. Signature changes can require
follow-up edits to callers in other files.

Declaration ordering groups each type with its constructors, exported methods,
and unexported methods. Other functions come last. The rewrite carries comments
with declarations and preserves function bodies and initialization order.

Fixes always run in dependency order and re-analyze after edits:

```sh
gstyle fix .         # preview
# gstyle fix --apply .
```

## Use with agents and scripts

Print the operating instructions for a agent:

```sh
gstyle --skill
```

The TUI exposes a localhost JSON-RPC API on port `8197`. Agents can read reports,
select functions, scroll source, refresh analysis, and wait for a new revision.
Choose another port with `--control-port 9000`.

```sh
curl -s http://127.0.0.1:8197 \
  -H 'content-type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"rpc.discover"}'
```

See [the OpenRPC document](application/openrpc.json) for methods and parameters.

Save quality facts to try a different policy without loading the code again:

```sh
gstyle check --format facts . > /tmp/gstyle-facts.json
gstyle check --facts-in /tmp/gstyle-facts.json --config examples/quality/gstyle.toml --format json
```

Export per-function dependence graphs for your own analysis:

```sh
gstyle patterns --format pdg-json . > pdg.json
```

## Reduce a bug reproducer

Give the reducer an input and a checker for one specific failure:

```sh
gstyle bug-reducer --tui=false failing-input -- ./check-bug.sh
```

The checker receives the candidate file path as its last argument. Exit **0** means
the same bug remains; a nonzero exit rejects the candidate. Make the checker reject
unrelated failures too. The reducer preserves the original and saves the best
accepted input to `failing-input.reduced`, refusing to overwrite an existing output.

In a terminal, the dashboard shows progress, accepted reductions, and checker
output. Press `q` or Ctrl-C to stop and save the best accepted input. Use
`--timeout 30s` to limit each checker invocation.

Default reduction removes whole lines. Go syntax reduction needs Tree-sitter:

```sh
gstyle bug-reducer --language go --go-parser /path/to/go.so failing.go -- ./check-bug.sh
```

See [the reducer guide](examples/bug-reducer/README.md) for setup and examples.
Try the bundled demo from this repository with `sh examples/bug-reducer/demo.sh`.

## Build and develop

```sh
go build -o gstyle .
./gstyle --skill
make check
make cognitive
make lint
make quality-gate
```

The root command and the compatible `cmd/gstyle` entry point share `internal/cli`.
Analysis and command implementations live in
`domain`, `adapters`, and `application`.

File obligation checks follow returned files through resolvable forwarding functions
in the same package (up to eight summary passes), plus the standard `io/fs.FS.Open`
contract. Callers must close the handle or return it again on supported normal
paths. Unresolved consumers, aliases and asynchronous handoffs stay unknown.
External wrapper bodies and arbitrary interface implementations are not inferred.

### SARIF reports

Use `gstyle check --format sarif . > quality.sarif` or
`gstyle patterns --format sarif . > patterns.sarif` to write SARIF 2.1.0.
`check` exits 1 when it reports findings. `patterns` exits 0 on success.
Both commands exit 2 for analysis or output errors.

Reports include rule IDs, messages, URI-escaped file paths, and source positions.
The first site is the primary location; other sites are related locations.
Locations use line numbers. Columns are omitted because Go uses byte offsets,
which SARIF does not support. Unknown line numbers are omitted.
`check` also supports SARIF with `--facts-in` and `--changed`.
See [SARIF verification](verification/sarif/README.md) for schema and test details.
