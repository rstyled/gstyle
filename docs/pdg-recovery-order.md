# Recovery and ordered call boundaries

The shared exit evaluator now distinguishes a direct recover in an executing
function body from ordinary nested calls and a deferred builtin invocation.
Ordinary calls enter without inherited recovery eligibility. A local function
invoked by the unwind machine receives the pending panic context. A direct
recover consumes that eligibility once. A bare deferred `recover()` invocation
returns without consuming the panic; it is not a function body that calls recover.

These control rules follow the [Go panic/recover rules](https://go.dev/ref/spec#Handling_panics).
Native Go cases check direct and indirect recovery, deferred builtin calls,
faults before and after recovery, and recovery with a suspended Goexit.
Whole-interface nilness and copied boolean guards now use the finite state in
`docs/pdg-recovered-values.md`. Concrete recovered payloads and types remain
required work.

## Canonical ordering facts

The compiler publishes `gstyle.go:invocation-order`, a positive function-local
ordinal for each call expression in postorder. This includes calls that compute
the callee before its argument calls. Nested closure bodies retain their own
context; construction does not execute them. Deferred invocation operations do
not become body invocations just because they have an ordinal. Only members of
the admitted operation flow establish a body order.

The reader rejects non-call owners, duplicate owners, duplicate ordinals and
invalid numeric payloads. Missing order or region context remains unknown.
The [evaluation-order rules](https://go.dev/ref/spec#Order_of_evaluation) order
calls but do not impose a total order on independent non-call operands.
Canonical operand dependencies and invocation ordinals therefore establish
bounds: an input required by an earlier call cannot move after a later call;
a value produced by a later call cannot fault before an earlier call.

## Faults across recovery

The evaluator includes independent sibling faults both before and after direct
recovery. The after case matters when the selected CFG traversal puts a fault
first: recovering that fault in the function's own defers can either leave the
inherited panic active or return after it was consumed. A parallel-RHS case
followed by abort distinguishes these outcomes without an outer argument call
exposing the same alternative.

Exit state records the latest direct recovery operation. Faults required before
that operation are not replayed after it by a later call in the same region.
Faults in different statements keep their statement boundaries. Actual calls
retain their canonical language order; the sibling-effect query does not move
call invocations. Resource release and handoff consumers use these same ordering
facts and exit summaries.

This closes direct recovery control for the admitted sequential exit context.
It does not lower all exceptional control/data edges, execute arbitrary closure
or callback bodies, model recovered payload/heap/result state, or establish all
blocking and concurrent behavior. Unknown operations, missing facts and foreign
code retain explicit unknown outcomes. The full completion goal remains open.
