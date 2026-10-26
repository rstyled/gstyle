# Production solver setup

`make verify-setup` checks actual `newGraphView` construction and the `solve`
prefix before its cursor loop. It regenerates production translations, audits
55 named theorems and rejects ten compile-valid production faults.

Use Go 1.27.3 for canonical generation, as the Lean CI jobs do:
`GOTOOLCHAIN=go1.27.3 make verify-setup`. Standard-library declaration output
differs across Go versions, including the opaque JSON import.

`wp_solve_setup_entry` takes a represented input graph and explicit scalar
structure and allocation bounds. It executes the real nil checks, constructor,
input/flag allocation, fresh entry-map store and entry enqueue. Its conclusions
include:

- a constructed, read-only `querySnapshotOwn` satisfying `ValueSnapshotValid`;
- snapshot identity with the supplied graph and graph pointer;
- separate, exclusive ownership of the empty field-roots map;
- the exact `initialWorkState`, with one reachable empty entry map, all other
  inputs unreachable, one queued entry and its one pending flag;
- `scalarWorkInvariant`, matching container lengths and `reachableQueue`.

No constructor correctness contract, existing snapshot, initial invariant or
successful report certificate is a premise of this theorem. The generic
`wp_solve_setup_entry_bind` helper has an explicit constructor callback; the
final theorem supplies `wp_newGraphView_scalar` to discharge it.

## Construction and scope

`ConstructorInput` records source graph/table/row contents. Its ownership
predicate owns no constructed receiver or constructor arrays. Actual production
execution validates the flow, allocates and fills symbol metadata, applies each
operand edge, initializes field data and runs the scalar tracker. Snapshot
validity follows from the produced arrays and reference checks.

The edge proof covers the full supplied edge list. Non-Data edges, unsupported
positions/labels and invalid references execute the production no-op paths.
Accepted updates retain their actual order, including repeated writes.

The supported input excludes calls, index/dereference/nil tracking roots, field
paths and decoded `parentheses` operators. It permits scalar reads, assignments,
comparisons, branches and nonempty or repeated assignment writes. Node count is
below `2^32`; symbol count is at most `2^43`; the entry theorem additionally
requires at most `2^45` flow blocks. Referenced outcome rows retain the existing
allocation bound. Write symbols must be zero or valid symbol references.

The source context selects the production cursor boundary.
`solve_setup_function_exact` checks that restoring its actual continuation
recovers `solve.impl`; no copied setup implementation is substituted. The
constructor allocation and range boundaries have corresponding source checks.

## Verification and remaining obligations

The ten controls alter input/flag lengths, entry allocation/slot/enqueue, the
view pointer, symbol kind/instability and operand target/value. Each must fail
its named Lean proof. Compiler, syntax, setup and unrelated failures do not
count as rejection. Audit counts describe the inventory, not proof coverage.

The nonempty flow-validator case and three-block entry-last model are positive
controls. They do not establish that every Go source produces admitted data.

O1 is established for this represented scalar subset. O2 remains open: source
lowering must establish graph facts and their concrete meaning. O3 remains
open: source paths must compose through solver execution and report projection.
This is not an unconditional source-to-report or binary-correctness theorem.

Pinned Goose/Perennial semantics, layouts, package initialization, machine and
allocation bounds remain explicit trust boundaries. The inverse map is borrowed
as represented package data. JSON declarations are opaque; empty field metadata
returns before `Unmarshal`, and no JSON execution contract is assumed. Named
proof audits allow only `propext`, `Classical.choice` and `Quot.sound`.
