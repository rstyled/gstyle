# Production worklist proof components

`make verify-worklist` regenerates the selected production translation, checks
source identity and pinned dependencies, audits 53 named theorems, and runs 32
fault controls. CI splits those controls into two shards. This is component
verification. The unconditional source-to-range theorem remains unfinished;
see [BRANCHES.md](BRANCHES.md).

## Execution covered

- `Layout.lean` and `FlowLayout.lean` use concrete `analysis`, `NormalFlow` and
  `FlowBlock` records. Their layouts are not replaced with opaque record axioms.
- `Enqueue.lean` executes `analysis.enqueue`. A marked reference is a no-op; an
  unmarked reference sets exactly its flag and appends exactly one reference.
  Inputs and the view pointer stay unchanged. `Queue.lean` composes this with
  valid references, uniqueness and flags for the unprocessed suffix. Processing
  clears a flag; a reference can later be queued again.
- `ExactComparison.lean`, `KnownComparison.lean` and `KnownBoolean.lean` execute
  signed comparisons, both operand/value queries, singleton guards and actual
  comparison dispatch. All signed 64-bit words are covered. Unknown operators
  remain unknown. Represented operands cannot cause a feasible edge to be pruned.
- `Successor.lean` executes zero conditions, direct comparisons and `if` wrappers.
  Its general contracts include both rejected and retained edges. Feasible-edge
  and interval-soundness contracts follow from them and the existing finite
  `assume` proof. `Decision.lean` composes these cases from snapshot and finite
  execution data; it does not assume a child Hoare triple or a sound report.
- `Inputs.lean` preserves ownership of every other slot when one input changes.
  `InputJoin.lean` executes join and equality with Go's nil/empty-map behavior.
  A nil input is unreachable; a reachable empty map is a different state. The
  joined contents preserve incoming ranges and valid metadata.
- `StepModel.lean` records exact input contents, flags and queue after each edge.
  `Traversal.lean` proves the actual complete `analysis.successors` array loop
  follows those prefixes: copy the source map, refine using outcome zero as
  true, skip rejected branches, join, skip equal reachable inputs, replace a
  changed input and call actual enqueue. The original source map is preserved.
  `wp_successors_invariant` composes that execution with valid input metadata
  and the pending-suffix queue invariant. It does not prove the outer solve loop.
- `FactSnapshot.lean` returns equality of the full expanded attributes record.
  This exposes scalar fields and row handles without asserting their source
  meaning. The existing operator/callee contract follows from that equality.

The loop proof ranges over every finite successor list with valid references.
It is not finite program enumeration. Package initialization, the pinned
64-bit Go semantics, allocation/resource limits and operational adequacy remain
explicit boundaries. Hoare execution proofs are not a new total-correctness
adequacy theorem for official Go.

## Premises still to establish

The contracts require valid immutable snapshots, valid input metadata, supported
value/assume derivations, operator tables and the existing ParseInt boundary.
Semantic refinement also requires represented concrete values and correct
snapshot meaning. These premises must follow from admitted source lowering and
the outer worklist invariant in the final theorem. They are not established by
queue flags alone.

Complete transfer/block execution, solve invariants and termination, report
projection, source lowering and producer metadata construction remain open.
Other decision forms and arbitrary Go features are outside this checkpoint.
A successful structural report check remains separate from these obligations.

## Fault controls

Controls alter queue guards, flags, references and input fields; signed versus
unsigned comparison; known flags and operand selection; boolean facts and
comparison dispatch; condition extraction, truth-test dispatch, pruning and
refinement; attribute construction; and input-slot ownership.

Traversal faults alias the source map instead of copying it, invert the edge
outcome, retain a rejected edge, join the wrong state, confuse nil with empty,
requeue an equal reachable input, store to the wrong slot, omit the input write,
or omit enqueue. Each must fail in its affected proof. Compiler, setup, syntax,
type and unrelated proof failures do not count as successful rejection.

## Refactoring boundary

No production architecture change or state-machine dependency is required for
this coverage. Keep three layers separate:

1. Source/report contracts and independent corpus checks describe externally
   visible behavior. Preserve these tests through internal refactors.
2. Interval, comparison, join, branch and queue laws describe reusable meaning.
   A change of internal structure does not by itself invalidate these laws.
3. Source-bound execution bridges connect current Go bodies to those contracts.
   Regenerate and repair the bridges whose implementation changes. CI must reject
   stale translation; bypassing source identity is not a refactoring strategy.

An implementation change can require bridge proof updates. It should not require
restarting the independent corpus, range algebra or public behavior contract.
Each checkpoint keeps the working proofs, commands, fault controls and open
boundary in the repository so work can stop and resume without an architecture
rewrite.
