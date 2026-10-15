# Production runtime fact queries

`make verify-runtime-facts` verifies the actual `extension`, `graphView.attrs`
and `graphView.fact` bodies in `domain/runtimeguard/graph.go`. The chain uses
proved production node-attribute expansion, generic list reads and Tables.Text.
Boolean admission recognizes only canonical `true` and `false` facts.

## Lookup policy

`wp_extension` returns the first row whose decoded Name is exactly
`gstyle.runtime:` followed by the requested name. It decodes that row's Value
with the actual string-table reader. A matching empty or missing value stops
the search; later duplicates do not override it. Missing keys return empty.
Names and values are arbitrary byte strings. There is no case folding,
normalization, trimming or JSON interpretation in this reader.

`extensionValue_first` and `extensionValue_missing` expose the corresponding
pure policy results. The execution theorem uses the recursive model of actual
rows and table strings, with a source-preserving range-loop invariant. It does
not assume the supplied rows already have the desired result.

`wp_attrs_present` reads the actual node at its one-based reference, then the
actual attribute record, and calls the proved expansion. `wp_attrs_absent`
covers valid nodes whose attribute reference is zero or out of range. The
lookup premises tie both node and record to their owned tables.

`wp_fact_present` composes that complete selection with actual extension
lookup. `wp_fact_absent` and `wp_fact_zero` prove empty results on their
respective absence paths. There are no assumed child implementations in this
query chain. These contracts preserve the view, graph, node/attribute tables,
shared rows, string table and selected source contents.

## Boolean admission

The generic extension schema allows any valid JSON value. Previously assume
interpreted every nonempty boolean fact except `true` as false. A string-valued
`"false"`, null or a number could therefore discard a feasible true branch.
`booleanContradiction` now rejects only an exact Boolean literal opposite to
the requested branch. Other values cause no Boolean conflict. This deliberately
includes noncanonical whitespace/case forms; it does not normalize JSON.

`booleanConflict_unknown` proves the unknown rule for every other byte string.
`booleanConflict_literals` proves both literal directions.
`wp_booleanContradiction` verifies the actual production predicate.
`wp_assume_boolean_reject` follows the actual source-bound assume body through
the proved fact query and predicate to its nil-map return, preserving metadata.
This path never reads or writes the state map; existing state ownership can be
framed around it. Its premise identifies an actual conflicting stored literal;
semantic truth of producer facts still needs its own contract.

The continuing, recursive and comparison paths of assume remain unproved.
Selecting the complete assume body for source binding does not prove them.

## Ownership and boundaries

Read shares for the view, graph, table/string storage and extension records can
be independent. The graph's node/attribute matrices and outcomes source use the
shared read contract from graph expansion. All shares are returned unchanged.
The selected Extension record is translated from its two production fields;
all existing graph and table metadata remains owned and preserved.

The absence paths have different pointer requirements:

- `attrs(0)` needs no view or graph ownership and accepts a nil view.
- Empty extension rows need no graph/table ownership and accept nil pointers.
- `fact(0, name)` reads the view's graph field before calling attrs. It requires
  an owned non-nil view but can have a nil graph.
- Nonzero node references must be valid in the owned node table. Invalid node
  references are unsupported; invalid attribute references return zero data.
- Nonempty extension queries need an owned non-nil graph and table. Missing
  string IDs return empty; nil and allocated-empty string storage are supported.

The machine model uses 64-bit Go ints and uint32 references. Present attribute
expansion inherits the outcomes bound of at most 2^46 uint32 entries under the
allocator model's 2^48-byte limit. This bound also applies to present fact
queries because the production attrs method expands outcomes before fact reads
its Extensions field. Larger allocations and allocation failure are outside
this contract. Zero and absent attribute paths need no allocation bound.

## Validation and remaining integration

Twenty-four execution contracts and four model policy theorems have an exact
standard-axiom audit. Source regeneration byte-compares runtimeguard, interval,
PDG and query code. Fifty-seven valid source mutants fail their expected Lean
module and independently fail the Go regressions. Controls cover namespace,
key/value fields and IDs, match direction, first/late/absent behavior, forbidden
writes, node selection, discarded metadata and fact dispatch.

Go regressions include duplicate keys (including different IDs with equal decoded names), an empty/missing first value, zero and
maximum IDs, missing/empty tables, byte/empty names, namespace/case distinctions,
reordered node records, zero and absent paths, shared views and input preservation.
The existing two profile shards also cover twelve Boolean controls; no CI job
is added by Boolean admission or binding-symbol lookup. Dependency pins, private incremental copies,
targeted mutation builds, two workers, existing shards, setup-error rejection
and the strict diagnostic filter remain unchanged.

The query result is actual stored data. An empty result alone does not prove
absence, and a nonempty result does not prove semantic truth. Further typed decoding,
producer/fact provenance, metadata-cache agreement, RHS/constant/conversion
evaluation, actual recursive conditions and whole state-transfer/control-flow
soundness remain open. Precise compound arithmetic remains outside the existing
conservative fallback proof. Go semantics, named-type/layout and source-bound
registration hypotheses are explicit; package initialization is not proved.
Convergence and program pointer/alias analysis remain separate stages.

The profile also covers production binding-symbol queries; see
[BINDING_READS.md](BINDING_READS.md) for contracts, explicit recursive-child
obligations and the additional controls.
