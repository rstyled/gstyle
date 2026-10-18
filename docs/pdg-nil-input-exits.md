# Nil-input faults and ordinary continuations

Before this change, `var p *int; _ = *p` reports return-or-panic. A direct
nil-input fault with a known nil operand cannot complete normally. Conversely,
a known non-nil operand discharges that fault without discharging independent
bounds, assertion, comparison, blocking or unknown effects.

The exit evaluator reads the existing `nil_input` contract and canonical operand
one in its current value context. It applies the fact only in an admitted
operation region without calls. Calls can change bindings in a different Go
operand order; the chosen CFG traversal is not an ordering proof. Missing region
or membership evidence also retains the original may-outcomes. Nested/promoted
pointer faults and independent panic flags remain conservative.

Ordinary observations now join non-call operation outcomes as well as call
outcomes over all visited states. Only a nonempty outcome set with no return or
unknown bit removes a normal continuation. All dependency passes use that
refined operation CFG. Rebuilt graphs retain their original exit context, so
ordinary pruning does not rewrite inherited recovery behavior. Statement-based
fallback execution/control anchors touching an operation proved unreachable in
ordinary flow are removed too. Operations outside that flow, including unwind
invocations, and data dependencies are preserved. An operation appearing in any
reachable block remains live.

Evidence includes twenty-two source exit cases and six shared-graph cases:
nil/non-nil dereferences and fields, method-value binding, boxed nil interfaces,
parameter/result/capture transfer, branch
assumptions, nested pointers, recovery, defers, mixed paths and mutating sibling
calls. Domain checks enumerate nil/non-nil/unknown inputs with independent
panic/block alternatives and reject refinement with incomplete or call-bearing
regions. Dependency checks cover mixed live/dead occurrences and operations
outside the ordinary inventory. Six compiling mutants remove the nil fault,
independent panic, operand-order barrier, operation observation, outcome join,
or stale-anchor removal; each fails its intended assertion.

This does not establish full exceptional control/data dependencies, arbitrary
operand-order execution, exact heap values, deferred state transfer or concurrent
semantics. Those remain open in the work plan. No new full-language proof is
claimed.
