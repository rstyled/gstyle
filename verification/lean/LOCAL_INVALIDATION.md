# Local-state preservation during field invalidation

`make verify-assignment` includes one prerequisite for plain-assignment transfer:
`graphView.invalidateFields` leaves the complete state unchanged when its records
have no field roots or field identities. This is an execution proof of the
translated production loop, not a replacement model of that loop.

## Review the claim in Go

`analysis.transfer` calls `invalidateFields` before reading the RHS. The loop can
remove a record through its field root or clear an identity through a field
identity. The supported input rules exclude both changes:

- For each key present in the state, `fieldRoots[key]` is zero. A missing entry
  has the same meaning as an explicit zero. Entries for other keys may be nonzero.
- Each record's identity is zero, odd, or has `identity/2 <= len(graph.Symbols)`.
  This admits nonzero local identities, including the exact upper boundary.
  Even identities above that boundary are field identities and are excluded.

`wp_invalidatesField_local` checks the production zero-root early return.
`wp_fieldIdentity_local` checks the production identity classifier.
`wp_localRoot_lookup` covers nil and allocated root tables. The composed
`wp_invalidateFields_local` applies these contracts to every entry in the actual
map loop. Its postcondition owns exactly the original map, so keys, bounds,
unknown-bound payloads, identities, flags and all other Value fields survive.
Interval membership is therefore unchanged without a separate arithmetic premise.

## Preconditions and limits

The state map is allocated, non-nil and exclusively owned. The graph view, graph
record and allocated root table are shared read-only snapshots. A nil root table
is also supported. The empty allocated state needs no per-record conditions.
The graph record is needed because Go evaluates `len(v.graph.Symbols)` even when
the identity is zero. Symbol-slice contents are not read.

The existing 64-bit Go machine model, type/layout, method unfolding and package
initialization assumptions apply. The proof establishes the existing WP
partial-correctness contract; it adds no total-correctness adequacy claim.
No validity or meaning is assumed for the numeric bounds or other metadata,
because the result preserves their exact records.

`TestPlainTransferCopiesRhsAndClearsGuardMarker` explicitly checks these input
rules, checks preservation by the real helper, then executes the real plain
assignment transfer. This is a constructed PDG fixture, not a proof of Go-to-PDG
extraction or of all production callers satisfying the input rules.

Mixed field/local states, field identities, nil state maps, concurrent mutation,
pointer/alias behavior and initialization are outside this contract. RHS
selection, guard clearing, increment fallback, range-write fallback and full
`analysis.transfer` composition remain separate proof work. No production Go
behavior changes in this stage.

## Verification

The assignment profile audits twelve named theorems and runs twenty Go mutation
controls. Five new controls target `LocalInvalidation.lean`: a wrong zero-root
result, a wrong root-lookup key, acceptance of identity zero as a field identity,
an off-by-one identity boundary, and an unconditional state-record overwrite.
Each must fail logically in the designated proof module. Setup, translation and
type failures do not count. Original mutation order and both CI shards remain.

Go regressions compare complete maps across nil, empty, explicit-zero and
unrelated root tables, empty state, zero/odd/even identities, maximum unsigned
identity, signed interval endpoints and unknown bounds. The unrelated-root case
places a nonzero root beside a present key to exercise wrong-key mistakes.
Pinned dependencies, source regeneration, axiom audits, source hashes, isolated
workers, exact evidence caches and existing CI worker limits remain unchanged.
