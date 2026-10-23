# Production worklist proof components

`make verify-worklist` checks selected production bodies against regenerated
Goose code, audits the named Lean theorems, and runs deliberate source faults.
This is component verification. The complete source-to-range theorem remains
unfinished; see [BRANCHES.md](BRANCHES.md).

## Current contracts

- `Layout.lean` uses the concrete production `analysis` record. Its queue, input
  state slice and graph-view pointer are not replaced by an opaque layout axiom.
- `Enqueue.lean` executes `analysis.enqueue`. A marked reference leaves all owned
  resources unchanged. An unmarked reference sets exactly its flag and appends
  exactly one reference. The input-state slice and graph-view pointer remain
  unchanged. Valid, positive one-based references and sufficient slice ownership
  are explicit premises.
- `Queue.lean` relates true flags to membership in the unprocessed queue suffix.
  It proves valid references and uniqueness in that suffix, enqueue preservation,
  cursor advancement with the current flag cleared, and exhaustion. A processed
  reference can be queued again. `wp_enqueue_invariant` composes the actual
  enqueue execution with this invariant.
- `ExactComparison.lean` executes both signed comparison helpers over all 64-bit
  words. It proves their returned value and known flag match the mathematical
  integer relation. Unknown operators do not become known.
- `KnownComparison.lean` executes operand lookup, both production value queries,
  singleton checks and exact comparison. It proves that represented concrete
  operand values cannot cause a feasible edge to be rejected. Operand evaluation
  derivations are discharged with the existing source-bound value proof; there
  is no assumed child execution callback.
- `KnownBoolean.lean` executes the actual comparison dispatcher. It excludes
  boolean-literal and recursive logical cases using explicit snapshot facts, then
  calls the proved comparison implementation. Its sound wrapper preserves the
  no-false-rejection result.
- `Successor.lean` proves that a zero condition preserves the state. For an `if`
  decision with a feasible comparison, it executes condition selection, the known
  boolean guard, and the actual finite `assume` derivation. Its sound wrapper
  composes this execution with `finiteAssume_sound`. It does not assume a child
  execution callback or report result. It still requires the metadata and meaning
  premises listed below; it does not prove the successor-list loop.
- `FactSnapshot.lean` now also returns equality for the full expanded attributes
  record. The existing operator/callee contract follows from this stronger
  result. The contract exposes scalar fields and row handles; it does not assert
  that producer metadata has the correct source meaning.

The comparison contracts require valid immutable graph snapshots, supported
value-evaluation derivations, represented concrete values and the existing
`ParseInt` boundary. These premises still need to follow from the source and
worklist invariant in the final composition. Queue flags alone do not establish
range soundness, convergence or report correctness.

## Fault controls

Queue faults ignore or invert the flag guard, lose the flag write, append the
wrong reference, duplicate a reference, overwrite the input-state slice, or
retain the processed flag when advancing the cursor.

Comparison faults reverse signed ordering, use unsigned ordering, mark an
unknown operator as known, use the left operand twice, or allow one non-singleton
operand. Dispatcher faults query the wrong fact, select the wrong comparison
reference, or forge a known false result. Successor faults select the wrong
condition operand, invert pruning, skip refinement, or discard the unconditional
state. An attribute fault substitutes the write-form handle for the construct
handle. Each fault must fail in its affected proof. Compiler, setup and type
failures do not count as proof rejection.

No production Go behavior, trusted axiom or dependency is added by these proofs.
Package initialization, memory/resource exhaustion and Go operational adequacy
remain stated boundaries. Other successor forms, transfer, complete successor
and worklist traversal, report projection and source lowering remain open.
