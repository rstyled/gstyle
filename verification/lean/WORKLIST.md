# Production worklist proofs

These are source-bound component proofs. The unconditional source-to-report
implementation theorem remains **open**. [BRANCHES.md](BRANCHES.md) describes the
separate structural report checker; its success premise is not an implementation
correctness proof.

## Run and audit

| Command | Named axiom audits | Fault controls | Scope |
| --- | ---: | ---: | --- |
| `make verify-worklist` | 61 | 44 | Queue, decisions, scalar transfer, block and successor execution |
| `make verify-scalar-flow` | 37 | 6 | Scalar identity, metadata and concrete range meaning |
| `make verify-solver` | 35 | 5 | Actual post-constructor cursor loop and acyclic progress |

Each profile checks pinned dependencies, regenerates and compares the selected
production translation, binds its sources and audits the named theorems. CI
splits the worklist controls into two shards. A fresh combined run can execute
all 55 controls and audit all 133 names; collecting old success reports is not a
combined execution. `make verify-branches` separately checks supplied-source
report integration and its eight controls.

Seven unused convenience wrappers are removed. Their original statements
reconstruct without new premises: exact comparison execution plus
`knownComparisonResult_no_false_rejection`; successor execution plus
`successorResult_feasible`/`finiteAssume_sound`; and `wp_decision` plus
`DecisionEvidence.scalar`/`sound`. Syntax-selection and positive formation checks
remain. Audit-name counts are an inventory, not a coverage measure.

## Execution and meaning covered

- `Layout` and `FlowLayout` provide concrete `analysis`, `NormalFlow` and
  `FlowBlock` layouts, not opaque record axioms. `FactSnapshot` proves equality
  of the full expanded attributes record, without asserting its source meaning.
- `Enqueue` executes actual `analysis.enqueue`: an existing pending flag is a
  no-op; otherwise exactly one flag is set and one reference appended. Inputs
  and the view pointer stay unchanged. `Queue` proves valid references,
  uniqueness and flags for the pending suffix. A processed reference can be
  queued again after its flag is cleared.
- `KnownComparison` includes signed exact comparison and actual `knownBoolean`
  dispatch. Unknown operators stay unknown. Represented operands cannot prune a
  feasible edge. `Successor` and `Decision` compose zero conditions, direct
  comparisons and `if` wrappers with finite numeric `assume` execution. Their
  contracts cover both retained and rejected edges.
- `Inputs` preserves other input slots. `InputJoin` executes join and equality,
  distinguishing nil (unreachable) from a reachable empty map. `StepModel` and
  `Traversal` prove the entire actual `analysis.successors` array loop: copy the
  source, refine with outcome zero as true, skip rejected edges, join, skip equal
  reachable inputs, store changed inputs and enqueue. Source contents remain
  unchanged. Prefix laws cover every finite successor list, including repeats.
- `ScalarTransfer` executes scalar `incrementValue`, `interfaceAssignment` and
  `rangeWrite`. It also executes actual `analysis.transfer` for no
  writes or one tracked signed write. It invalidates fields, evaluates the
  selected RHS, clears `Guarded`, runs the guards and stores the exact target.
  Other entries stay unchanged. `ScalarBlock` executes the complete solver-mode
  block loop (`report == nil`) for every finite operation list.
- `ValueDerivation` derives finite resolver fuel and result identity from
  `ValueEvaluation`; there is no fixed expression-depth cap. `ScalarRefinement`
  proves that numeric constraints, finite `assume` and joins preserve scalar
  identity. Non-pointer kind premises exclude missing pointer defaults.
- `ScalarBlockMeaning` composes primitive value facts and concrete environment
  updates with actual block execution in `wp_block_scalar_sound`. It proves
  scalar identity, metadata and range meaning together. `ScalarDecision` proves
  feasible-edge coverage; `InputCoverage` preserves earlier environments even
  with repeated targets. `wp_successors_scalar_sound` composes these laws with
  the actual successor loop, covering every feasible outgoing edge.

These contracts use primitive metadata and finite operation/value/branch
execution derivations. They do not store child Hoare triples, assumed child or
report soundness, or a successful report-checking premise.

## Actual post-constructor solver

`SolveSyntax` selects the real cursor initializer and lexical tail of `solve`
from the production translation. Kernel-checked shape and initializer contracts
reject a missing selection or a changed start; opaque library values are leaves
of syntax selection. No copied replacement loop is supplied.

`wp_solve_cursor` starts **after** `newGraphView`, allocation, initial input
construction and entry enqueue. It requires ownership of the snapshot, block
rows, maps, flags, queue and local locations. The actual loop reads a queue
reference, clears its flag, copies its input, calls the proved scalar block and
complete successor traversal, then advances the cursor. Block/transfer contracts
preserve the original receiver permission, including exclusive ownership. State
maps stay exclusive.

The result owns the exact final worklist, preserves scalar identities and
kind/stability metadata, and retains every environment represented in prior
inputs. Queue exhaustion implies no pending flag remains. This does not show
that all source-level reachable environments entered those inputs.

`SolveModel` contains FIFO execution evidence and its progress laws.
`reachableQueue` requires a non-nil input for every queued reference; flags alone
do not imply this. Enqueue, edge, prefix and trace laws preserve it.
`solverDataComplete_terminal` shows the data-completeness premise is satisfiable
for every admissible terminal empty-block state; it is not source correctness.

An acyclic block weight exceeds the sum of all successor weights. Processing
removes that larger weight and can append at most the successor total, so each
step strictly decreases pending weight despite repeats or skipped enqueue.
`solverExecution_exists` derives finite execution by well-founded recursion from
this budget and `SolverDataComplete`. The latter requires primitive execution
data for each pending valid scalar state with reachable inputs. Neither that
data nor the rank has yet been established from source.

`wp_solve_cursor_ranked` composes this existence result with actual execution; it
does not assume a supplied finite trace. Its explicit resource premise bounds
final queue lengths to the pinned signed cursor representation. The ranking
rules out infinite supported solver transitions. Iris WP alone is not an
allocation or official-Go total-correctness theorem.

## Open source-to-report boundary

1. Execute `newGraphView`, producer/snapshot metadata and initial maps, queue and
   entry enqueue.
2. Establish scalar write sites, finite value/branch derivations, operator
   tables, metadata agreement and complete execution data from actual admitted
   source lowering. Prove its acyclic rank and signed-word domain facts.
3. Carry full source-path concrete meaning through the constructed solver, then
   execute non-nil report mode and final `analysis.report`/`analysis.call`
   projection.

`ScalarMeaning`, `ScalarMeaningExecution`, `ScalarWriteSite` and
`metadataAgreement` name explicit primitive evidence. Constructing these records
does not prove the extractor produces it. Supported writes require zero writes
or one tracked signed write, absent step/range metadata, correct operand edges
and scalar value derivations. Non-scalar/multiple writes and arbitrary Go
features remain outside this checkpoint.

`parseInt64Contract` and `parseEmptyUint32Contract` are unproved standard-library
execution parameters. The latter says only that `strconv.ParseUint("", 10, 32)`
returns an error; the pinned translation has no proved body/specification for
it. This premise is visible in scalar range/transfer/block contracts. Parser and
compiler adequacy, pinned 64-bit Go semantics, package initialization and resource
limits remain trust boundaries. Axiom audits do not discharge theorem premises.
No `sorry`, new trusted axiom, native proof shortcut, precision weakening or
production rewrite fills these gaps.

## Independent controls and refactors

The 44 worklist faults exercise queue flags/references, signed comparisons,
operand/condition/boolean dispatch, refinement, attribute and slot ownership,
copy/join/nil/equality/store/enqueue, scalar RHS/tracking/target/guards and block
operation traversal. The six scalar-flow faults forge identities, join against
the wrong map, invert edge truth or select the wrong wrapper condition. The five
solver faults change cursor start, retain a dequeued flag, lose input, skip the
block or stop cursor progress.

Each must fail its named proof. Go overlays are independently compiled before
new-control preflight. Compiler, setup, syntax, type, ambiguous-name and unrelated
module errors do not count as logical rejection. These faults are deliberate
controls, not discovered production defects; the strict failure filter remains.

Keep external source/report tests, reusable algebra/queue/range laws, and
source-bound execution bridges separate through refactors. Regenerate and repair
the affected bridges when Go changes; CI rejects stale translations. Internal
changes need not restart the independent corpus or range algebra. Coverage
requires no architecture rewrite or state-machine dependency.
