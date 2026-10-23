# Actual solver cursor loop

`make verify-solver` checks the actual post-constructor cursor loop in
`domain/runtimeguard/flow.go`. It binds 57 source files, audits 35 named
contracts and runs five Go fault controls. The unconditional source-to-report
implementation theorem remains unfinished.

## Exact execution boundary

`solve` is now selected in the production translation. `cursorSite` selects its
actual cursor initializer and lexical tail from that generated syntax. It does
not supply a copied loop or a replacement initializer. Kernel-checked shape and
initializer contracts reject a missing selection or a changed cursor start.
Opaque library values are leaves of this syntax selection.

`wp_solve_cursor` starts after `newGraphView`, worklist allocation, initial input
construction and entry enqueue. Those are explicit open boundaries. The local
locations, immutable snapshot, block rows, input maps, flags and queue must
already be owned. The selected code executes the real cursor initializer, then:

1. Reads the current queue reference and clears its pending flag.
2. Reads and copies that block's input, retaining every other owned input map.
3. Calls the proved production scalar `analysis.block` with a nil report.
4. Calls the proved entire production `analysis.successors` traversal.
5. Advances the actual cursor and repeats until the queue is exhausted.

The block/transfer contracts now preserve the original receiver permission.
They accept exclusive or read-only receiver ownership, so the outer solver does
not discard write access to the same analysis record between calls. The state
maps remain exclusive. No production Go body changes.

The final contract owns the exact final worklist and preserves scalar identity,
kind/stability metadata and every concrete environment represented in earlier
input maps. Its queue invariant implies that no pending flag remains at exit.
This is not a proof that all source-level reachable environments were placed in
those inputs; source meaning and path propagation still need composition.

## Finite runs and acyclic termination

`reachableQueue` requires every queued reference to have a non-nil input. The
proof preserves it through enqueue, joins, prefixes and complete traces. Queue
flags alone do not imply reachability. `solverDataComplete_terminal` is a positive
formation control for every admissible map/queue of a terminal empty block; it
checks that the data-completeness premise is satisfiable, not source correctness.

`SolverStep` contains queue/input/block lookups, concrete scalar operation/value
derivations and branch derivations. `SolverExecution` composes them in the real
FIFO order. Neither contains a child Hoare triple, child correctness theorem,
report acceptance predicate or an assumed sound final result.

A block weight must exceed the sum of all its successor weights. With this
acyclic budget, each real solver step strictly decreases the weighted pending
queue. Repeated references, joins and skipped enqueue requests do not invalidate
the bound: a step can append at most the successor list's total weight, while
processing removes the current reference's larger weight. The transition
relation is well-founded.

`solverExecution_exists` derives a finite execution from that budget and
`SolverDataComplete`. The latter explicitly requires the primitive operation,
value and branch execution data for every pending valid scalar state with reachable queued inputs. It does
not assert their soundness. It is still an unproved source/producer obligation;
naming this data does not establish it for arbitrary Go source.

`wp_solve_cursor_ranked` composes this existence result with actual loop
execution. It does not assume a supplied finite trace. The signed cursor and
queue lengths must fit the pinned machine representation. Its `resource`
premise bounds final queue lengths of the formed derivations; it is not a new
allocation or official-Go total-correctness theorem. The acyclic ranking proves
no infinite chain of supported solver steps; Iris WP alone would not prove
termination. No fixed syntax-depth or program-enumeration limit is introduced.

## Open source-to-report obligations

- Execute and establish `newGraphView`, producer/snapshot metadata and the
  initial maps, queue and entry flag.
- Establish scalar write sites, finite value/branch derivations, complete
  execution data and an acyclic weight budget from actual admitted source
  lowering, including `newGraphView`'s operand and tracking paths.
- Carry full source-path concrete meaning and signed-word domains through the
  completed loop, then execute non-nil report mode and final call projection.
- Discharge or retain the explicit unproved `ParseInt` and empty-input
  `ParseUint` execution parameters. No standard-library body is newly proved.
- Preserve the parser/compiler, Go-machine, package-initialization and resource
  boundaries stated in [WORKLIST.md](WORKLIST.md) and [BRANCHES.md](BRANCHES.md).

The component is source-bound execution, not a universal source-to-report
implementation theorem. No checker-success premise, finite enumeration,
`sorry`, native proof shortcut, new trusted axiom, architecture rewrite or
precision weakening fills the remaining gaps.

## Independent controls and reusable contracts

Each Go fault overlay first compiles independently, then must fail the named
proof: start at cursor one, retain the dequeued flag, lose the copied input,
skip the real block, or stop advancing the cursor. Setup, syntax, type,
ambiguous-name or unrelated-module failures do not count. These are deliberate
faults, not discovered production defects. The verifier's strict diagnostic
filter is unchanged.

The separate worklist and scalar-flow profiles retain their own controls. Local
combined verification can execute their 44 and six controls with these five in
one fresh 55-control run, auditing the union of 140 named theorems. This is an
executed combined profile, not a bundle manufactured from prior success files.
Receiver permission, queue-prefix, input-coverage and weighted-progress laws are
reusable across internal refactors. The source-bound syntax and execution
bridges must be repaired when their actual Go bodies change.
