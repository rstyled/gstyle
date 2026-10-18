# Lexical closure body exits

Closure construction already has a canonical value operation and a stable body
identity. The old package inventory lowers declarations only. As a result,
`func f() { func() { panic("stop") }(); println("after") }` retains a reachable
continuation after a nonreturning closure. Even a deferred empty closure has an
unknown invocation outcome.

Package analysis now lowers lexical closure bodies through the same typed graph
builder as declarations. A declaration facade holds the original type and body
nodes. Synthetic name objects live in one private type-information view per
package; the compiler syntax and loader information stay intact. Nested bodies
receive the existing enclosing-body identity plus their source byte offset.
Construction never executes the body.

The callable catalog owns declaration and closure graphs. They take part in the
same exit/result solve and are published through the source-free program catalog.
`Extraction.Callables` exposes those canonical graphs. `Extraction.Funcs` retains
its declaration-level findings and source-edit contract. Closure bodies do not
become named-function edit targets. Named reports still share canonical storage.

Call-target queries follow the existing canonical value origins, then decode a
unique closure body identity. Stored copies and target joins retain their origin
semantics. Unknown, missing or invalid targets stay unknown. The same invocation
states model direct arguments, deferred argument snapshots, LIFO unwind and
ordinary versus inherited recovery. Normal caller continuations and condition
pruning consume those outcomes.

`TestClosureBodiesComposeExitEffects` covers no-op and panicking calls, deferred
recovery, nested recovery, nested bodies, stored values and copies, arguments,
deferred snapshots, Boolean/tuple results, construction without invocation,
capture uncertainty, capture writes and target joins. The continuation probe
checks reachability and dependency removal. The inventory test checks private
compiler information, nested inventory, source ownership and declaration report
storage. A loaded-package case checks transitive closure exits and caller reachability
across package boundaries. These are finite tests, not a new closure semantics proof.

Captured cells remain unknown inputs to closure bodies. Their writes keep the
existing conservative caller invalidation. Binding closure environments, recursive
captured callbacks, arbitrary callback parameters, package-initializer closure
origins, heap payloads and deferred result-state execution remain open. A standalone
exit query needs the full callable catalog; declaration-only graphs do not supply
missing closure bodies.
