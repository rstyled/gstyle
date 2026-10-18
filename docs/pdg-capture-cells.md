# Lexical capture cells

A lowered closure body previously labels a free variable as local. An
intermediate closure can also omit a cell that only its nested closure uses.
That loses both the body role and the relay dependency at construction.

The adapter now uses compiler scope ancestry and variable identity to distinguish
captures from parameters, locals, globals, constants and fields. Captured symbols
carry CaptureRole and a captured-storage marker. Direct and relayed capture cells
use the conservative general binding profile rather than the closed local
profiles. The memory model gives each captured cell external storage; captured
reference values start with unknown external referents. Captured cell definitions
are memory inputs, not zero-valued local declarations.

Each lexical body publishes `gstyle.closure:environment` on its graph:

```json
{"owner":"example.f","cells":[{"symbol":"s1","source":"s2"}]}
```

`symbol` belongs to the body, and `source` belongs to its enclosing graph. An
intermediate closure owns an explicit relay symbol, so nested mappings form a
chain through the enclosing body. IDs are lexical cell families; they do not
identify runtime allocations or individual loop iterations.

The source-free `bindingflow.CaptureBindings` query validates both graphs,
owner identity, known nonempty matching source revisions, names, types, scopes, roles,
complete capture inventories and one-to-one cell bindings. It compares text
across intern pools. Absent metadata yields no evidence; an explicit empty
inventory yields a nonnil empty map. Empty maps do not establish purity or
absence of global reads. Invalid or ambiguous metadata returns an
error. Missing compiler scope information also prevents publishing a closure
environment. Returned maps own their data; borrowed graphs remain unchanged.

`TestLexicalCaptureCells` covers fourteen cases, including shadowing, receivers,
named outputs, generic parameters, range cells, nested relays and pointer writes.
Boundary cases reject missing/duplicate cells, shadowed same-name variables,
mixed, empty or unknown source revisions, duplicate environments and invalid graphs.
The cross-pool case checks representation independence. Memory-model evidence
checks addressable captured storage and unknown external pointer referents.

This establishes lexical cell identity and conservative memory dependencies.
[Constant cell-family histories](pdg-captured-invariants.md) can feed ordinary
invocation inputs. Exact runtime environments, cross-invocation captured writes,
closure instances and deferred state updates remain required work. The tests are finite evidence, not a new
formal proof of closure semantics.
