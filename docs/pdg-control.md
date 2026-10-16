# Control dependence

Canonical PDG control edges are semantic dependencies. Lexical containment is
not emitted as a dependency, exported in the PDG, or supplied to matching.
Forward-dominator execution edges remain a separate relation.

## Analyses

The existing scalar-normal profile retains its acyclic operation lowering and
post-dominator-tree analysis. The independent finite-path oracle checks all
33,867 forward-edge DAGs through six vertices, including disconnected vertices.
This change does not broaden that profile's conformance or data-flow claims.

General extraction uses a separate divergence-sensitive analysis. For each real
CFG vertex z, it starts at {z} and adds a predecessor only when all of its
nonempty successor set must reach z. The least fixed point describes vertices
whose every maximal finite or infinite path encounters z. Cycles that avoid z
remain outside it. Mutual post-dominance in unavoidable cycles is permitted.
No synthetic executable exit or immediate-postdominator chain is used.

An outcome x -> y controls operations in post(y), excluding strict
post-dominators of x. Justified self-control edges are retained. The CFG splits
at each operation; basic-block membership does not imply self-control for all
operations in a cyclic block. Edge outcomes retain true/false labels and
one-based internal successor positions (matching uses zero-based arm positions).

The general analysis covers if/for/range, labelled and unlabelled transfers,
expression/type switches, fallthrough and short-circuit expressions. Switch
selectors are explicit predicates. Transfers retain their original source
operation. A private statement view adds lookup markers before transfers omitted
by x/tools; compiler-owned syntax is not changed. Repeated lowerings reuse each
short-circuit predicate. Compiler-known boolean outcomes and short-circuit
constant results remove impossible general-flow transfers.

## Limits

General control evidence is approximate. Calls are assumed to return except
resolved built-in panic, os.Exit and runtime.Goexit. Implicit panics, arbitrary
non-returning callees, call effects, panic recovery, deferred execution and
concurrent/blocking execution are not complete. Bodies containing closures,
defer, recover, go, select, sends or receives get unsupported control evidence
and no lexical fallback. No complete Go PDG or interprocedural claim is made.
Range iteration uses the CFG alternatives, without a type-specific termination
proof. General expression lowering does not prove every Go operand evaluation
order. Data-flow, alias and conformance capabilities remain separate.

The fixed-point calculation uses O(V(V+E)) work and O(V²) bits. General analysis
fails explicitly above 16,384 operation-CFG vertices; it does not report a clean
or complete result after truncation. Control edges can themselves require
quadratic output. This is a correctness/coverage change, not a speed claim.
The general extraction profile is gstyle.abstraction/0.3.0.

## Consumers and checks

Matching receives semantic control edges, including multiple controllers.
Single-parent helpers reject ambiguity. Legacy case/counter/propagation rewrites
run only when the actual control relation is an acyclic hierarchy; they do not
flatten a general PDG into source containment. Other matching behavior is
unchanged. An empty dependency set is not evidence of independence.

A second oracle checks every directed graph through four vertices (66,066),
including self loops and disconnected cycles. It independently searches for a
terminal path or reachable cycle avoiding each candidate. It checks exact
post-dominance and branch-labelled control edges against the fixed-point producer.
Hand-derived source regressions cover transfers, nested control, joins, explicit
exits, constant outcomes, short-circuiting, switches and unsupported constructs.
These finite checks are not a proof for all graphs or all Go semantics.
