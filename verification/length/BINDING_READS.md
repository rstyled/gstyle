# Production binding-symbol lookup

Run `make verify-runtime-facts`. The existing runtime-query profile also verifies
actual `directBindingRead`, `graphView.symbol` and borrowed `graphView.operand`
execution. PDG Access is a source-selected four-field record, and the Read
category constant is generated from its actual declaration. No replacement
production query or opaque Access layout is assumed.

## Contracts

Four direct-read contracts cover every owned read slice, plus category and arity
short circuits that need no access to the slice's backing storage. Admission
requires category Read, exactly one access and Location zero. Symbol,
UnresolvedReason and Position do not affect this predicate. The complete record
and slice remain unchanged, including all fields that the predicate ignores.

Symbol lookup contracts cover:

- Zero references, including a nil view, return zero without a dereference.
- A nonzero cached field reference returns exactly that value before graph,
  node, attribute, operator or kind lookup. Cache contents are preserved.
- A cache miss follows the actual node and sparse attribute tables. A direct
  read loads its actual Symbol, rejects values above the kind-slice length and
  accepts the boundary value. Zero remains zero.
- Absent attributes return zero. This path needs no table contents or access
  rows. An unavailable operator also returns zero, even if its payload spells
  parentheses. Known operators use the actual string-table lookup and exact
  byte equality; every name other than parentheses returns zero.
- Parentheses read operand zero and return the actual recursive child result.
  The general handoff theorem requires an explicit child execution contract
  over the same resources. Separate end-to-end contracts discharge that
  premise with the proved zero-child and cached-child implementations.

The borrowed operand contract covers both array positions and restores the
view, outer slice, array and entry. Its reference, row and position assumptions
are explicit. Negative positions are outside this contract.

## Assumptions and limits

Nonzero references must select owned field-cache and node rows. Present
attributes must select the stated actual record, access row and outcomes row.
Independent read shares preserve the view, caches, graph, tables and rows.
Present attribute expansion inherits the existing outcomes allocation bound:
at most 2^46 entries under the 64-bit Go model. The direct-symbol conversion
proof uses the fact that every uint32 reference fits a signed 64-bit Go int.

The cache theorem proves exact retrieval, not semantic validity of the cached
symbol. Metadata construction, producer truth and agreement with source Go
bindings remain separate obligations. The kind-slice contents are not read by
the direct-symbol limit check; its contract uses the actual length header.

Finite recursive chains are now covered by the shared-snapshot induction in
[SYMBOL_RESOLUTION.md](SYMBOL_RESOLUTION.md). That contract supplies the child
proof internally under explicit reference, allocation and finite-witness/rank
assumptions. The local handoff theorem remains useful independently. Queried
cycles, snapshot construction and semantic cache/producer agreement remain
outside this stage. Full value/condition/transfer and control-flow composition
remain unfinished. Program pointer/alias semantics and initialization remain
separate stages.

## Verification controls

The fourteen local execution contracts form part of the runtime-query audit
extended to 56 named results by recursive resolution, using only the existing standard-axiom allowlist. The profile checks
exact Go-to-Goose regeneration for runtimeguard, interval, PDG and query source.

Eighteen new valid Go mutations cover category, arity, location, wrong access
rows, zero references, cache selection, symbol limits, operator availability,
exact operator names, operand selection, bypassed recursion and forbidden input
writes. They join the prior 39 Go controls and thirteen recursive model/row controls. Each must fail its expected proof
module with a recognized proof diagnostic; setup errors do not count.

Go regressions check all 256 category and operator-status bytes, empty/single/
multiple reads, all Access metadata fields, zero and maximum references, exact
operator bytes, direct-read limits and input preservation. Independent Go
mutation controls cover all eighteen new mutations. Existing two CI shards,
two mutation workers, private build copies, pinned dependencies and verifier
safeguards remain unchanged; no CI jobs are added.
