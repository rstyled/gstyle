# Resolved call memory preservation

Before this change, a no-op between closure construction and invocation erases
an otherwise constant captured value:

```go
flag := false
fn := func() { if flag { panic("unreachable") } }
noop()
fn()
```

The parent reports return-or-panic. With the no-op body supplied, the summary
now proves normal return. A separate parent probe also retains an impossible
conditional arm after a no-op on an addressable Boolean. The existing ordinary
condition observations now prune that arm from shared dependency flow.

This proof is about caller-owned Go storage, not
purity, I/O, termination, or exact values written by an effectful callee.

`ExitSummary.PreservesCallerMemory` is computed from canonical body flow before
exit/result solving. Private whole-binding stores are allowed. Captured/global
stores, indirect/unknown stores, unknown operations, missing bodies and unknown
call targets prevent the proof. Every reachable deferred registration also
requires a preserving invocation. A greatest fixed point removes candidates
that reach an unproved target; recursive components without a writer preserve
storage on every finite execution prefix. A finite intrinsic list uses qualified
builtin identity, so user functions with the same spelling do not inherit it.

Exit value transfer discharges a call's conservative memory invalidation only
when every target in the active catalog preserves memory and the call site has
no explicit unresolved write. The proof survives exit-summary iterations and
remains immutable after solving. It preserves Boolean/nil binding facts and
captured cell-family invariants, including nested readers and argument/result
contexts. It does not transfer exact captured writes back into the caller.

Raw PDG may-write records and dependency edges remain conservative. Removing
those records would make a later query with an incomplete catalog lose its
fallback invalidation. The existing recovery test that injects an explicit
unknown write remains unchanged and passes. Neither a callee-body proof nor a
missing catalog can erase that write. General heap/mod-ref dependencies and
exact state transformers remain required work.

Evidence includes seventeen source cases, four shared-graph condition cases,
an incomplete-catalog condition/invocation observation,
and three cached dependency revisions (reader, writer, reader). The independent
writer-reachability oracle covers all 4,096 three-function call graphs/write
sets. Seven compiling mutants remove the unknown-target boundary, captured-store
classification, deferred invocation check, fixed-point iteration, explicit site
write barrier, transfer discharge, or summary-proof retention. Each fails its
intended semantic assertion. Full-language lowering, captured writeback,
unwind-time state and concurrency remain open; these tests add no new full-Go
proof.
