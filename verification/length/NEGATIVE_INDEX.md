# Negative-index verification

Run `make verify-negative-index` with the shared pinned Lean/Goose project.
This stage covers intervals, the risk decision and stable per-binding selection.
Advisory payload construction and generic node lookup are also proved. PDG
transfer analysis and the outer projections remain outside Lean.

`domain/runtimeguard/interval` owns the actual Go join and risk decision.
`Value.mergeBounds` uses that join; the negative-index matcher uses that decision.
`Value.Bounds` projects the existing runtimeguard record into the four interval
fields. The original record layout and other join metadata stay unchanged.

## Meaning and contracts

A range represents every integer satisfying each known bound. An absent lower
bound means no lower restriction, regardless of the stored Lower payload. It
does not mean zero. Absent upper bounds work the same way. Contradictory known
bounds represent the empty set; no input-validity assumption hides that case.

The actual Go join retains a lower/upper bound only when both branches know it.
It takes the smaller lower bound and larger upper bound, and zeroes unknown
payloads. The exact implementation theorem includes normalization. Its semantic
soundness theorem proves that every integer represented by either input remains
represented by the actual Go result. This covers signed 64-bit extremes without
arithmetic or overflow assumptions.

The risk decision applies only to collection index accesses with upper-check
metadata. For those triggers, an absent lower bound or negative lower bound
produces a warning. A theorem about the actual Go function proves that if its
input range contains a negative integer, it returns true. This prevents a false
safe decision relative to the supplied interval. UpperChecked is pattern-trigger
metadata; it is not an assertion that all control-flow paths are upper-bounded.

`Select` uses a Go hash map and returns the first risky occurrence of each supplied
binding ID, in input order. Safe accesses do not mark their binding as seen. Its
loop proof relates every map lookup to the model's seen set, proves index bounds
and counter safety, and preserves ownership and contents of the input slice.
Selected records retain all fields, including the original source-node reference.
The semantic execution theorem establishes a subsequence of input records, unique
binding IDs, risky-only output and coverage of every risky binding. A composed
execution theorem proves that a triggered access whose supplied interval contains
a negative integer has a risky output record for the same binding. The retained
record can be an earlier risky occurrence; this does not claim every access is
reported separately. Zero and the largest uint32 ID remain ordinary raw keys, as in the former matcher; this does
not validate those IDs against the PDG tables.

Twenty-three interval/pattern theorems are audited: the interval model and execution properties,
selection order/key properties, and the actual Go selector contracts. Thirty-two
Go mutations test bound narrowing, forged knowledge, normalization, trigger and
zero handling, missing deduplication, key confusion, forgotten map writes, safe
accesses hiding later risks, skipped/duplicated access records and forged source
nodes. Each mutation builds only this proof module after the full preflight.

The matcher rejects invalid node references before binding deduplication, so an
invalid first record cannot hide a later valid source node. It resolves selected
nodes through the generic `pdg/query.At` API. That API is proved for any represented
Go element type: successful lookup returns the actual table element, while failed
lookup returns its zero value with false. Both cases preserve the borrowed table;
a present zero record or nil pointer remains distinct from absence. No PDG node
type declarations are moved or replaced by a separate model.

`Describe` builds the exact negative-index payload from path, function, symbol
name and an unsigned source line. Its execution proof covers all fields and text.
A numeric theorem proves the conversion to the 64-bit Go int preserves the line
and cannot make it negative, including uint32 maximum. The payload cannot express
an automatic fix; the common Candidate projection is checked by integration tests.

Go tests compare actual joins against set membership at integer extremes and with
all combinations of known/unknown flags, including contradictory intervals. The
existing typed PDG/negative-index regression suite checks the runtimeguard and
pattern adapters. An independent prefix oracle checks every binding/risk sequence
through length six using three IDs, including zero and uint32 maximum. Additional
cases cover absent trigger flags, positive/zero lower bounds and unknown positive
payloads. Tests check both input preservation and absence of output aliasing. Node-resolution regressions
include zero, out-of-range and uint32 maximum references, nil tables, and an
invalid first occurrence followed by a valid occurrence of the same binding.
Projection tests compare the final Candidate with the proved payload.

The production non-strict lower and upper transfers now use `interval.Lower`
and `interval.Upper`. Their execution theorems establish the exact resulting
record, including unchanged opposite bounds. Semantic execution theorems prove
that each result represents exactly the old set intersected with `x >= bound`
or `x <= bound`. There are no consistency or known-bound preconditions; unknown
payloads, contradictory ranges and signed extremes are included. Eight added
mutants test direction, missing knowledge, unknown payloads and boundary loss.
Go set-membership tests cover both intersections, while adapter tests check that
all unrelated Value metadata survives delegation. The Value projection and
copy-back are still outside Lean, as are strict comparisons and full transfer.

## Remaining work and trust

This does not establish that PDG extraction or transfer functions produce sound
intervals for Go executions. State transfer, widening, convergence, graph traversal,
Value/Access projection, query integration, corpus traversal and the common
Candidate projection/presentation remain outside Lean. Generic table lookup and
payload construction themselves are proved; the complete matcher wiring and
PDG semantics are not. The negative-index pattern is not yet fully verified.

The pinned translation and Go semantics, the 64-bit model, package unfolding and
record-layout hypotheses, compiler and Lean kernel are trust boundaries. Global
axioms are restricted to propext, Classical.choice and Quot.sound. There are no
replacement implementations or sorry proofs. The execution theorems use Iris
weakest preconditions, not a separate total-correctness logic.
