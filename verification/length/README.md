# Function-length proof pilot

This project uses Lean 4, Iris and Goose/Perennial. The Go runtime has no new
external dependencies. Lean dependencies are development-only and pinned in
`lakefile.toml`, `lake-manifest.json` and `lean-toolchain`.

This directory also hosts the [PDG query proofs](PDG.md) and the
[unchecked-output proofs](OUTPUT.md), and [negative-index core](NEGATIVE_INDEX.md).
Each has its own verification profile,
axiom audit and mutation controls.

The production metric counts physical lines containing Go tokens. Blank lines,
comments and inserted semicolons do not count. The signature and braces count;
every physical line inside a multiline string counts. `//line` directives do not
change physical line numbers.

`domain/quality/linecount/counter.go` contains the production accumulator and
strict threshold comparison. Goose generates `Gstyle/Code` from that actual Go
package; proofgen generates `Gstyle/GeneratedProof`. Neither generated directory
is a hand-written model. `Gstyle/Proof/Length.lean` specifies the new physical
lines as a duplicate-free range and verifies the Go accumulator's boundary
checks, exact increment, unchanged rejection, signed-integer bounds and strict
threshold comparison.

The specification assumes spans arrive in source order, with the next start at
or after the preceding span's last line. The counter checks this at runtime.
Invalid spans fail source extraction instead of producing a clean report.

## Scope and trust

This is a proof of the accumulator and comparison, not the complete analyzer.
Go's scanner, token-to-span extraction, configuration validation, suppression and
diagnostic assembly remain outside this proof. Go regression tests cover that
integration. The trusted translation and Go semantics come from the pinned
Goose/Perennial revision. Its 64-bit platform model applies. Package unfolding,
struct representation, layout and Go semantics are explicit theorem hypotheses;
the audit of global axioms does not remove these hypotheses.

No new axioms, `sorry`, native decision procedures or trusted replacement
implementations are permitted in this pilot. `Audit.lean` lists the global axioms
of the production-code proofs.

## Run

Install elan and Go, then run `make verify-length`. Put elan's `bin` directory on
`PATH`. The pinned Lean version installs through elan. The first build fetches
and compiles Perennial, Iris, Batteries and Qq; later builds reuse Lake's cache.

The command checks regenerated output against the checked-in translation, checks
all nine theorem axiom lists, and retranslates four deliberately broken Go
variants. Each variant must fail in our proof module. Tool/setup failures do not
count as successful mutation rejection. Logs and the success report go to
`dist/verification/function-length/`; stale success is removed before a run.

Dependencies are checked once before mutation tests. A mutation counts as
rejected only for a recognized proof failure in its expected proof module.
Missing modules, syntax errors, unrelated diagnostics and mixed setup/proof
failures abort verification. No unchanged build is added per mutation.

Each mutant builds only its affected proof module. The unchanged audit imports
and all mutation proof targets are built before mutation tests. Lake builds their dependency closure; unrelated
profiles are not built in every CI job. The function-length profile also builds
the `Gstyle` umbrella once, preserving the full project import check.

Mutation checks default to two workers. `GSTYLE_MUTATION_WORKERS` accepts 1–4,
capped by available CPUs. CI uses four on its four-CPU runners. Worker subprocesses
default to one Lean thread and one Go processor; explicit environment overrides
remain supported. The baseline build keeps its normal parallelism.
Every worker uses a private Go/Lean
workspace and a private copy of the successful preflight build artifacts.
Lake checks input hashes and rebuilds changed modules and their dependents.
Pinned dependency builds are shared. A worker failure aborts verification;
result lists retain profile order even when progress messages finish out of order.

Interval, runtime-value, and unchecked-output proofs are split by operation.
The public `Interval.lean`, `RuntimeValue.lean`, and `Output.lean` modules still
export every original theorem.
The baseline build and axiom audits remain complete; mutation builds target the
specific operation contract listed in `verify.py`.

CI caches the project's `.lake/build` per profile, as well as pinned dependency
builds.
An older shared cache is a fallback; Lake still checks every restored input.
Lake invalidates changed inputs before reusing artifacts. Larger mutation suites
run across separate runners with `GSTYLE_MUTATION_SHARD=INDEX/COUNT` (zero-based).
CI first checks every baseline, source binding, and axiom audit in one preflight
job. Shared generation configurations run once. It validates cached mutation
evidence and starts the original shard jobs only for misses. Missing or invalid
evidence executes the controls. Each fresh shard still checks its own baseline,
source binding, and axioms. The aggregate `verify-length` check requires successful
preflight plus every needed shard; a skipped shard matrix passes only when
preflight reports no misses. A complete checked bundle is saved after aggregation.
The matrix in `ci_profiles.json` covers all 432 controls exactly once. Runtime
validation rejects omitted profiles, gaps, and duplicate controls.
Local commands default to one shard and execute every mutation.

`GSTYLE_MUTATION_CACHE` enables local reuse explicitly. The content key covers all
project Lean/configuration files, bound Go sources, dependency contents, translator
binaries, effective compiler settings, verifier code, and selected controls.
Unused preinstalled Go paths from runner images do not affect the key; the actual
selected Go configuration still does. Any project Lean edit invalidates all
profiles. Cached diagnostics must pass the current strict rejection classifier.
Inputs are hashed again before publishing results. `result.json` distinguishes
`executed` and `reused` evidence and records its key; proof logs stay in artifacts.
Clear the local cache directory to force fresh controls.

Warm CI avoids separate runner setup for all 40 shards. Changed proof inputs still
require fresh controls. The Go race-test job has a separate build cache so a
non-race build cannot supply an incomplete warm cache for it.

`make verify-value-evaluation` checks the production runtime value evaluator,
49 theorem audits, and 37 Go/model mutations. CI uses three shards with four CPU-capped
workers. See [VALUE_EVALUATION.md](VALUE_EVALUATION.md) for the finite
query contract, the explicit `ParseInt` premise, and remaining proof gaps.

`make verify-comparison` checks production numeric comparisons, 34 theorem
audits, and 40 Go/model mutations. CI uses four shards with the same two-worker
limit. See [COMPARISON.md](COMPARISON.md) for map preservation, numeric interval
soundness, explicit input assumptions, and remaining proof gaps.

`make verify-one-sided` checks five one-sided numeric soundness contracts and
fourteen semantic mutations in one CI shard. The comparison profile retains its
existing Go execution controls. See [ONE_SIDED.md](ONE_SIDED.md) for operand
interpretation, input assumptions, and remaining integration obligations.

`make verify-shared-flow` checks ten join/logical snapshot contracts and six
production-Go mutation controls in one CI shard. See [SHARED_FLOW.md](SHARED_FLOW.md)
for shared metadata permissions, exclusive map ownership, and the remaining
child-execution obligations.

`make verify-assume` checks sixteen finite-dispatch and interval-preservation
contracts and ten production-Go mutation controls in one CI shard.
See [ASSUME.md](ASSUME.md) for supported certificates, meaning assumptions,
and the distinction between finite recursive expansion and total correctness.

Comparison mutation checks use small orchestration contracts for value and symbol
queries. The original comparison theorems discharge those contracts with the
production value/symbol proofs; their statements and assumptions are unchanged.
Pure numeric/comparison models do not import execution proofs. Nil-state controls
retain their existing theorem statements in a separate module.

Numeric constraint orchestration also uses an integer-transfer contract, which
the original `wp_constrain_numeric` theorem discharges with `wp_integerConstraint`.
Tracking proofs are separate so their controls do not rebuild numeric transfer
proofs. Comparison transfer callbacks are discharged by the existing production
query and constraint theorems; callback assumptions do not replace those proofs.
