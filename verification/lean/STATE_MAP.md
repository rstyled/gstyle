# Production state-map join and equality

Run `make verify-state-map` with the existing pinned Lean project. This verifies
actual Go bodies in `domain/runtimeguard`, including `graphView.join`,
`graphView.binding`, `graphView.base`, `copyState` and `equalState`. No production Go body is
replaced by a model. The join calls the existing proved `Value.merge` contract.

## Contracts and assumptions

For two non-nil maps, `wp_join` returns the original maps and graph metadata
unchanged, plus a separately owned result map and an exact lookup contract:

- A key present on the left uses that record merged with the right binding
  lookup. A right entry participates when the binding is stable; unstable
  bindings use the graph base value even when an entry exists.
- A right-only key uses its record merged with the left binding lookup.
- A key absent from both maps is absent from the result.

The loop proofs cover every key in each finite map, independently of iteration
order. Shared keys survive the second loop without being overwritten. Full
ownership of the inputs is returned with their original contents; separate
output ownership also prevents output/input aliasing under the contract.

The input assumptions are explicit in the theorem:

- Both non-nil input maps are separately owned and disjoint. The graph view,
  kind slice and instability slice are owned and remain unchanged.
- The two metadata slices have equal length. Every cached input key is a
  one-based reference within that length. The supplied kind and instability
  functions agree with the slice entries for those keys.
- The existing Go semantics, type/layout, method-unfolding and initialization
  hypotheses hold. As elsewhere, the machine model uses 64-bit Go ints.

The actual binding lookup is proved, not supplied as an assumed callback.
Stable present entries retain the complete record, including present zero
values. Missing signed/unknown bindings use an unknown Value. Missing unsigned
bindings retain the lower bound zero. Missing pointer bindings receive the
actual `uint64(symbol)*2` identity. Unstable bindings use the graph base value.
Index checks and conversions are proved under the reference assumptions.

The [shared metadata contracts](SHARED_FLOW.md) also support read-only graph
metadata and a complete PDG snapshot. Input-map ownership remains exclusive and
disjoint.

## Interval meaning

`wp_join_sound` proves exact union of the stored key domains and interval
soundness for effective binding lookups: any integer represented by either
input lookup is represented by the output lookup. This uses the existing
`Value.merge` interval-preservation proof. Unknown payloads, empty contradictory
intervals, signed extremes and instability are included.

This is conditional abstract-state soundness, not a proof of concrete Go
execution. If an input abstract lookup contains every concrete value of an
incoming state, the output lookup contains those values too. For that concrete
interpretation, kind/instability functions must describe the actual metadata
for every queried key. The execution proof requires slice agreement only for
cached keys, since those are the keys the join visits. It does not prove that
PDG extraction, transfer functions or the caller supplied sound incoming states.
Pointer identity and other metadata have exact execution contracts; this
interval theorem does not establish pointer/alias semantic soundness.

## Nil, empty and copying

Separate execution theorems cover a nil left input, nil right input and both
inputs nil. They preserve and copy the other map exactly, including all raw
Value metadata, rather than applying merge normalization. Both nil produces an
allocated empty map. Empty non-nil maps follow the ordinary two-loop contract;
they are not treated as nil. Copy contracts preserve the source and own a new
map with exactly its keys and values. The nil-branch proofs need no graph-view
ownership because those branches never dereference the receiver.

Treating a nil input as an unreachable incoming state is a caller convention,
not proved here. Exact copying preserves the other side's abstract meaning;
with two unreachable inputs, interval preservation is vacuous.

## Exact content equality

`wp_equalState` proves the actual production size check and map loop return true
exactly when the two maps have identical keys and complete Value records. Each
visited key must exist on the right; a missing key is distinct from a present
zero Value. This fixes the production bug where `{1: Value{}}` and
`{2: Value{}}` compared equal. The loop covers arbitrary iteration order,
unequal values, missing keys and early returns. No input assumptions about
interval validity or metadata consistency are needed for content equality.

Both maps retain their exact contents. The contract takes independent read
shares, so it also covers aliased maps when compatible shares establish the
same contents. This does not extend the exclusive-input join contract to
aliased maps. Concurrent writes are unsupported.

Separate contracts cover nil on either side and both nil. Nil and allocated
empty maps compare equal by content. This function does not decide reachability;
the flow caller separately tests whether the previous input map is nil. The
length proof uses the size bound supplied by the owned-map Go semantics; it
does not assume arbitrary integer-to-word conversions are injective.

## Safeguards and CI

Sixteen named theorems are audited against the existing standard-axiom allowlist.
The audit includes actual execution contracts and the semantic/domain lemmas.
Explicit graph-view layout instances avoid importing unfinished instances for
unrelated opaque PDG types. The resulting proofs may not depend on those opaque
axioms or `sorryAx`; the existing fail-closed audit checks this transitively.
Package initialization itself remains unproved.

The new profile regenerates and byte-compares runtimeguard, interval, PDG type
and query code. The runtime Value profile also binds its newly explicit PDG
imports. Twenty-seven source mutations cover wrong-side lookups, missing or
overwritten keys, nil branch errors, copy aliasing/value loss, unstable lookup,
unsigned defaults and pointer identity. Ten equality controls cover missing-key
checks, wrong keys, ignored/inverted values, omitted length checks and forbidden
input writes. All fourteen Value fields have independent Go regression cases.
Mutation failures must still occur in the expected proof file with a recognized proof diagnostic. The diagnostic
filter and setup-error rejection rules are unchanged.

Private incremental build copies, pinned dependency checks, two mutation
workers, targeted proof modules and shard validation remain in use. The CI
profile keeps two shards; existing profile shard counts and cache paths stay the
same. Go tests cover exact key cases, nil versus allocated-empty inputs, raw
metadata copying, input/metadata preservation and interval membership.

## Unsupported cases and remaining gaps

The Lean join contract does not cover aliased non-nil input maps, invalid cached
references, unequal metadata slice lengths or concurrent mutation. Go regression
tests exercise equal input maps and output independence, but that aliasing case
is not formally proved. The nil cases are covered separately.

PDG extraction, remaining transfers, fixpoint convergence, traversal,
Access/query/Candidate integration, pointer/alias semantics and package
initialization remain outside this stage. The equality contracts do not prove
the actual successor/worklist composition.
