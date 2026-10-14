# Verified PDG query foundation

Run `make verify-pdg` using the same pinned Lean 4 / Goose / Iris / Perennial
project as `make verify-length`. The location is historical; the proof modules
and verification profiles are separate. This adds no Go runtime dependencies.

## Verified production code

`domain/pdg/query` owns a closed set of canonical evidence records and queries.
The main `pdg` package exports aliases, so existing source code and serialized
PDG data retain their API spellings, numeric values and record layout. The
named types now originate in the query package, which changes reflection's Go
package path; this does not change their JSON representation.

Goose translates every declaration in that package. Nothing is axiomatized or
replaced with a trusted implementation. `Gstyle/Proof/Queries.lean` proves:

- a successful reference resolves exactly `ref - 1`, inside the table;
- zero, negative lengths and out-of-range references are rejected;
- successful lookup returns the actual referenced record;
- lookup preserves ownership and contents of the borrowed evidence slice;
- absence remains distinct from a present record with Unknown status;
- unavailable evidence maps to Unknown;
- capability coverage accepts exactly Known, Supported and NotApplicable;
- existing dependency edges accept exactly Known and Supported;
- the pattern's dependency admission rule composes lookup and policy;
- missing evidence cannot make that rule succeed;
- known false remains distinct from every unavailable Boolean payload;
- the non-variadic interface gate succeeds exactly for known false.

Twelve theorem axiom lists are audited against Lean's standard logical axioms.
The actual Go source and record declarations are regenerated and checked against
committed translations. Six deliberately broken Go variants must fail proofs:
accepting zero references, using the wrong slot, rejecting the last slot,
forging a supported status for absence, admitting inapplicable edges, and
forging availability for an unknown Boolean.

## Consumers and limits

Semantic coverage for taint, barrier slices, thin slices and chops uses the
proved dependency admission rule. Malformed evidence references now produce
partial coverage instead of an out-of-bounds panic. `Tables.Text` also uses the
proved reference-index primitive. The non-variadic interface validator uses the proved KnownFalse gate. Go tests
cover those adapters, all 256 status values and all 512 Boolean status/payload
combinations.

This does **not** prove the whole query layer or every consuming pattern. The
proof covers evidence records, reference resolution and the dependency evidence
and interface admission decisions. Graph extraction, snapshot identity, name-to-capability
lookup, whole-graph traversal, finding semantics and rewrite correctness remain
outside it. In particular, supported evidence does not itself establish that
an edge accurately represents a Go execution. The pinned Go semantics and
translation, package representation/unfolding hypotheses and 64-bit model remain
explicit trust boundaries.

Next: define traversal contracts for actual canonical operations, types, result
slots and bindings; prove those queries; then prove a complete finding such as
an explicitly discarded error using those facts. Anti-unification continues to
propose candidates and does not become a trusted proof step.

Dependencies are checked once before mutation tests. A mutation counts as
rejected only for a recognized proof failure in its expected proof module.
Missing modules, syntax errors, unrelated diagnostics and mixed setup/proof
failures abort verification. No unchanged build is added per mutation.
