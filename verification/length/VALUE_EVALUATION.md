# Production value evaluation

`wp_value_evaluation` proves the result of the selected production `graphView.value` body for a finite `ValueEvaluation` witness. It preserves the shared PDG snapshot and the complete state map. Recursive calls use induction on the witness; the theorem has no child-execution premise.

`valueResolve_evaluation` constructs that witness from each successful result of the fuel-bounded `valueResolve` model. `wp_value_resolve` connects those results to production execution. Its `ValueSnapshotValid` assumption also requires valid first-operand references for nonzero nodes.

The witness covers zero references, resolved bindings, binding IDs beyond the kind table, parentheses, constant and recursive conversions, ordinary values, and recursive length evaluation. Numeric and extent fields use signed 64-bit decimal decoding. Failed parses admit neither numeric bounds nor an extent. Integer and extent decoding are independent. Only a known callee record with the exact decoded name `len` enables length evaluation.

The generated execution model uses 64-bit Go `int`/`uint` operations for indexing and fixed-width 64-bit numeric fields. This contract does not claim 32-bit Go execution support.

Input assumptions:

- `SymbolSnapshotValid` supplies table lengths, two-element operand rows, valid symbol-recursion references, and the existing allocation bound for copied outcome rows.
- The unstable and kind tables have equal lengths.
- Snapshot ownership describes the actual graph, table, and row contents; state ownership describes the actual map.
- Each nonzero evaluated reference is in the node table. Each recursive value call has its own witness. Symbol calls have a finite `symbolResolve` witness at the supplied common fuel.
- `parseInt64Contract` specifies the actual `strconv.ParseInt` call with base 10 and width 64. This execution contract is a theorem premise, not an added axiom or a proof of the standard-library implementation.

The theorem establishes exact execution results under these assumptions. It does not establish that producer facts are true of a Go program, that snapshot construction satisfies the assumptions, or that the resulting intervals describe concrete program execution. Those need separate composition proofs.

A finite witness is required only for queried paths. Unrelated graph cycles are allowed. There is no theorem that every input has a witness, no worklist convergence proof, and no claim about stack or resource exhaustion. Invalid value references that could cause production table indexing to panic are outside this contract. Program pointer/alias semantics and package initialization remain separate stages.

The Lean model checks decimal signs, zero, leading zeros, both int64 endpoints, overflow, empty and sign-only inputs, whitespace, underscores, hexadecimal notation, and non-ASCII digits. Go regressions exercise the production constant decoder, binding precedence, and every callee status byte across exact and near-miss names.

`make verify-value-evaluation` runs the full Lean build, canonical Go translation comparison, exact audit of 49 theorems, and 37 isolated Go/model mutation controls. Logs and the source-bound result go to `dist/verification/value-evaluation/`. CI uses three shards with the existing two-worker limit. The allowed kernel axioms remain `propext`, `Classical.choice`, and `Quot.sound`; setup failures do not count as mutation rejection.
