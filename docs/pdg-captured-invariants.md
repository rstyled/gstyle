# Captured cell-family invariants

A caller can know `flag := false` while its closure's exit summary still includes
an impossible panic from `if flag { panic("bad") }`. Lexical capture links alone
do not transfer value facts into the body.

Exit evaluation now retains a finite Boolean/nil history for each captured cell
family. A local family starts when a closure first retains its cell; earlier
writes cannot affect a closure that does not yet exist. A family is known only when all observed values agree. Disagreement or
an unknown write makes it unknown. Histories survive later declarations at the
same lexical site, so an older cell retained by a closure is not replaced by the
latest loop iteration's binding. Unknown writes poison capture histories.

Invocation keys append implicit capture slots after the explicit formal slots.
[Proved single cells](pdg-captured-writeback.md) now use current values and
ordinary normal-return writeback; repeated families keep these histories.
Validated owner/body links supply family values; missing or incompatible links
keep capture inputs unknown. Captured bindings initialize the callee state, and
return tuples use the same invocation key. Nested relay bodies can transfer a
constant family through successive lexical owners. History is part of state
identity and survives call-result snapshots, branch assumptions and recovery.

This is an invariant abstraction, not an exact heap or runtime cell-instance
model. A repeated family that changes from false to true remains unknown even if its
latest binding is known. Branch assumptions do not turn an unknown history into
a family invariant. Opaque calls retain their conservative write invalidation.
Unconfined, repeated-cell and deferred writes are not propagated back as exact
current-binding values.

Deferred argument values retain their registration snapshots. Capture-family
facts are explicitly removed at registration: a defer reads live cells at unwind,
and registration-time capture values can become stale. Deferred capture inputs
remain unknown until unwind-time state is modeled.

The solver caches validated environment links only while it owns mutable solve
state. After sealing, observations use read-only lookups or independently owned
link results, including new graph views that miss the cache.

Evidence includes twenty-one source cases, a 780-sequence independent join oracle,
concurrent observations under the race detector, and strict Boolean/unique-marker
boundary tests. Source cases cover constant/nil/boxed values, arguments, returns,
relays, indirect writes, deferred snapshots, uniform loop cells and retained older
cells. Four compiling mutants remove capture input keys, history joining, history
from state identity, or the deferred capture barrier. Each fails its intended
semantic assertion. These are finite tests, not a new full-language proof.
