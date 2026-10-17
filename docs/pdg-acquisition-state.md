# Resource acquisition boundaries

The adapter uses `resourceflow.AfterAcquisition` for an already identified
fresh-or-empty first call result. A generic post-operation query cannot supply
this contract: the resource can exist before its assignment completes, and
caller defers can have been registered before the call.

The boundary requires a reachable call and binding operation in the same
canonical evaluation region, one complete RHS input naming that call, and an
unambiguous direct first-slot write to the tracked binding. The compiler now
publishes RHS input inventories for simple assignment and declaration bindings.
Edges still identify producers. Missing inventories, extra inputs or conflicting
writes cannot establish the binding contract.

`OperationEffects.LaterOperations` identifies effects that can follow the
allocation call's normal return. Required operand producers stay before that
call. A fault or unknown effect during the result binding cannot be skipped in
favor of cleanup or a handoff later in the body. An indirect tuple store can
fail after the resource has been returned; a native Go test also verifies that
the first local write can have completed before the later store panics.

## Existing defers

A backward reachability index excludes prefix paths that cannot reach the
acquisition. The prefix walk retains each applicable saved defer stack until
the allocation return. Paths that never return a resource impose no obligation
for this acquisition. Revisited prefix states with the same stack need not
prove loop termination. A cycle that grows an unresolved stack remains unknown.
Earlier print/println defers are identity continuations for this resource query
under its existing builtin contract, so repeated identity registrations can be
elided. This does not discard their argument evaluation.

Earlier captures are not proof that a defer owns the newly returned result.
They retain false/unknown ownership matches, rather than being relabeled from
the target variable's later assignment. Prior panic, Goexit, abort and unknown
continuations can cancel an ownership return. The suffix query receives the
saved prefix stack when it needs that context.

Cleanup already proved without older frames or return transfer is independent
of those older frames: the new cleanup executes first. This permits a new saved
Close to discharge the resource before an older Goexit or abort acts.

## Handoffs and remaining work

The acquisition-aware path proof cannot use a later handoff to hide a
binding-time failure. Post-binding handoffs now use the same universal walker
as release, including operation faults, nonreturning calls and deferred exits.
A possible handoff alone cannot suppress the obligation. Typed boundary and
value-carrier contracts are described in `docs/pdg-handoff-paths.md`. Richer
heap alias and recipient-state interpretation remains required.

This is a resource-lifecycle query over canonical facts. It does not lower every
individual Go store commit into the shared PDG, execute arbitrary deferred heap
or deferred named-result writes, resolve all preexisting aliases, or support every
acquisition expression. Full partial-store state, recovery, closure/callback
and concurrent semantics remain requirements of the broader completion plan.
