# Scalar extraction boundary

The acyclic scalar producer exports each original reaching definition directly
into a read. Reads do not create definitions or replace the reaching-definition
set. For `x:=a; if b { x=1 } else { x=2 }; return x`, only the two arm assignments
reach the final read: each arm kills `x:=a`. The assignments retain their own
symbol and definition identities on those edges. The final read postdominates
the branch, so neither branch outcome controls it and neither individual arm
assignment dominates it. This is a representation contract for this profile;
other profiles still use explicit binding-definition joins.

Short-circuit expression merges remain nonexecuting value entities. Their
incoming data edges use `subkind=value_alternative`, with explicit `outcome`
and zero-based exported alternative slots: true at 0, false at 1. These are
selection slots, not lexical operand positions. For `a && b`, the true input is
`b` and the false input is the saved read of `a`. For `a || b`, true selects the
saved `a` and false selects `b`. No second LHS evaluation or false literal is
introduced. RHS execution/control dependencies retain conditional evaluation.

Each alternative exports its own selecting decision in the existing edge
annotation extension point, for example:

```json
{"gstyle.go:guards": [[{"controller_pdg_node_id": "n8", "outcome": "true"}]]}
```

The outer array is a disjunction of conjunctions of guard atoms. This producer
emits one local decision atom per alternative. Nested expressions keep their
own merge and decision identities; the local guard does not claim that an
inner expression is available outside its enclosing execution region. Consumers
must retain the controller reference as a graph relationship, not compare its
incidental node ID as a string. This annotation and the edge outcome must agree.
The shared JSON schema already permits namespaced annotations and string
subkinds; no schema relaxation is required.

These representations apply to the acyclic scalar profile only. General bindings
and cyclic extraction retain their existing representations.

Routine extraction now checks eligible scalar graphs with the same independent
verifiers used by native matching. Certification runs on the final graph after
package exit refinement. It requires a known source snapshot and target width,
an exact source inventory, binding and interface agreement, CFG transitions,
execution/control completeness, reaching definitions, value dependencies and
projection agreement. The whole body must exclude unspecified sibling-operand
orders, and routine certification requires exactly one scalar output. Void
fallthrough and multiple-output functions keep their partial evidence.

Only graphs that pass these checks report `node_extraction=supported` and
`conformance=conformant`, under `gstyle.go.scalar-normal/0.2.0`. Unsupported or
unverified functions keep incomplete evidence. Missing source/target evidence
also blocks certification. For an admitted scalar function with that evidence,
a verifier failure stops extraction with an error identifying the function and
failed check; it cannot silently downgrade a broken graph to partial support.
Exceptional flow remains unsupported:
certification concerns bounded normal flow. Compiler/runtime agreement, exact
graph equivalence, schema validity and producer evidence remain separate checks.

Regression tests compare original definition identities without expanding
merges, reject killed definitions and incorrect branch dependencies, and check
selected values, positions and controller identities for conjunction,
disjunction, repeated reads and nested expressions. The external six-case
compatibility suite uses independent handwritten expectations, not regenerated
producer snapshots.

The Go/Rust compatibility workflow runs the external suite at a fixed commit
against the PR checkout and a fixed Rust producer revision. It preserves raw
outputs and checks exact graphs, schemas, source identity, spans, evidence,
renaming and runtime results. A second job corrupts those raw outputs while
leaving their conformance claims intact. Missing/reversed/duplicate dependencies,
wrong literals, binding/definition identities and short-circuit selections must
fail the independent oracle. Evidence and source corruptions must also fail.
