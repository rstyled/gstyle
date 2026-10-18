# Call-result facts for exit queries

The callflow solver propagates Boolean and nil shapes from normal returns to
caller exit queries. For example, `func disabled() bool { return false }` makes
`if disabled() { panic("unreachable") }` return normally. Previously the caller
summary also includes a panic outcome.

Return operands belong to the canonical graph. The adapter publishes their
ordered inventory; callflow reads it through bindingflow. The joint fixed point
keeps finite exit outcomes and result tuples together. Empty evidence starts at
bottom; conflicting slot values join to unknown. Divergence refinement preserves
finite result evidence. Argument-specific invocation states carry their own tuples.

Calls snapshot scalar and tuple results before normal continuation. Copies,
forwarding returns and tuple argument packing use the same representation rules.
Boxing a nil pointer into an interface preserves a nonnil interface result.
Named Boolean results start at false when compiler type evidence establishes it.

`TestCallResultFacts` covers exact and mixed returns, recursion, divergence,
nil and boxed nil, tuple copies and forwarding, argument contexts, named result
zeros and writes, unknown calls and recovery. These are finite test evidence,
not a new formal proof.

This change improves exit queries. It does not publish inferred results as
compiler constants or prune every result-controlled branch in the canonical
normal graph. Named results with registered defers remain unknown. Deferred
closures can retain UnknownExit; recovered returns without an explicit return
have unknown results. Heap state, arbitrary values, tuple correlations and full
deferred result-state execution remain open.
