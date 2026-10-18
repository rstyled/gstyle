# Ordinary captured-cell writeback

Before this change, `flag := false; func(){flag=true}(); if flag { panic("yes") }`
reports return-or-panic. Native Go panics. Normal-return summaries now retain one
Boolean/nil postcondition per captured cell and apply it to the validated lexical
owner. A separate probe shows an impossible arm after a false store remains
reachable on the parent; joined condition observations now remove it from the
shared dependency CFG.

## Storage and effect contracts

The typed producer records `gstyle.memory:single-cell` as a Boolean symbol fact.
Parameters, named outputs and capture slots denote one cell in an activation.
An ordinary local declaration is admitted only outside syntactic loops, and
functions with goto conservatively exclude local single-cell proofs. For/range
header declarations are repeated too, including Go's per-iteration cells. A range
assignment to an existing outer variable does not declare a new cell. Nested
function bodies use separate activations. Missing metadata is unknown; null,
non-Boolean and duplicate markers are rejected.

Ordinary invocation inputs use current binding values for proved single cells;
repeated families keep the existing invariant history. Normal-return capture
postconditions join in the same finite-exit/result fixed point, including
argument-specific states. Returning target alternatives join with the entry
value for targets that leave a cell unchanged. Nonreturning alternatives do not
supply a normal postcondition. Unknown execution supplies unknown values.

A separate greatest fixed point proves that stores stay within private bindings
and captured cells. Globals, indirect or unresolved stores, unsupported operations,
missing targets and explicit unknown call-site writes prevent this proof. Nested
calls require validated lexical cell links; source revisions must match. The
memory-preservation proof remains separate. Raw PDG may-writes remain available
for incomplete catalogs.

Writeback restores exact current bindings only for proved single cells. Other
cells lose exact binding values and update their conservative family histories;
an older retained loop cell cannot replace the latest binding. Callees with any
registered defer publish unknown captured postconditions. Deferred registration
still clears capture facts: it cannot freeze a live cell at registration time.
Exceptional writeback and unwind-time stores are not modeled by this change.

## Operand evaluation

The Go specification orders calls and required producers without fixing every
sibling read's position relative to a call. See the
[evaluation-order rules](https://go.dev/ref/spec#Order_of_evaluation).
Required argument reads retain pre-call snapshots. A read that can move across a
writer remains unknown, including when the selected CFG traversal visits it
later. Read barriers form part of interned state and survive assumptions,
recovery, call results and captured writes. They remain conservative across
later visits; this is not a complete scheduler for all permitted orders.

## Evidence and limits

Evidence includes 27 source exit cases, three shared-graph cases, nine allocation
proof cases, missing-marker behavior and three cached writer revisions. Two
independent finite oracles cover 1,290 target-choice joins and eight cell-instance
scenarios; the existing 4,096 call-graph/write-set oracle now also checks frame
confinement. Strict marker tests and the unchanged recovery unknown-write test
cover metadata boundaries. Eleven compiling mutants fail their semantic checks:
default/input postcondition loss, unsound single-cell restore/read, loop admission,
global-store admission, target-join loss, deferred-post misuse, two read-order
barriers and deferred registration snapshots.

Three older tests become more precise: a changed unique capture reads its current
value, a writer supplies definite panic, and a false store supplies definite
return. No suppression or complexity-ceiling exception is added. Raw aggregate
and mutation costs of the owned state records remain in the refreshed baseline.

General runtime environments, heap/mod-ref refinement, callbacks, exceptional
capture state, live deferred cells, deferred named-result writes and concurrent
semantics remain required work. Finite tests do not establish full-Go lowering or
add a new full-language proof.
