# Owned values in local bindings

Resource release and handoff proofs now carry a path-local set of bindings that
hold the acquired value. The initial binding starts owned. A canonical strong
copy transfers that tag; a direct overwrite removes it. The original variable
can lose its tag while an alias or saved deferred receiver retains the value.
A discarded local copy still leaves the obligation open.

The interpreter uses `bindingflow.Writes` and canonical value operands. It does
not recover copies from AST names or source positions. Branches receive separate
immutable snapshots. Memo keys include owned bindings and the saved defer stack,
so a proof from one branch cannot discharge a different state at a shared block.
Repeated control points still retain the existing conservative cycle policy.

All RHS ownership reads use the state before a parallel assignment. Direct
stores commit by canonical binding-slot order. Repeated destinations keep their
last slot. Access-table order has no semantic role. A write without a slot
cannot establish the final value and is applied as an unknown overwrite after
ordered writes. Unknown/conflicting operands and weak copies lose their tag.
Unknown binding-write targets clear the current set. Known memory locations
retain their separate canonical storage meaning; they are not direct copies.

State updates occur only on normal operation continuation. Fault and unwind
paths use the earlier state. Deferred receivers and arguments retain saved tags,
so later variable reassignment does not change the captured value. An alias
created before acquisition does not acquire the newly returned value. Explicit
and named return paths, Close calls and handoff boundaries use the current set.

This closes local direct-copy identity in the resource consumer. It does not
execute deferred bodies, infer recipient ownership contracts, resolve general
heap/container aliases, or establish all partial-store, recovery, callback and
concurrent behavior. Address escapes and captures of owned bindings remain
conservative barriers. The full completion plan remains open.

Canonical slot ordering now lives in `bindingflow.Writes` and is shared with
recovered-value transfers. Resource ownership keeps its separate protocol tags.
