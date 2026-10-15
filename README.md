# gstyle

Find complex code, heavy mutation, and repeated patterns in Go. Inspect the results
in a terminal UI, get JSON reports, or hand the tool to a agent.

## Install

```sh
go install github.com/rstyled/gstyle/cmd/gstyle@latest
```

## Start with your code

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
The advisory `unchecked_output` pattern flags discarded errors from typed
`fmt.Fprint`, `fmt.Fprintf`, and `fmt.Fprintln` calls. It covers standalone
calls, assignments, and declarations that discard the error with `_`. Known
`*bytes.Buffer` and `*strings.Builder` destinations are excluded. Best-effort output needs
review; this pattern has no automatic fix. It does not track indirect function
calls, deferred writes, or later uses of assigned errors.

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
go build -o gstyle ./cmd/gstyle
./gstyle --skill
make check
make cognitive
make lint
make quality-gate
```

The command lives in `cmd/gstyle`. Analysis and command implementations live in
`domain`, `adapters`, and `application`.

### Lean verification of unchecked output

Run `make verify-output` with Go, Python 3 and elan on `PATH`. The same pinned
Goose/Perennial project as the length and PDG proofs translates the production
`domain/patterns/outputcheck` package into Lean. The proofs cover exact function
and writer identities, the composed site decision, suppression-name lookup and composed
selection and advisory payload construction. The reporting loop preserves order,
metadata and both inputs. Thirty-four deliberate Go mutations check these contracts. CI preserves
logs and source hashes.

This remains a **partial detector proof**: AST/type extraction, corpus traversal and
Candidate projection are checked by Go tests. Incorrect extracted facts can still produce
incorrect findings. See [the output verification scope](verification/length/OUTPUT.md)
for the contract
and trust boundaries. The former Gobra-only decision pilot is replaced by this
Lean pipeline; Docker is no longer needed.

### Lean verification of the function-length core

Run `make verify-length` with Go, Python 3 and elan on `PATH`. The pinned
Goose/Perennial stack translates the production token-span counter into Lean 4.
Lean proofs cover exact added-line counts, input rejection, integer bounds and
the strict threshold. The command audits theorem axioms, checks that generated
code matches current Go source, and rejects four deliberate Go mutations.

The scanner and analyzer integration remain outside the formal proof. See the
[proof scope and instructions](verification/length/README.md).

### Lean verification of PDG queries

Run `make verify-pdg` to verify actual reference/evidence queries and the composed
semantic-pattern dependency and interface admission rules. The command audits
fourteen theorems, checks generated source, and rejects nine deliberate Go bugs. See the
[PDG proof scope](verification/length/PDG.md); complete graph traversals and
pattern findings are not yet formally proved.

### Lean verification of the negative-index core

Run `make verify-negative-index` using the same pinned Lean project. The current
stage proves that the production interval join preserves both input sets and
that the risk decision reports represented negative values when its pattern
triggers apply. The map-based selector keeps the first risky occurrence per
binding in input order. Generic node lookup and advisory payload construction are
also proved. `make verify-runtime-value` proves the actual Value projection and
comparison copy-back, plus the complete Value join and its interval-set
preservation contract. Numeric state constraints are covered separately below.
Remaining transfers, traversal and outer projections remain outside the proof. See [the negative-index verification scope](verification/length/NEGATIVE_INDEX.md).

### Lean verification of state-map joins

Run `make verify-state-map` to verify the production join and its binding/copy
helpers. The contract covers both/missing keys, nil branches, input preservation
and conditional interval soundness using the existing Value merge proof. See
[the explicit assumptions and unsupported cases](verification/length/STATE_MAP.md).

### Lean verification of numeric state constraints

Run `make verify-numeric-transfer` to verify the production numeric constraint
path: tracking, integer and length constraints, impossible branches and exact
map updates. The soundness contract retains represented integers that satisfy
the branch comparison under explicit assumptions. See
[the contract and remaining stages](verification/length/NUMERIC_TRANSFER.md).

### Lean verification of signed steps

Run `make verify-signed-step` to verify the production signed 64-bit increment
and decrement helpers. The contract covers unknown bounds, negative values,
wrapped results and metadata invalidation. See
[the width assumptions and proof boundary](verification/length/SIGNED_STEP.md).

### Lean verification of assignment writes

Run `make verify-assignment` to verify production RHS eligibility and non-pointer
state-map writes, including unsigned defaults and conservative compound fallback.
See [the contracts and remaining integration obligations](verification/length/ASSIGNMENT.md).

### Lean verification of branch composition

Run `make verify-branch` to verify operand reads and production logical branch
composition under explicit child contracts. It covers sequential refinement,
independent forks and unreachable alternatives. See
[the contract and unresolved child/PDG obligations](verification/length/BRANCH.md).

### Lean verification of PDG text lookup

Run `make verify-pdg-text` to verify the production string-table accessor for
all references under shared read-only ownership. See
[the lookup contract and remaining PDG boundaries](verification/length/PDG_TEXT.md).

The [operator availability proof](verification/length/OPERATOR.md) checks the
production reader that separates known operators from unavailable reason text.

The [generic PDG list proof](verification/length/PDG_LIST.md) preserves exact
row descriptors and absence across production attribute-list lookups.

Run `make verify-pdg-attributes` for the [graph attribute and outcomes
contracts](verification/length/PDG_ATTRIBUTES.md), including real record
selection, metadata preservation and the complete copying loop.
