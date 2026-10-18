# Callable values across invocation boundaries

On parent `256ccae`, a named Boolean callback loses its identity at the receiving
parameter:

```go
func disabled() bool { return false }
func apply(cb func() bool) bool { return cb() }
func f() { if apply(disabled) { panic("unreachable") } }
```

The parent exit query reports UnknownExit (128). Native Go returns normally.
The solver now reports Returns (1). Declaration and closure body identities come
from the existing compiler-owned canonical graph facts, without another AST
value or dependency engine.

## Value and execution contracts

An immutable callable set holds a finite may-target inventory. Its constructor
sorts and deduplicates owned names; reads return owned slices. Empty or invalid
names do not establish a target. Producer identities require one nonempty JSON
string. Duplicate, malformed and contradictory callable/scalar facts fail the
query boundary.

Whole-value copies, argument slots, forwarded calls, tuple results, named return
slots and deferred snapshots retain callable identities. Function type
conversions receive a compiler-certified identity transfer. Boxing retains its
separate representation contract. Unknown values do not invent a target. A known
nil function call panics; argument evaluation remains separate canonical flow.
Returning target alternatives join callable sets by union. Other incompatible
value shapes remain unknown, and tuple-slot correlations remain incomplete.

Invocation keys, return tuples, captured postconditions and state keys use owned
JSON encodings. A tuple's explicit unknown slots remain distinct from bottom.
The encoding carries arbitrary identity width and target counts; it does not
assign function identities to a byte-sized enum. State records quote strings
and preserve slot/map boundaries. Loaded program closure includes callable
value references as well as direct/static call targets.

## Effects and control

The invocation solver also carries memory-preservation and frame-confinement
proofs for each input context. These proofs descend from top as reachable
prefixes and child contexts grow. Exit/result evidence grows from bottom.
Ordinary calls and actual deferred invocations contribute their effects,
including prefixes with no finite return. An explicit unresolved-write barrier
still rejects a proof. Captured writes require validated lexical owner links;
a passed closure does not gain a fabricated environment in another frame.
The public argument-independent memory summary keeps its existing contract.

Synthetic switch predicates now publish operation effects. The existing typed
comparison check proves ordinary completion for strict comparable operands,
nil comparisons and type decisions. Interface-containing comparisons retain a
possible panic. Operand evaluation is still separate. This closes the missing
operation record that otherwise stops a many-target switch before its return.

## Evidence and remaining work

Evidence includes 33 source cases and 32 finite native callback traces; a 300-case independent
concrete target-choice oracle across direct, recursive forwarding and saved
invocation; 144 set/shape joins; 1,728 distinct three-slot state keys; 300-target
source and encoding cases; five typed switch source cases and three native
comparisons; a separate source/native abort case; metadata boundaries; and
normal, bounded and cached cross-package catalogs with false/true/false source
revisions. Infinite recursion stays in analysis evidence and is excluded from
the finite native harness. Existing scalar, captured-cell and unwind oracles
retain their original semantic expectations.

General closure activation/environments, bound receiver and interface dispatch,
heap/mod-ref state, callbacks hidden in heap containers, concurrent/blocking
semantics and full exceptional dependencies remain open. A passed captured
reader can still return either Boolean shape, and an unlinked captured writer
still invalidates caller facts. Eighteen compiling fault controls detect lost identities, copies, contexts,
target unions, results, memory effects, barriers, snapshots, nil faults,
comparison faults, catalog references, tuple slots and conversion facts.
These finite checks add no full-Go proof.

[Live closure environments](pdg-closure-environments.md) extend callable targets
with captured-cell bindings and outcome-specific stores. The name-only limits
above describe the earlier stage; general heap and dispatch limits remain open.
