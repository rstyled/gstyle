# Live closure environments across returns

On parent `01761b3`, a returned closure retains its body identity but loses its
captured cells. `read := factory(false); if read() { panic("bad") }`, where
`factory` returns a closure reading its Boolean parameter, reports return-or-panic
(5). Native Go returns (1). A factory that captures a function parameter instead
reports UnknownExit (128). Both probes now return (1).

## Cells and identities

A callable target can carry a source revision and body-capture-slot to live-cell
bindings. The compiler's existing lexical environment inventory validates each
binding at construction. Slots refer to cells, not snapshots of their contents.
Sibling closures share an address. Arguments, copies and returned aliases keep
that address. Nested closures can relay it after both lexical owners return.
The immutable callable set retains targets with equal body names but different
cell bindings; code identity alone does not identify an environment.

Each body evaluation owns a cell store. Reads use current cell contents, while
required operand and return-expression snapshots keep their existing lifetimes.
Direct stores update singleton cells strongly. Repeated allocation families use
weak updates; a write through one retained instance cannot replace the value of
all other instances. Unknown/unconfined writes invalidate environment facts.
Named results, ordinary returns and panic/recovery/Goexit continuations publish
separate post-stores after their deferred calls execute. Completing alternatives
join each caller-owned cell’s post-value or unchanged input value. Fresh
allocations retain their producing branch’s value when other branches do not
allocate them. These pointwise joins can lose correlations between different cells. The `recover` intrinsic
retains its explicit pending-panic result during normal value transfer.

A local address identifies an owner function, source revision and symbol. A
returned allocation adds the caller's canonical call-site identity. Existing
input cells keep their addresses. Sites form a finite set, rather than an
unbounded call string. Revisiting a site or colliding with a retained allocation
marks a family as repeated. This abstraction can merge different paths and lose
precision, but does not use an arbitrary allocation or call-depth cap. Cell
references can form cycles without recursively embedding their contents.

Invocation keys include reachable input cells, capture bindings and multiplicity.
Post-stores retain input roots and cells reachable from returned/captured callable
values. New allocations and all their aliases rebase together at the receiving
call site. Effects apply to the same cells even when a callback runs in a helper
with no lexical capture of its own. Explicit unresolved-write barriers remain;
stale revisions, wrong slots and absent cells cannot prove exits or memory
preservation.

## Evidence and costs

Twenty-two source/native cases include reader/writer sharing, separate factory activations,
stores before return and during unwind, function-valued cells, alias forwarding,
same-code closures with different environments, callback writes, relay captures,
panic prefixes and repeated allocation sites. Independent tests check 256
concrete two-object traces, 364 finite-site sequences, 50 multiplicity/write
combinations, 360 unchanged-cell alternative combinations, conditional fresh
allocations, and preservation of input-cell identity. Cross-package normal,
bounded and cached extraction checks false/true/false factory revisions and the
shared normal continuation. Fifteen compiling fault controls fail semantic
assertions for lost writes, split sibling cells, merged activations, strong family
updates, lost multiplicity, rebased input cells, missing post-stores/return rebasing,
missing panic stores/capture links, unbounded site sequences, lost guarded callable
identity, admitted stale environments, discarded untouched alternatives and
widened conditional allocations.

The first 256-trace run takes 159 seconds and completes. Profiling identifies
repeated conversion/JSON work in value-context construction. Per-evaluation
immutable contexts and parsed-value caches reduce the same run to 35 seconds.
Cache lifetimes stay inside the body evaluation; no global mutable cache is
introduced. Operand snapshot code clones a borrowed node map before clearing an
entry. The callable constructor also joins already validated canonical JSON
records directly; it no longer decodes and re-encodes each record. Value
transfer reuses the set’s validated encoding. Empty stores retain their canonical
encoding without reflection. Projections with no new cell identities retain the
existing values and tuples without decoding and re-encoding them.

The first complete local and CI race runs hit Go’s ten-minute package timeout in
existing oracles. The expanded finite-oracle suite makes that budget sensitive
to runner speed. The
race target now sets an explicit twenty-minute package budget. All source,
native and oracle cases remain enabled; the normal test budget stays unchanged.

The baseline records 4,568 functions (4,503 before). Raw excess changes are:
aggregate 54→64, parameters 39→39, mutated targets 61→67, mutation per target
350→360, and side-effect density 28,031,166→28,441,099. These costs include
owned heap records, aliases, context keys, store installation and transitive
effects. No suppression or complexity exception is added. Cyclo agent notes
record the source review; maximum production cyclomatic complexity remains six.

These tests do not prove full Go execution. Addresses are abstract families, not
runtime object IDs. General heap field/index storage, arbitrary values, bound
receiver/interface dispatch, concurrent/blocking semantics, and full exceptional
PDG dependencies remain open. The work plan keeps those requirements active.
