# Recovered values in canonical exit flow

The exit evaluator now carries a finite value state: unknown, nil, non-nil,
false or true. A direct inactive recover returns nil. An eligible recover has a
non-nil whole interface value under the recorded Go 1.21+ language-default
rules. Older, missing or invalid language versions retain unknown nilness.
This does not infer the payload's concrete type or the nilness of an unboxed
pointer. Runtime compatibility overrides need a separate execution contract.
See the [recovery rules](https://go.dev/ref/spec#Handling_panics) and
[Go compatibility settings](https://go.dev/doc/godebug).

Each recover call records its own result before eligibility is consumed. A
second call is nil, while an earlier copied value retains its first result.
Branch selection uses the shared `bindingflow.Evaluate` query and canonical
condition operands. Unknown conditions keep all successors. Native cases check
ordinary and active recovery, panic(nil), typed nil payloads, a second recover,
and legacy nil-panic behavior.

## Shared binding transfers

`bindingflow.WriteValues` transfers whole-value nilness and boolean facts through
canonical strong copy slots. Representation facts distinguish unchanged values
from boxed values: a nil pointer boxed in an interface is non-nil. Missing
operands or representation facts cannot establish a value. Zero initialization
needs one explicit nilable-type fact. Conflicting or missing type facts remain
unknown. Other scalar values and general conversions remain unknown.

All RHS values use one pre-store context. `bindingflow.Writes` now owns slot
ordering, including repeated destinations and unknown slots; resource ownership
uses this same ordering. Overwrites kill stale facts, and an unknown binding
write target clears current binding facts. Stored value and node maps are
immutable snapshots. Memo keys include the complete value-state key as well as
recovery and deferred continuation state. Updates occur on normal continuation.
Partial exceptional stores and deferred result/heap writes remain required work.

## Conditional operand faults

Fault-order queries now derive necessary branch guards from admitted operation
flow. Every entry path must pass the guard, and exactly one arm must reach the
candidate before retesting that guard. A sibling after the arms join is not
conditionally guarded. Conditions and actual operand dependencies can therefore
require a recover or handoff call before a short-circuit RHS fault.

Synthetic short-circuit decisions now publish their canonical operand inventory
and evaluation region. They are created after the original source-fact pass;
missing those facts must not trigger a source fallback. Known false guards
exclude disabled faults from before/after recovery alternatives. Guard value
filtering stays within one evaluation region. It does not reevaluate an outer
statement's earlier decision after a later store changes the binding.

Tests retain independent faults after a logical join and faults before abort
inside a previously selected outer guard. These prevent selected CFG order or
current binding values from becoming false language-order proofs.

This improves admitted sequential recovery and its cleanup consumers. Full
recovered types/values, general conversions, deferred body/result/heap execution,
callbacks, complete exceptional PDG edges and concurrency remain open.

The quality review keeps the aggregate finding on `scalarCFGBuilder.logicalValue`:
changing a canonical node's join kind and its explanatory attributes is one
semantic operation across two records. Splitting it only to reduce the metric
would hide that coupling. No suppression is added. Other new state transitions
construct complete snapshots, and the CFG tail has one direct successor.

Canonical nilness/Boolean argument contexts now cross local calls and are saved
at deferred registration. Their contract and remaining call boundaries are in
`pdg-call-arguments.md`. This does not add deferred result or heap execution.
