# Production operator availability

`make verify-operator` proves the actual `operatorText` helper used by all four
runtimeguard operator readers: condition dispatch, symbol forwarding, value
evaluation and predicate-definition tracking.

Previously these readers ignored `Fact.Status`. For an unavailable fact,
`Fact.Value` is the reason text. A reason equal to `>=` could create a lower
bound, `parentheses` could forward a symbol/value, and comparison text could
register a predicate definition. Regression tests demonstrate these failures
before the fix. The readers now accept only `Known` facts.

## Contracts

`wp_operatorText_unavailable` covers every status byte other than Known (1).
It returns an empty string without requiring or reading table storage. Even a
nil table pointer is permitted on this path. It does not claim the input graph
passes schema validation; invalid status bytes also fail closed.

`wp_operatorText_known` calls the proved production Tables.Text contract. It
returns the exact string for the fact's ID, or the existing empty-string result
for an absent ID. It preserves the table's six fields and string storage under
fractional or persistent read-only ownership. Known facts require an owned,
non-nil table. The machine model uses 64-bit Go ints and uint32 IDs. The Fact
record layout and status aliases are translated from production source.

The two contracts have an exact standard-axiom audit. They use the existing
explicit Go semantics, named-type/layout and package-initialization hypotheses.
No new opaque function axiom, unfinished layout proof or initialization claim
is added.

## Checks and limits

Eight production mutations test inverted/overbroad status acceptance, rejection
of Known, forged unavailable text, wrong IDs, loss of known text and metadata
writes. Source binding covers runtimeguard, interval, PDG and query translations.
Setup failures are rejected by the existing unchanged diagnostic filter. CI
adds one small profile shard; dependency pins, private incremental copies,
two-worker mutation limit and existing shards remain unchanged.

Go regressions cover bounds, symbol/value forwarding and predicate definitions,
plus all 256 status bytes, known/absent references and nil storage on unavailable
paths. The graph call sites are tested but are not yet formally composed with
the reader contract. The proof does not certify that producer-supplied Known
text is true, graph attribute extraction, extension provenance, recursive child
conditions or full transfer/control-flow semantics. Those remain integration
obligations. Convergence, pointer/alias semantics and package initialization
remain separate stages.
