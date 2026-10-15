# Production recursive symbol resolution

Run `make verify-runtime-facts`. `wp_symbol_resolve` connects the actual
`graphView.symbol` body to finite resolution over a shared graph snapshot.
Its induction supplies every recursive child execution proof. The theorem has
no child-execution premise.

`wp_symbol_ranked` supplies the finite model witness from a natural-number rank
that strictly decreases on recursive parentheses edges. The rank is a Lean
contract parameter. Production Go needs no new arguments or annotations.

## Snapshot and execution contract

`querySnapshotOwn` owns persistent read shares of the actual view, caches,
operand arrays, kind/tracking metadata, graph records, attribute matrices,
string table and every reference/access/extension row. The generic
`readonlyRows` rules select actual row descriptors and contents, including
zero and invalid IDs. Persistent shares support repeated and shared row reads.
The execution contracts preserve this same complete snapshot at every step.

The model follows production order: zero reference, cached field, node and
attribute selection, direct read and its symbol limit, operator availability,
exact operator string matching, and recursive operand zero. Distinct attribute
records and equal strings stored under different IDs are supported. A terminal
result is independent of excess fuel, and two finite witnesses agree.

The input invariants are explicit:

- Field-cache and operand arrays have the node count. Every operand row has two
  entries, matching the Go array type.
- Each recursive child is zero or an in-range node reference. The initial
  reference has the same requirement.
- Each attribute's selected outcomes row has at most 2^46 entries, retaining
  the existing allocation bound for the 64-bit Go model.
- The snapshot describes the actual stable heap records. Go semantics,
  named-type/layout and source registration assumptions remain explicit.

The finite-witness contract also covers a query that terminates while unrelated
parts of the graph contain cycles. The ranked wrapper requires its supplied
rank to decrease on every recursive model edge. A queried cycle has no finite
witness and is outside these contracts.

These are ordinary WP safety/result contracts. The rank theorem proves finite
model resolution; native runtime resource exhaustion and total correctness are
not established. Worklist convergence is a separate obligation.

Snapshot construction and the stated invariants are not derived from the
production builder in this stage. Returned IDs are exact stored/query data;
semantic cache validity and agreement with source Go bindings remain open.
Full constant/value/conversion evaluation, recursive conditions and state
transfer/control-flow composition also remain unfinished. Program pointer/alias
semantics and package initialization remain separate stages.

## Controls and CI

The combined runtime-query stage audits 56 named results against the existing
standard-axiom allowlist. Canonical Go-to-Goose code is still regenerated and
byte-compared before controls run. The 57 Go mutations still regenerate their
private changed Go source. Thirteen additional Lean model/row mutations reuse
the checked canonical code and change only private proof-tree copies.

Model controls cover zero/cache results, direct-read admission and limits,
operator status/name, operand choice, recursive progress, empty fuel and row
index/absence behavior. The strict diagnostic filter is unchanged: every
control must fail its expected proof module; setup errors do not count.

Twenty-four verifier guard tests include private model isolation, unchanged
canonical generated code, Go regeneration, path/symlink boundaries, bound-source
requirements and refusal to publish success after source drift. Bound Go and
model source hashes are captured before verification and checked again before
writing the success report.

Go regressions exercise chains through 257 nested nodes, distinct attribute
records, shared rows, duplicate decoded text IDs, zero/cache/direct terminals,
exact limits, early unavailable/other operators, operand-one decoys and input
preservation. Existing two CI shards, two workers, private incremental copies,
pinned dependencies and source/audit safeguards remain in place. No CI jobs
are added.
