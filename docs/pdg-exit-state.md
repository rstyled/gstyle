# Deferred exit state

The deferflow machine schedules one function's saved deferred calls. It now keeps
the pending continuation as well as the LIFO stack: normal return, propagating
panic, goroutine exit or process abort. A panic payload is an abstract value with
an explicit representation kind. Unknown is valid evidence of missing value
information; NilPanicError identifies a known normalized panic(nil) payload.

Go runs deferred calls while panicking and allows a directly deferred function
to recover. Older deferred calls still run after recovery. See the
[language rules](https://go.dev/ref/spec#Handling_panics).
Goexit runs defers but does not itself provide a recoverable panic; see
[runtime.Goexit](https://pkg.go.dev/runtime#Goexit).
A later panic can be recovered without cancelling the pending Goexit. The
runtime's [recovery loop](https://go.dev/src/runtime/panic.go) preserves that
barrier. Native tests exercise it across a caller boundary.

## Invocation outcomes

| Completed deferred call | Machine action |
| --- | --- |
| Returns normally | Keep the pending continuation |
| Recovers its inherited panic and returns | Resume normal return, or the pending Goexit |
| Escapes with a new panic | Replace the inherited panic; retain a Goexit barrier |
| Calls Goexit | Discard the inherited panic and continue goroutine exit |
| Aborts the process | Discard all pending work |

CompleteRecovered reports a proven call outcome. It is not an implementation of
the recover builtin and does not infer whether a nested call can recover. A
panic recovered by the deferred callee's own defers is ordinary completion from
its parent's view. Execution, direct recovery eligibility, named result writes,
memory effects and callback context still need a separate canonical body engine.

Completed exceptional continuations can propagate to caller machines. A normal
return cannot initiate caller unwind. The result's Complete flag distinguishes
a pending exit request from a fully drained stack. The state key includes the
immutable continuation pointer, stack pointer, phase and active-call status.
Noncomparable abstract values do not make the key noncomparable. Go value-copy
semantics still apply to references inside captured or panic values; consumers
must fork their own abstract memory state.

## Resource queries and implicit returns

Resource release now follows exact canonical panic and runtime.Goexit targets
through saved defers. A deferred panic or Goexit can reach an older close; an
earlier process abort cannot. Resource transfer from a return requires the
final continuation to remain a normal return. A deferred panic or Goexit cannot
establish that the caller receives the returned resource. Unknown deferred call
effects cannot establish that a later cleanup runs. Close remains the existing
protocol action, not a proof of an arbitrary Close implementation.

Void functions also need a canonical return when their body falls through.
The adapter now publishes the implicit return already materialized by x/tools
CFG. It uses a synthetic return operation and a zero-width closing-brace span.
It does not invent source text or infer completion from a downstream AST scan.
Terminal calls and infinite paths cannot acquire a reachable normal return from
this rule.

This work does not complete implicit fault lowering, arbitrary deferred-body
execution, source-level recover eligibility, deferred result-state effects,
resolved interprocedural exceptions or concurrent execution. Those remain in
the PDG completion plan. No new formal proof is claimed.

## Canonical local-call exit summaries

`callflow.ExitEffects` now evaluates sequential canonical operation flow and
composes local callees with finite-exit and divergence fixed points. It retains separate ordinary
and directly deferred panic contexts. Outcomes include return, inherited-panic
recovery, panic, panic with a suspended Goexit, Goexit, process abort, possible
nontermination and unknown execution. Finite exits grow from bottom; possible
divergence then descends from top with those exits fixed. Recovery guards and
actual deferred invocation determine reachability, rather than a static call
cycle. See `docs/pdg-exit-fixedpoints.md`. An unresolved callee cannot invent
normal completion.

Deferred registrations compose a finite continuation over the four pending
unwind modes. This preserves LIFO order and supports repeated registration in
loops without imposing a stack-depth limit. Composition uses the same
`deferflow.ResolveInvocation` transition rules as the scheduler. Completed child
panics preserve their own suspended Goexit as well as the parent's barrier.
Missing capture facts on registration operations now fail validation.

The compiler publishes per-operation panic, blocking and unknown-effect facts.
These describe one operation, not a completed operand evaluation. The reader
rejects null, malformed and unrecognized effect fields. Missing unsupported
operation facts yield unknown execution. Receiver wrappers conservatively add
possible panic before a method body starts.

Recovery bodies use a separate `gstyle.pdg:exit-body-flow` context. This does not
relax the ordinary control-flow admission contract. Direct recover consumes the
inherited eligibility in the evaluator, while nested ordinary calls do not.
Recovery summaries now use canonical invocation ordinals and operand dependencies
to retain both permitted sibling-fault orders. Faults ordered before direct
recovery are not replayed afterward. Missing ordering facts remain unknown.
See `docs/pdg-recovery-order.md`. Whole-interface nilness and copied boolean guards now refine branches as
described in `docs/pdg-recovered-values.md`; concrete recovered payloads remain
unknown. Unknown closure bodies, callback context and heap/result
writes remain unknown. Finite external-call contracts remain narrow; this is
not a general proof that external code terminates.

Resource consumers now use these summaries for ordinary and deferred local
calls. Recovery-order proof, richer operation effects and full exceptional
value/memory flow remain required work. No additional formal proof is claimed.


## Resource integration of local-call exits

Package extraction builds canonical body graphs for every local declaration,
including void helpers. Pure scalar cyclic bodies also publish the admitted
body context; fixed-point scalar lowering no longer omits the exit context
merely because it does not use the acyclic normal-flow profile. Local exit summaries pass through
the extraction context to resource policies and cleanup-helper fixed points.

A resource query resolves ordinary and saved deferred targets with
`callflow.CallSites`. Every possible known outcome must discharge the obligation.
A returning ordinary call continues the body; panic or Goexit drains saved
defers. A child panic preserves a suspended Goexit. Abort, possible divergence,
unknown execution and absent or invalid outcome sets cannot establish later
cleanup. Deferred branches fork the immutable scheduler. Close and proved cleanup
helpers retain their separate protocol discharge contract.

Exit control is not a state-preservation contract. A normal local helper outcome
therefore cannot alone prove that a later resource return transfers ownership.
The body state records that uncertainty per path, and deferred local calls also
block transfer without a separate state proof. Saved Close captures can still
discharge ownership. Precise named-result/heap effects remain required work.

The older helper case `io.WriteString(f, text); return errors.Join(err, f.Close())`
now retains its obligation: an opaque writer can panic before Close. A native
Go callback test demonstrates this path. This is a correction of a normal-return
assumption, not a general termination contract for io.WriteString. Admitted faults in the caller's own operations now use the shared
operation-effect query and operand-region checks described in
`docs/pdg-operation-effects.md`. Full exceptional state lowering remains open.

Admitted live cells, exit-time result values and registration loops now use
[value-aware suffix completions](pdg-unwind-values.md). The broader execution
and dependency limits above still apply outside that admitted model.
