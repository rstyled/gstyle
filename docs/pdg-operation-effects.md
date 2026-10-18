# Canonical operation effects

`callflow.OperationEffects` reads the compiler's per-operation effect facts.
Both local-call exit summaries and resource release queries use this reader.
The facts describe one operation; operands, call bodies and saved deferred
invocations retain their own canonical operations and completion rules.

The boolean fields are `panic`, `nil_input`, `block` and `unknown`. A nil-input
fault requires canonical operand one. The reader rejects absent operands,
malformed objects, unknown fields, duplicate fields, null flags and duplicate
fact records. Missing evidence for an unsupported operation remains unknown.
Blocking retains a possible nontermination outcome. Unrefined known effects include possible normal completion; this is a may-outcome
analysis. [Known nil-input values](pdg-nil-input-exits.md) now refine ordinary
exit evaluation and shared normal continuations in admitted call-free regions.

`nil_input` is separate from other panic causes. The resource policy denotes a
nonempty outstanding resource. Its exact canonical input path can discharge the
nil-input condition for that resource. It cannot discharge faults in another
pointer, a nested or promoted pointer, bounds checks, interface comparisons or
type assertions. The shared query preserves independent panic, blocking and
unknown effects when applying a non-nil fact.

## Operand regions

A selected source traversal is not a must-before proof. Go orders calls and
certain other operations, but leaves some operand evaluation and indexing order
unspecified. See the [Go specification's evaluation-order rules](https://go.dev/ref/spec#Order_of_evaluation).

The compiler tags source operations with function-local evaluation-region
ordinals. Each region bounds operand evaluation for one statement. The query
uses reachable canonical operation flow, invocation ordinals and operand dependencies to
identify sibling non-call faults that can precede a call. It does not promote
source positions or the chosen CFG traversal to language-order guarantees.
A fault that depends on the call's result stays after that call. Effects from
later statements cannot preempt it. Missing region evidence for reachable calls
or potentially faulting operations makes the ordering query unknown; a tag
without an admitted execution context does not establish order.

Before accepting a release or other call continuation, resource analysis checks
those possible earlier faults against the saved unwind stack. Local-call exit
summaries retain their exceptional outcomes too. The query conservatively keeps
some alternatives whose exclusion needs stronger control or value facts.
It is not a complete scheduler for every permitted operand order. Direct recovery now retains both permitted sibling-fault continuations using
the call-order contract in `docs/pdg-recovery-order.md`. Missing ordering
evidence and unsupported operations still retain unknown execution.

## Evidence and remaining scope

Native Go counterexamples and compiler-derived cases cover bounds checks,
pointer reads, integer division, assertions and interface comparisons. Tests
also cover deferred input evaluation, unrelated versus dependent sibling
faults, nested-pointer guards, metadata removal and invalid payloads.

This adds the previously omitted caller-operation panic path for admitted,
already-bound resource obligations. Acquisitions now use the separate
`AfterAcquisition` boundary described in `docs/pdg-acquisition-state.md`;
individual partial-store state still needs full lowering. It does not complete exceptional control/data edges throughout
the PDG, path-sensitive bounds/type facts, full recovery ordering, deferred
result/heap effects, closure/callback execution or concurrent semantics. These
remain required work in the full PDG completion plan.
