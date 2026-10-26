# Production worklist proofs

These are source-bound component proofs. The unconditional source-to-report
implementation theorem remains **open**. The separate
[structural report checker](PROGRAMS.md#arbitrary-depth-checker) requires checker
success; that premise is not an implementation correctness proof.

## Run and audit

| Command | Named axiom audits | Fault controls | Scope |
| --- | ---: | ---: | --- |
| `make verify-worklist` | 61 | 44 | Queue, decisions, scalar transfer, block and successor execution |
| `make verify-scalar-flow` | 37 | 6 | Scalar identity, metadata and concrete range meaning |
| `make verify-solver` | 35 | 5 | Actual post-constructor cursor loop and acyclic progress |
| `make verify-setup` | 55 | 10 | Actual scalar constructor and initial solver state |

Each profile checks pinned dependencies, regenerates and compares the selected
production translation, binds its sources and audits the named theorems. CI
splits the worklist controls into two shards. A fresh combined run can execute
all 55 controls and audit all 133 names; collecting old success reports is not a
combined execution. `make verify-branches` separately checks supplied-source
report integration and its twelve controls.

Seven unused convenience wrappers are removed. Their original statements
reconstruct without new premises: exact comparison execution plus
`knownComparisonResult_no_false_rejection`; successor execution plus
`successorResult_feasible`/`finiteAssume_sound`; and `wp_decision` plus
`DecisionEvidence.scalar`/`sound`. Syntax-selection and positive formation checks
remain. Audit-name counts are an inventory, not a coverage measure.

## Coverage and premise producers

**Conditional** means Lean proves the linked statement under its full premises;
the table names the main premises, not every parameter. **Tested** means finite
concrete executions. **Assumed** marks an external contract or execution boundary
not proved here. **Open** means the required implementation connection is missing.
All execution rows also use the pinned heap/layout/Go model and package-initialization
premises (A1). An axiom audit does not establish these premises.

| ID / guarantee | Contract and actual code | Required premises and producer | Status | Representative control or completion check |
| --- | --- | --- | --- | --- |
| C0: snapshot reads preserve the full attributes record | [`wp_attrs_snapshot_full`](Gstyle/Proof/Transfer/FactSnapshot.lean#L127); [`graphView.attrs`](../../domain/runtimeguard/graph.go#L52) | Valid snapshot and read-only table ownership; O1/O2 must establish the representation and its source meaning. This getter proof establishes no source meaning. | Conditional | `snapshot-wrong-construct` |
| C1: enqueue preserves the pending queue | [`wp_enqueue_invariant`](Gstyle/Proof/Worklist/Queue.lean#L126); [`analysis.enqueue`](../../domain/runtimeguard/flow.go#L16) | Valid references, owned arrays and existing queue invariant; initial establishment is O1. | Conditional | `enqueue-duplicates-reference`, `queue-retains-processed-mark` |
| C2: supported decisions retain feasible edges | [`wp_decision`](Gstyle/Proof/Worklist/Decision.lean#L117), [`DecisionEvidence.sound`](Gstyle/Proof/Worklist/ScalarDecision.lean); [`knownComparison` / `successorState`](../../domain/runtimeguard/known_boolean.go) | Valid snapshot, operand meaning, operator tables and finite assume evidence; source producer is O2. | Conditional | `known-wrong-right-operand`, `successor-inverts-pruning` |
| C3: scalar transfer and the whole nil-report block execute as specified | [`wp_transfer_scalar_step` / `wp_block_scalar`](Gstyle/Proof/Worklist/ScalarBlock.lean#L134); [`analysis.transfer` / `block`](../../domain/runtimeguard/flow.go#L41) | `ScalarExecution`, supported zero/single signed writes, scalar state and snapshot validity; O1/O2 must establish them. | Conditional | `scalar-wrong-symbol`, `block-drops-transfer` |
| C4: successor traversal preserves exact inputs and queue invariants | [`wp_successors_invariant`](Gstyle/Proof/Worklist/Traversal.lean#L320); [`analysis.successors`](../../domain/runtimeguard/flow.go#L24) | `DecisionEvidence`, valid input states, successor bounds and queue ownership; O1/O2 supply initial data. | Conditional | `traversal-confuses-nil-empty`, `traversal-wrong-input-slot` |
| C5: blocks and feasible edges preserve concrete range meaning | [`wp_block_scalar_sound`](Gstyle/Proof/Worklist/ScalarBlockMeaning.lean#L105), [`wp_successors_scalar_sound`](Gstyle/Proof/Worklist/ScalarTraversal.lean#L65); same block/successor methods | `ScalarMeaningExecution`, `ScalarMeaning`, `metadataAgreement`, represented input environments and signed-word facts; source establishment remains O2. `valueIntervalMeaning` already requires resolver range meaning. | Conditional | `scalar-flow-join-left`, `scalar-flow-outcome` |
| C6: the actual post-setup cursor loop preserves invariants and has finite execution | [`wp_solve_cursor_ranked`](Gstyle/Proof/Worklist/SolveLoop.lean#L268); cursor loop in [`solve`](../../domain/runtimeguard/flow.go#L185) | Owned setup (O1), `SolverDataComplete`, decreasing weights, metadata and resource bound (O2/A1). Preserves prior represented inputs; full source-path meaning remains O3. | Conditional | `solver-cursor-start`, `solver-skips-block`; the terminal-empty-block witness does not establish general source data. |
| C7: an accepted report contains every grammar result for every signed 64-bit input; direct literals retain exact bounds | [`checkReport_sound`](Gstyle/Proof/Programs/Structural.lean#L98), actual-certificate binding in [`verify_programs.py`](verify_programs.py); captured [`analysis.call`](../../domain/runtimeguard/flow.go#L127) ranges via [`collect`](../programs/collect.go) | Checker acceptance, captured subject equality and trusted source/collector meaning (A1). Does not establish unconditional solver/report correctness. | Conditional | `wrong-endpoint`, `erased-bounds`, `substituted-report`, `certificate-axiom` |
| T1: compiled source agrees with the grammar on sampled executions | [`checkRuntime` / `compareRuntime`](../programs/runtime.go) | Go compiler/runtime and the supplied finite corpus; endpoints and threshold neighbors are sampled. | Tested | Executed Go results must equal independent grammar results; this is not the all-input Lean proof in C7. |
| A1: external execution contracts and interpretation | [`parseInt64Contract`](Gstyle/Proof/Transfer/ParseIntContract.lean#L20), [`parseEmptyUint32Contract`](Gstyle/Proof/Worklist/ScalarTransfer.lean#L41); pinned Goose/Perennial, parser/compiler, source renderer and collector | Callers supply the stdlib execution contracts, heap/layout/package-init and resource premises. Tool/model adequacy and renderer meaning remain external boundaries. | Assumed | Dependency/source/axiom audits and T1 check evidence; they do not prove these contracts or official Go adequacy. |
| O1: setup establishes the solver's initial invariant and ownership | [`wp_newGraphView_scalar`](Gstyle/Proof/Worklist/Constructor.lean), [`wp_solve_setup_entry`](Gstyle/Proof/Worklist/SetupEntry.lean); actual `newGraphView` and setup in `solve` | Represented raw graph, scalar structural restrictions, package data and allocation bounds; see [SETUP.md](SETUP.md). No constructor contract, initial snapshot or invariant premise. | Conditional, scalar subset | Actual construction derives snapshot validity, exclusive empty roots, exact initial containers, scalar invariant and reachable queue. Source production of admitted data remains O2; wider constructor cases remain outside this theorem. |
| O2: admitted source establishes primitive meaning and solver data | [`runtimeFlow` / `runtimeFlowCFG` / `runtimeBlock` / `runtimeCondition`](../../adapters/gopatterns/ir_runtime_flow.go) and graph metadata construction | Typed source lowering must supply write sites, metadata, value/branch meaning, signed-word facts, `SolverDataComplete` and acyclic weights used by C2–C6. | Open | Complete when those facts follow for every admitted finite branch tree within the stated machine/resource bounds, without taking their correctness as input. No current source theorem or fault control closes this bridge. |
| O3: source paths compose through solve and actual report projection | [`analysis.report` / non-nil `block` / `call`](../../domain/runtimeguard/flow.go#L74), composed with O1/O2 | Source meaning through the constructed solver, not only prior-input preservation from C6. | Open | Complete when actual execution yields a raw report containing every admitted source result, without assuming checker acceptance or child/report soundness. |

Control names are examples from [the registered profiles](verify.py),
[branch controls](verify_branches.py) and [certificate controls](verify_programs.py).
They test failure detection, not premise establishment or coverage completeness.
The O1 scalar theorem executes the actual setup; O2 and O3 remain continuation
obligations. Creating an evidence record does not discharge them. The sections
below give the supported cases and remaining limits.

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

1. The represented scalar setup now executes `newGraphView`, metadata construction,
   initial maps, queue and entry enqueue in [SETUP.md](SETUP.md). Source production
   of these admitted inputs and wider constructor features remain separate work.
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
