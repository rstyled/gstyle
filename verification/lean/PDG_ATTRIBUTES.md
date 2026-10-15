# Production graph attribute lookup

`make verify-pdg-attributes` verifies the actual `Graph.NodeAttributes`,
`Graph.expand` and `Graph.outcomes` bodies in `domain/pdg/attributes.go`.
No production Go behavior changes. The contracts use the already proved
generic list reader and concrete source-selected graph and attribute records.

## Contracts

`wp_NodeAttributes_present` reads the actual AttributeSet at a valid one-based
node reference and calls the proved expansion. Its lookup premise ties the
record to the owned attribute table. `wp_NodeAttributes_absent` returns a zero
Attributes record for zero or out-of-range references. Both preserve the graph
and borrowed storage. `wp_NodeAttributes_zero` proves the short circuit without
requiring graph ownership; this path also permits a nil receiver. Nonzero
references require an owned, non-nil graph, even when they are out of range.
Presence uses slice length, not spare capacity.

`wp_expand` preserves every scalar and complete Fact record, including its
status and value/reason. Symbols, definitions, reads, writes and extensions
return the exact shared row descriptors selected by their own references.
Missing rows are nil, and present allocated-empty rows retain their descriptors.
The caller supplies read ownership of the selected outcomes row; the contract
does not assume that outcomes were already converted correctly.

`wp_outcomes` proves the actual allocation and range loop. Every uint32 word is
copied in order into newly owned result storage. The copied-prefix invariant
accounts for every iteration and the initially zero suffix. `filledPrefix_step`
proves its update. Empty source rows return nil, including allocated-empty rows.
The graph, reference table and source contents remain unchanged. Output writes
do not require or consume write permission to input storage.

All borrowed resources accept a fractional or persistent read-only share. Full
graph ownership includes its other fields and nested metadata; they are returned
unchanged. Inner storage for non-outcome shared views is not read or certified.
The output record still contains borrowed views; it is not a deep copy.

## Assumptions and boundaries

The machine model uses 64-bit Go ints and uint32 references. Source-selected
layouts cover ten concrete records and their nested scalar/fact fields. The
layout module retains all fields while omitting unused generated access lemmas.
It does not import unfinished layouts or opaque implementation proofs.

The allocator model limits an allocation to 2^48 bytes. The outcomes contract
therefore requires at most 2^46 uint32 entries in the selected row. This is an
explicit model/resource bound, not a new Go language rule. Allocation failure
and larger rows are outside this contract. No such bound is needed for missing
attribute references or the zero-reference short circuit.

Go semantics, named-type/layout and source-bound function/method registration
hypotheses remain explicit. Package initialization is not proved here. The
contracts return actual stored facts; they do not certify that a producer's
Known fact is true or that any extension is justified by source semantics.

## Checks

Five execution contracts and the copied-prefix update theorem have an exact
standard-axiom audit. Production PDG/query code is regenerated and byte-compared.
Twenty-one source mutations cover indexing, first/last/zero references,
capacity misuse, discarded records, swapped facts and list references, copied
word corruption/order/loss, incorrect empty-result allocation, and writes to
graph metadata, attribute records or source rows. Setup errors do not count as
rejected proofs. The diagnostic filter is unchanged. The same 21 valid Go
mutants also fail the independent behavioral regressions, including the
distinct read/extension references and nonzero last outcome word.

Go regressions cover nil/empty graph boundaries, spare capacity, distinct
records, every returned field, unknown fact statuses, maximum IDs and words,
shared views, copied outcome storage, empty rows and input preservation. CI
adds two shards for this profile; existing shard counts, dependency pins,
private incremental copies and two mutation workers remain unchanged.

## Remaining integration

Graph construction/packing, extension lookup, producer and fact provenance,
actual graphView attribute access, recursive child conditions, RHS evaluation
and whole state-transfer/control-flow composition remain open. Compound
assignments still have the documented conservative fallback; precise compound
arithmetic is not proved. Convergence, program pointer/alias analysis and package
initialization remain separate stages.
