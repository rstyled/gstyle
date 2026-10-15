# Signed increment and decrement transfer

`make verify-signed-step` verifies the production `integerStep`,
`incrementInteger` and `decrementInteger` bodies in `domain/runtimeguard`.
The normal state transfer calls these helpers for signed 64-bit step facts.
The proofs cover widening, wrapped Go results and exact metadata invalidation.

## Type widths and representation

The arithmetic contract is for signed 64-bit integers. A Lean `w64` is a
64-bit machine word; `sint.Z` interprets it as an `int64`. `stepWord` adds or
subtracts one modulo 2^64, including MaxInt64 -> MinInt64 and the reverse.
Go's signed overflow has this deterministic result under the
[Go integer-overflow rules](https://go.dev/ref/spec#Integer_overflow).

The adapter emits the signed-step fact for `int64` and named types with that
underlying type. It also emits it for native `int` only when the loaded package's
`types.Sizes` reports eight bytes. Unknown target sizes do not qualify. The
classification uses the analyzed target's sizes, not the analyzer's host width.
Narrow signed integers and unsigned integers receive
no fact and retain the existing conservative fallback. Their precise arithmetic
is outside this proof. Fact extraction, dispatch and assignment integration
have Go regressions but are not formally proved in this stage.

An input integer must be in the interval described by `Value.Bounds`.
For the guarded-increment shortcut, a true `BelowLength` flag additionally means
the integer is strictly below a valid Go length. Such a length is at most
MaxInt64, so this input is strictly below MaxInt64. The theorem makes this
assumption explicit; it does not prove the flag's semantic provenance.
`integer_step_interval_sound` needs no length assumption when the flag is false.
The decrement theorem needs no length assumption at all.

## Transfer contracts

For increment, overflow is excluded by either that valid strict length bound
or a known upper bound below MaxInt64. The result keeps the old lower bound
and its presence flag and drops the upper bound. If neither condition holds,
the result is unknown, including intervals that may contain MaxInt64.

For decrement, a known lower bound above MinInt64 excludes underflow. The result
keeps the old upper bound and its presence flag and drops the lower bound.
Otherwise the result is unknown, including intervals that may contain MinInt64.
Unknown bounds never become known from their raw payloads.

These are deliberately widened intervals: ++ does not increase the retained
lower bound, and -- does not decrease the retained upper bound. This avoids
introducing a bound that changes on every loop iteration. This stage does not
prove fixpoint convergence. Negative inputs and contradictory intervals are
covered; membership claims for an empty interval are vacuous.

Every represented input's wrapped Go result remains in the returned interval
under the stated length assumption. Possible overflow produces an unknown
interval, which includes the wrapped result. The execution contract fixes the
complete returned record. Identity, nil/non-nil, guard, upper-check, constant,
length, below-length, extent and extent-presence metadata are cleared.
Only the retained bound and its flag survive. Unknown bound payloads can remain
raw data, but a false flag gives them no interval meaning.

The graph view, state map and fallback routing are outside the helper execution
contract. The production helper takes and returns a Value by value and does not
read or mutate the state map.

## Demonstrated fix and tests

The old transfer loses a nonnegative lower bound after an increment guarded by
`i >= 0 && i < 7`, then reports a false positive at a later guarded index access.
The new known-upper check safely retains that lower bound. The integration
regression fails against the old production files through a Go overlay and
passes against this change.

Go tests also cover unknown bounds, negative inputs, singleton extremes,
overflow in both directions, strict length guards, stale metadata, named types,
width classification and sampled Go-result preservation. Unbounded increments
still lose the lower bound because they can wrap; decrements can cross zero.

## Verification safeguards and remaining stages

Nine named theorems are audited against the existing standard-axiom allowlist.
Seventeen Go mutations exercise direction, unknown flags, signed limits,
retained bounds and stale metadata. Source translation is regenerated and
byte-compared for runtimeguard, interval, PDG and query code. Setup failures do
not count as proof failures. Existing pinned dependencies, private incremental
build copies, targeted mutation modules and two-worker limit are unchanged.
CI adds two shards and leaves prior shard counts unchanged.

PDG/operator/type-width fact extraction, semantic length-flag provenance,
assignment into the state map and other arithmetic remain separate gaps.
Convergence, pointer/alias semantics and package initialization remain separate
stages with their own contracts. No further proof stage is part of this PR.
