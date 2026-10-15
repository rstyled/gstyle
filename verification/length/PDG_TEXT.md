# Production PDG string-table lookup

`make verify-pdg-text` proves the actual `(*Tables).Text` method in
`domain/pdg/tables.go`. It calls the existing proved `query.ReferenceIndex`
contract. The selected Tables layout and Text/Capability aliases are translated
from production source; no opaque PDG implementation axiom enters these proofs.

## Contracts

`wp_Text_present` returns the actual string at a valid one-based reference.
`wp_Text_absent` returns the empty string for zero and out-of-range references.
`wp_Text` gives the complete lookup function for every 32-bit identifier.
The index uses slice length, not capacity. The project machine model uses
64-bit Go ints. Owned slice metadata links that length to the supplied list.

All three contracts accept a valid fractional or persistent read-only share of
the table and string storage. They return those resources unchanged. The table's
six fields are preserved, including non-string metadata. Nil and allocated-empty
string slices both work. The receiver must be owned and non-nil; a nil Tables
pointer is not a supported input. String contents are unrestricted, including
empty values and arbitrary byte sequences.

The accessor returns only a string: a present empty string and an absent
reference both yield the empty string. The contracts do not manufacture a
presence flag. Callers that need presence must retain a separate reference/status
check. The current method's behavior does not change in this stage.

The Tables layout uses only its source-bound type/layout hypotheses. It does not
import unfinished layout instances for unrelated PDG types. The existing
Go-semantics, named-type and initialization hypotheses remain explicit. Query
initialization is not proved here.

## Checks

Three named execution theorems have a fail-closed standard-axiom audit. The
profile regenerates and byte-compares PDG and query code. Ten source mutations
check wrong and constant slots, shortened length, capacity misuse, shifted and
wrapped references, inverted/ignored presence, forged missing values and writes
to string storage or table metadata. Setup failures do not count as proof
failures, and the existing diagnostic classifier is unchanged.

Go regressions cover first and last entries, zero and oversized IDs, present
empty strings, Unicode/NUL bytes, nil and allocated-empty storage, spare capacity
and preservation of the data and metadata. CI uses one shard for this small
profile. Pinned dependency checks, private incremental build copies, targeted
modules, two mutation workers and existing profile shards remain unchanged.

## Remaining integration

This closes the production string-reader boundary needed by graph attributes
and runtime facts. It does not prove string interning, graph construction,
attribute/list/extension lookup, typed fact provenance, RHS evaluation or child
condition soundness. Those remain the next PDG/condition obligations. The
assignment and branch contracts retain their documented integration assumptions.
Convergence, pointer/alias semantics and package initialization remain separate
stages.
