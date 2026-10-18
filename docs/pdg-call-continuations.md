# Call continuation integration gap

The source-derived probe on master `d5357c3` uses:

```go
func stop() { panic("stop") }
func f() { stop(); println("unreachable") }
```

`callflow.ExitEffects` reports exactly `callflow.Panics` for `example.stop`.
The same extracted package marks the `println` operation (reference 2 in
`example.f`) reachable in `Graph.ValueFlow`. A temporary assertion requiring
that operation to be unreachable fails:

```text
--- FAIL: TestCallContinuationGapProbe (0.02s)
    ir_call_continuation_gap_test.go:28: println after proven panic is reachable at operation 2
```

This is a semantic mismatch, not missing syntax recognition. The normal CFG
and the call-exit solver answer different continuation questions today.
At that revision, `semanticControlDescription` in
`adapters/gopatterns/ir_control.go` explicitly assumes that ordinary calls
return. Package graph construction precedes resource/call summaries.

## Implemented integration

Package extraction prepares the typed source inventory once. A provisional
dependency solve supplies the existing canonical call-exit solver. Only functions
with newly proved nonreturning calls need a second dependency solve. The final
solve applies those outcomes before runtime flow, reaching definitions,
execution dominance and control post-dominance consume the operation CFG.
Package summaries and consumers still share the final graph and node storage.
The private provisional inventory copies mutable tables and attributes; it
borrows compiler facts and the package intern pool.

Known nonreturning calls lose their normal successors. Divergence retains a
self edge, with a terminal alternative when finite exceptional exits also exist.
These synthetic alternatives have no invented Boolean condition. Unknown calls
and calls whose outcomes include Returns retain normal continuation. Deferred
registrations continue; their saved invocations remain outside registration
flow. Unsupported bodies retain partial coverage.

The original body context remains in `NormalFlow.ExitContext`. The exit solver
needs that context for deferred recovery and possibly earlier effects of
unordered sibling operands. Its operations use the same source inventory as
the final graph. Both contexts receive structural validation. Keeping this
context prevents the two regressions found during implementation: lost inherited
recovery outcomes and lost sibling index panics before Goexit. Normal dependency
edges are recomputed; clearing reachability alone is insufficient.

## Verification

`TestCanonicalCallContinuations` covers panic, pure and mixed divergence,
recursion, Goexit, abort, returning/unknown targets, literal arguments, saved
function values, nested calls, defer registration and callee recovery.
Additional source tests check reaching definitions, false-arm control,
divergence preservation, exit-summary preservation, partial concurrent bodies
and validation of refined reachability.

Three valid Go overlay mutants fail the expected assertions: omitting CFG
refinement, replacing divergence with a finite exit and discarding the original
exit context. Setup/build failures do not count as mutation rejection.

Full `make check cognitive lint`, the quality gate and affected race checks
pass. The quality baseline is regenerated. Raw quality findings increase:
source inventory copying needs explicit local mutations, and the new summary
and finalization paths expose more transitive effects. These are not claimed as
quality metric improvements. No suppression or metric-only helper is added.
The selected PDG translation gains declarations for the new context-selection
and validation helpers; existing Graph layouts and proof claims stay the same.

## Path-state call outcomes

At `bb4502f`, these three source regressions still leave `println` reachable:

```go
flag := true; stop(flag); println("after")
flag := false; flag = true; stop(flag); println("after")
flag := false; if choose { flag = true } else { flag = true }; stop(flag); println("after")
```

Here `stop(flag bool)` panics when flag is true. The source tests fail with
`after call reachable=true want false`. Literal-only/default argument facts
cannot read the current caller binding state.

`callflow.OrdinaryCallOutcomes` now observes invocations in the existing exit
evaluator. It joins outcomes over reachable ordinary states at each call site.
Assignments, selected paths, loop states and invalidated bindings use that
solver's existing value semantics. Deferred registrations are not invocations;
an absent map entry proves nothing. This adds no competing value propagation
engine and does not change evaluation when observation is disabled.

The finalizer uses the joined call-site outcomes. A Returns or UnknownExit bit
on any observed path retains continuation. Tests cover both-branch assignments,
one returning arm, zero-iteration loops, indirect overwrites and unknown call
writes. An independent interpreter enumerates both concrete Boolean input
paths for all 18 programs formed from two initial values and three possible
stores on either branch. Its expected invocation outcomes agree with the query.
Three valid Go mutants fail their specific assertions: replacing path-state
outcomes with default argument queries, keeping only the last visited path
instead of joining outcomes, and stopping observation at an unknown prefix.
Observation follows the possible normal continuation of UnknownExit; otherwise
an opaque call on one branch could hide a returning state and make pruning
unsound. The summary solver retains its existing terminal UnknownExit
abstraction when observation is disabled.

## Condition-derived caller facts

At `4b03f2b`, `if flag { stop(flag) }` still gives the selected stop call
`Returns | Panics`, although `stop(value bool)` panics when value is true.
The negated Boolean arm and direct nil/non-nil pointer tests have the same gap.
Four source assertions fail with `selected-arm call outcomes=5 want 4`.

Selected binary CFG edges now add direct whole-binding Boolean or nilness facts
to the exit evaluator's path state. Later writes use its existing transfers and
invalidation. Joined paths retain their own state; heap fields and unknown
conditions acquire no guessed facts. The final normal dependencies now show the
false-arm control of `println` after `if flag { stop(flag) }`.

Symbolic Boolean reads use the existing compiler-owned
`gstyle.go.scalar-normal:resolved-type` primitive record. Type spellings alone
establish nothing. Named Boolean conversions preserve values, and the existing
canonical conversion fact now records that identity. Duplicate, malformed,
extra-field, null and wrong-shape primitive records fail the query; absent or
non-Boolean primitive evidence grants no Boolean identity.

Source cases cover true/negated guards, nil/non-nil guards, named Booleans,
subsequent stores, alias writes, unknown writes and joins. A separate concrete
finite-value oracle checks 24 Boolean/nil programs across guard polarity,
callee polarity and three possible stores. Metadata boundary tests reject
malformed records and distinguish resolved types from original spelling.
Four valid single-change Go overlay mutants fail their specific assertions:
missing selected-edge facts, inverted Boolean truth, inverted nil truth and an
incorrect primitive tag. Full Go checks, complexity/lint, the quality gate and
affected race checks pass. The baseline is regenerated for 4,338 functions;
non-density excess counts stay the same and transitive effect density rises.
No quality-metric improvement or new formal proof is claimed.

## Loaded cross-package call exits

At `68c5bb9`, a caller of an imported `stop.Now` that always panics still marks
its following `println` reachable. The combined `callflow.ExitEffects` query
already proves the imported callee panics. Extraction solves each package alone,
so that fact never reaches its final dependency solve.

`callflow.Program` now owns source-free indexes for finalized functions. A
package solve includes its current functions and the transitive loaded callees
selected by canonical call targets. It reuses the existing finite-exit,
divergence and argument-state solver. Unrelated registered roots are omitted;
missing targets remain unknown. Each graph keeps its own intern tables.

Normal extraction, bounded compiler batches and cached fix families process
resolved imports before their callers. Ordering precedes batch packing, so it
also holds when a package exceeds the batch budget and stands alone. Compiler
syntax/type maps are still released per batch. The program catalog retains no
compiler objects. Provisional inventories never replace finalized registrations.

Cached revisions register unaffected inventories from the existing source
records. Changed dependencies and reverse importers rebuild in dependency order.
A caller-only edit keeps its imported argument contexts; a returning dependency
revision cannot retain an old nonreturning summary. Unchanged revisions skip
catalog reconstruction. Resource consumers receive the same exit scope used by
the graph finalizer, rather than recomputing package-only exits.

Source regressions cover normal, bounded and cached extraction through a
three-package argument-forwarding chain; caller/dependency overlay revisions;
a real compiler-batch boundary; and unloaded-body uncertainty. Exit cases cover
panic, divergence, recursion, Goexit, abort, callee recovery and unknown calls.
Consumer regressions prove a non-nil runtime guard after an imported panic and
deferred resource release through that panic. Selected-callee solving agrees
with whole-inventory solving for the source fixture and excludes an unrelated
root. Public query-boundary and resolved-import-order tests also pass.

Five valid Go overlay mutants fail specific assertions: missing imported scope,
missing ordering in the pure batch policy, discarded unaffected cache entries,
one-level-only callee traversal and package-only resource exit summaries. The
batch-policy assertion does not depend on incidental package-loader ordering.
Planning preserves borrowed import metadata. Full Go checks, complexity/lint,
the quality gate and affected race suites pass. The quality baseline is refreshed
for 4,354 functions. The new traversal owner and query registration need local
state updates; raw aggregate/mutation/effect counts rise. No suppression or
quality-metric improvement is claimed.

## Boolean-constant equality

At `961190a`, `if flag == true { stop(flag) }` still gives the selected call
`Returns | Panics` when `stop(true)` panics. A source probe fails with
`equality-guarded stop outcomes=5 want Panics`. Restoring the base comparison
body in an isolated overlay reproduces that failure; the same probe passes
with the new comparison body.

Equality/inequality between one symbolic Boolean binding and one exact Boolean
constant now preserves that binding's truth relation, including negation and
reversed operands. Constant comparisons keep their existing exact path. Other
operators, two unknown operands and non-Boolean constants grant no new fact.
The selected-edge evaluator uses the relation through its existing assumption
and write-invalidation rules; no new branch engine is added.

A native Boolean oracle checks all 32 valuations across operator, input
negation, operand order, constant and concrete input. Source regressions check
true/false equality, inequality, operand reversal, later writes and the final
false-arm control dependency. Four valid Go mutants fail their expected
assertions: dropped symbolic transfer, inverted constant polarity, lost operand
reversal and treating ordering as equality. Full Go gates, isolated quality
checks and affected race suites pass; the baseline is refreshed for 4,356
functions. No broader predicate or formal-proof claim follows from these cases.

## Remaining work

Compound predicate relations beyond direct Boolean/nil bindings and Boolean-constant
equality, imported
frame/factory contracts and receiver/closure execution still need broader
integration. Cross-package exits cover resolved targets in the loaded scope;
callbacks and unresolved or unloaded bodies retain unknown evidence. Package
initialization execution is outside this contract.
General call results, heap execution, complete implicit
panic lowering, arbitrary operand-order flow, exceptional/deferred state writes,
concurrency and transformation safety remain open in the full PDG work plan.
The selected source-order CFG and retained exit context do not establish full
Go dependency completeness. This change adds no full-language formal proof.
