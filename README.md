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

Suppress a rule with a reason when the tradeoff is intentional:

```go
// gstyle-allow(fn_params): stable public interface
func Export(a, b, c, d, e int) {}
```

`check` exits **0** with no findings, **1** with findings, and **2** on an analysis,
configuration, or IO error. Start with advisory reports before using it as a gate.

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
