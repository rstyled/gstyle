# Negative-index verification

Run `make verify-negative-index` with the shared pinned Lean/Goose project.
This is the interval and risk-decision stage of negative-index verification.
Candidate deduplication, payload construction and the PDG transfer analysis are
not yet proved.

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

Six theorems are audited: interval-union preservation and negative-value detection
in the mathematical model, exact execution contracts for Join and NegativeIndex,
and their semantic execution consequences. Ten Go mutations test bound narrowing,
forged knowledge, missing normalization, omitted triggers, zero-bound handling
and trusting an unknown lower payload. Each mutation builds only this proof
module after the full unchanged preflight.

Go tests compare actual joins against set membership at integer extremes and with
all combinations of known/unknown flags, including contradictory intervals. The
existing typed PDG/negative-index regression suite checks the runtimeguard and
pattern adapters.

## Remaining work and trust

This does not establish that PDG extraction or transfer functions produce sound
intervals for Go executions. State transfer, widening, convergence, graph traversal,
Value-to-Range projection, candidate deduplication and rendering remain outside
Lean. The negative-index pattern is not yet fully verified.

The pinned translation and Go semantics, the 64-bit model, package unfolding and
record-layout hypotheses, compiler and Lean kernel are trust boundaries. Global
axioms are restricted to propext, Classical.choice and Quot.sound. There are no
replacement implementations or sorry proofs. The execution theorems use Iris
weakest preconditions, not a separate total-correctness logic.
