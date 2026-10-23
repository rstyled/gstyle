# Scalar block and successor meaning

`make verify-scalar-flow` checks semantic composition of the actual production
`analysis.block` and `analysis.successors` bodies. It regenerates and compares
production translations, binds 53 source files, audits 38 named theorems, and
executes six fault controls. CI runs this profile beside the existing
[worklist execution profile](WORKLIST.md).

## What follows from the contracts

- A scalar map has no stored pointer identities. Numeric constraints, finite
  numeric `assume` execution and joins preserve this invariant. Non-pointer kind
  premises exclude the pointer identities created for missing pointer bindings.
- Every `ValueEvaluation` derivation computes the same result in `valueResolve`
  for some finite fuel. The fuel follows the derivation; no fixed expression or
  syntax-depth bound is added. Existing primitive value meaning therefore
  applies to the exact value returned by the production execution proof.
- Scalar source kinds and an incoming scalar map establish the evaluated
  value's zero identity. Assignments need no separate assumed result-identity
  fact. Their exact numeric bounds and all other bindings are preserved.
- `ScalarMeaningExecution` records operations, value-evaluation derivations,
  primitive value facts and concrete environment updates. Induction erases it
  to the existing execution evidence and derives concrete range meaning.
  `wp_block_scalar_sound` composes it with the actual solver-mode block loop
  (`report == nil`) and establishes scalar identity, metadata and meaning for
  the resulting map.
- `ComparisonEvidence.sound` composes actual evaluation evidence, exact known
  comparison pruning and `finiteAssume_sound`. A concretely feasible edge cannot
  lose its environment. `DecisionEvidence.sound` covers direct comparisons,
  `if` wrappers and unconditional edges.
- Joins retain every environment represented in an earlier input and include
  the new incoming environment. Successor-prefix induction covers arbitrary
  finite lists and repeated targets. `wp_successors_scalar_sound` composes these
  facts with the entire actual successor loop. The postcondition includes the
  pending-queue invariant, valid metadata, scalar identities, preservation of
  earlier inputs, and coverage of each feasible outgoing edge. The source map
  remains unchanged.

These are source-bound Hoare execution contracts. They do not assume successful
report checking, a child block/report soundness theorem, or the result of an
opaque analyzer call. The execution witnesses used by the contracts are finite
derivations whose actual method calls are discharged by the existing proofs.

## Meaning and source obligations remain explicit

`ScalarMeaning` records primitive value, symbol and comparison meaning for one
concrete environment, plus the signed 64-bit input domain. Its fields are
premises about the immutable snapshot; naming or constructing this record does
not prove that the production extractor establishes them. Source lowering must
still establish these facts and the finite derivations for every admitted
program. `metadataAgreement` states exact kind/stability array lookup agreement;
it is not a producer correctness theorem.

The unconditional source-to-range goal remains unfinished. The actual outer
`solve` execution and acyclic termination, `newGraphView`/producer metadata
construction, source lowering, non-nil report mode and final report/call
projection remain open. This checkpoint makes the scalar, metadata and concrete
meaning contracts needed by that composition available; it does not supply the
missing composition by assuming it.

The existing ParseInt execution contract and explicit empty-input ParseUint
execution parameter remain unproved standard-library boundaries. No new trusted
axiom, `sorry`, native proof shortcut, new dependency or production rewrite is
added. Parser/compiler adequacy, the pinned 64-bit Go semantics, package
initialization and resource limits remain stated boundaries. Non-scalar or
multiple writes, arbitrary Go features and total Go operational adequacy are
outside these contracts.

## Independent fault checks

Five Go overlays forge an identity during numeric refinement or pointer merging,
join against the wrong incoming map, or invert the true/false successor
convention. Each overlay must still Go-compile and must fail its named execution
proof. One isolated Lean model fault selects the wrong wrapper condition and
must fail its semantic proof.

Setup, syntax, type, ambiguous-name and unrelated-module failures do not count as
logical rejection. Faults are deliberate negative controls, not claims that
these bugs existed in production. The 44 worklist controls remain in their own
profile; this profile adds six controls for the new composition layer.

## Reuse through refactors

The store, refinement, join and environment-coverage laws are reusable contracts.
The source-bound bridges must be repaired when their actual Go bodies change.
The profile rejects stale translations; a refactor cannot retain coverage by
copying an old success artifact. Full source/report integration remains checked
by `make verify-branches`, with its checker premise kept separate from the open
unconditional implementation theorem.
