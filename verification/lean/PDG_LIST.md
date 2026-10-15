# Production PDG list lookup

`make verify-pdg-list` proves the existing generic `listAt[T]` body in
`domain/pdg/attributes.go`. No production Go behavior changes. The function is
used by attribute expansion for reference, access and extension lists.

## Contracts

`wp_listAt_present` returns the exact stored slice descriptor at a valid
one-based reference. `wp_listAt_absent` returns a nil slice for zero and
out-of-range references. `wp_listAt` combines both cases for every uint32
reference. All three execution contracts quantify over the Go element type;
there is no separate hand-written implementation for each instantiation.

The contracts preserve the outer table under fractional or persistent
read-only ownership. The returned descriptor retains its pointer, length and
capacity. A present nil row remains nil. A present allocated-empty row remains
allocated-empty. Missing rows return nil. Nilness alone therefore cannot prove
presence. The outer slice's length, not spare capacity, determines presence.
Nil and allocated-empty outer tables both work.

The project machine model uses 64-bit Go ints. Ownership links the outer
slice's length to the supplied descriptor list and establishes nonnegative
length. uint32-to-int indexing is safe under the valid-reference bound. The
body reads only the stored slice descriptor, so inner row contents and storage
ownership are not needed or certified by these contracts. It returns a shared
view; no copy or independent ownership of row contents is promised.

The existing explicit Go semantics, query named-type semantics, function-unfold
and package-initialization hypotheses remain. No unfinished PDG layout proof or
opaque implementation axiom enters the three audited theorems. Initialization
itself is not proved here.

## Checks and remaining integration

The profile regenerates and byte-compares production PDG and query code and
requires an exact three-theorem standard-axiom audit. Nine production mutations
cover wrong/constant slots, length versus capacity, exclusion of the last row,
zero-reference underflow, loss of present rows, collapse of allocated-empty
rows, forged absence and writes to the borrowed table. Setup errors do not
count as rejected mutations; the diagnostic filter is unchanged.

Go tests instantiate reference, access and extension rows. They cover nil and
allocated-empty rows, nonempty rows, descriptor/backing identity, zero, first,
last and maximum IDs, empty outer tables, spare capacity and preservation of
row contents and descriptors. CI adds one small shard; dependency pins,
private incremental copies, the two-worker limit and existing shards remain.

This closes the generic list-reader boundary. Graph.NodeAttributes, expansion,
outcomes conversion, graph construction, extension lookup and fact provenance
remain integration obligations. Actual child condition and whole transfer
soundness still depend on those layers. Convergence, pointer/alias semantics
and package initialization remain separate stages.
