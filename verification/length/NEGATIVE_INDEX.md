# Negative-index verification

Run `make verify-negative-index` with the shared pinned Lean/Goose project.
This stage covers intervals, the risk decision and stable per-binding selection.
Payload construction and the PDG transfer analysis are not yet proved.

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
binding IDs, risky-only output and coverage of every risky binding. Zero and the
largest uint32 ID remain ordinary raw keys, as in the former matcher; this does
not validate those IDs against the PDG tables.

Thirteen theorems are audited: the interval model and execution properties,
selection order/key properties, and the actual Go selector contracts. Eighteen
Go mutations test bound narrowing, forged knowledge, normalization, trigger and
zero handling, missing deduplication, key confusion, forgotten map writes, safe
accesses hiding later risks, skipped/duplicated access records and forged source
nodes. Each mutation builds only this proof module after the full preflight.

Go tests compare actual joins against set membership at integer extremes and with
all combinations of known/unknown flags, including contradictory intervals. The
existing typed PDG/negative-index regression suite checks the runtimeguard and
pattern adapters. An independent prefix oracle checks every binding/risk sequence
through length six using three IDs, including zero and uint32 maximum. Additional
cases cover absent trigger flags, positive/zero lower bounds and unknown positive
payloads. Tests check both input preservation and absence of output aliasing.

## Remaining work and trust

This does not establish that PDG extraction or transfer functions produce sound
intervals for Go executions. State transfer, widening, convergence, graph traversal,
Value/Access projection, source-node resolution and candidate payload construction
and rendering remain outside Lean. The negative-index pattern is not yet fully verified.

The pinned translation and Go semantics, the 64-bit model, package unfolding and
record-layout hypotheses, compiler and Lean kernel are trust boundaries. Global
axioms are restricted to propext, Classical.choice and Quot.sound. There are no
replacement implementations or sorry proofs. The execution theorems use Iris
weakest preconditions, not a separate total-correctness logic.
