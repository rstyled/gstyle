# Canonical resource handoff paths

The acquisition query now uses the same universal path proof for release and
handoff. The former may-escape walk is removed. A possible handoff on one branch
cannot suppress a still-owned obligation on another branch. Faults, blocking,
unknown call outcomes and saved deferred continuations use the existing PDG
operation/call queries in both modes.

`Policy.TransferCalls` permits handoff boundaries in addition to Close and
return transfer. The adapter first runs release/return-only analysis. It then
runs the handoff-enabled proof to classify caller responsibility. A handoff is
not recorded as a Close, and it does not prove that the recipient closes the
handle. This remains an advisory protocol boundary, rather than a whole-program
ownership contract inferred for arbitrary foreign code.

## Typed call boundaries

The compiler publishes `gstyle.go:call-entry` (function, method, builtin or
unknown) and `gstyle.go:resource-inputs` (Close-capable formal input positions).
These are typed inventories, not source-spelling tests or a competing path
engine. Variadic inputs and implicit receiver offsets retain their canonical
positions. The shared call reader validates entry kinds, distinct positive
positions and rejects malformed or duplicate inventories. Missing facts cannot
establish a handoff; returned indexes own their slice.

A call handoff needs a resolved non-nil free-function entry, complete canonical
argument values and the tracked resource in a Close-capable formal position.
Formatting, reader-only inputs and builtins do not establish that boundary.
Methods, wrappers and unresolved function values need further entry contracts.
Missing metadata cannot be replaced by a source name. Direct global binding
stores are also boundaries, after their operation effects complete. An unresolved
access cannot prove a direct binding copy, read identity, value origin or global
handoff. Binding transfer records retain uncertainty so origin queries cannot
recover a known source from an unresolved target.

## Actual invocation and value carriers

A deferred handoff occurs at invocation, not registration. Saved argument
identity matters; an intervening abort can prevent the handoff. A newer handoff
can run before an older abort, just as newer Close actions can. Independent
sibling faults are checked before accepting a nested handoff call. A fault that
must consume that call's result remains after the call.

Explicit struct, array, slice, map and key/value constructors retain their
canonical resource inputs when returned. Arbitrary scalar computations or call
results are not carriers merely because they depend on the handle. Constructor
operands keep their own faults. Struct/array/slice construction has no additional
recoverable fault in this model; map hashing retains possible panic when its key
type is not strictly comparable.

A discarded local alias is not a handoff. The earlier ownership fixture now
reports those leaks instead of suppressing them as possible escapes. Richer
heap alias identity, pointer/container handoffs, method/closure entry, recipient body
ownership effects, deferred result/heap state and full concurrent execution
remain required work in the broader completion plan.

Local direct-copy identity, reassignment, saved receivers and parallel binding
stores now use the path-local ownership state in `docs/pdg-owned-bindings.md`.
