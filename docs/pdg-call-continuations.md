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

## Remaining work

The initial integration uses the existing call query's default argument facts.
Values learned only by executing earlier assignments or selecting a branch can
still leave continuation conservative. A follow-up must use the exit solver's
path state at each call. General call results, heap execution, complete implicit
panic lowering, arbitrary operand-order flow, exceptional/deferred state writes,
concurrency and transformation safety remain open in the full PDG work plan.
The selected source-order CFG and retained exit context do not establish full
Go dependency completeness. This change adds no full-language formal proof.
