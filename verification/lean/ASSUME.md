# Finite assume execution

`make verify-assume` proves supported production `graphView.assume` executions from finite certificates. `FiniteAssume` has a natural-number depth. Parentheses, negation, and both logical children use a smaller depth. A zero-depth certificate is impossible. A parentheses self-cycle without a contradictory Boolean fact has no certificate; it is not classified as an infeasible branch.

The certificates cover Boolean contradiction, parentheses, negation, `&&`, `||`, and numeric comparisons with a nonzero symbol on the left, right, or both sides, plus the unchanged-state path when neither operand has a symbol. This includes admitted Boolean literals, whose fallback comparisons have no symbols. Comparison evidence supplies finite successful value and symbol queries and the existing numeric constraints. It supplies no child Hoare contracts. `wp_finite_assume` derives the recursive execution contracts by induction and proves the translated production body returns the specified result with the shared PDG snapshot and operator tables intact. Logical alternatives copy their input before execution; their join has the exact existing lookup semantics. `none` denotes an infeasible branch, never missing evidence. Absence of a certificate makes no claim about runtime reachability.

`constraint_state_valid`, `comparison_result_valid`, and `finiteAssume_valid` prove metadata validity for every surviving state. `finiteAssume_sound` and `wp_finite_assume_sound` show that every represented numeric environment satisfying the selected condition remains represented in the output. Such an environment cannot produce an infeasible output. These proofs cover inactive symbols, unknown bounds, negative values, and signed endpoints through the existing comparison contracts. The all-key property includes keys not constrained by the expression.

## Preconditions

- Input state maps are allocated, non-nil, and exclusively owned. Every cached key has a positive, in-range reference and agrees with the snapshot's kind and instability metadata. The snapshot has valid operand rows and the existing value-query validity conditions.
- Numeric comparison bounds are non-nil when a symbol is constrained. No-symbol comparisons leave both values unconstrained. Both-symbol comparisons use values read before the first update. Aliased symbol IDs are allowed; heap alias reasoning is not added.
- Value and symbol query results agree with the supplied numeric environment. Boolean facts, unary and logical operators, and comparison conditions satisfy `AssumeMeaning`. These are explicit source-program meaning assumptions, not proofs of the PDG producer.
- Environment values fit signed 64-bit integers. The existing Go semantics, layout, helper contracts, parser contract, and package-initialization assumptions hold. Inverse and reverse table contents have the stated canonical meaning.
- Each supported expression has a finite certificate. Comparison evidence is supplied for every valid intermediate state on which that leaf can run. This is a sufficient supported subset; automatic certificate synthesis is not supplied.

## Limits

The depth proves finite recursive expansion of the supported `assume` paths and the execution theorem binds that expansion to the production dispatcher. The inherited separation-logic WP is a partial-correctness contract; this stage does not add a Go operational total-correctness adequacy theorem or prove termination of all called query implementations. It does not prove arbitrary cyclic graphs terminate.

Pointer/nil constraints, general nil input states, and 32-bit execution remain unsupported by this certificate. Existing production tests for some of these cases remain enabled. Full transfer/CFG integration, producer and parser implementation truth, pointer/alias semantics, package initialization, and convergence remain separate work.

## Verification

The profile audits twenty named theorems and rejects ten mutations of the actual Go dispatcher. Controls cover bypassed or incorrect Boolean guards, wrong unary operands or truth values, missing disjunction dispatch, and inverted logical or comparison truth. Rejection must be a logical failure in the designated `AssumeDispatch.lean` or existing `Boolean.lean` module; setup and type errors do not count. All existing comparison, branch, and metadata controls remain in CI.

Go regressions compose parentheses, negation, conjunction, disjunction, and numeric bounds. They check represented values across signed endpoints, unknown and one-sided bounds, unrelated state records, fork input preservation, and Boolean rejection before a self-reference. No production behavior changes or demonstrated production bug fixes are part of this stage.

CI uses the configured mutation shards. Pinned dependencies, canonical translation comparisons for all four packages, exact axiom audits, before/after source hashes, private build copies, strict error classification, and the two-worker verifier limit remain enabled.

## Dispatch contracts and mutation cost

`AssumeDispatch.lean` proves the production `assume` body routes parentheses,
negation, logical operators, and comparisons to the specified child call with
its original reference, operator, truth value, and state. Each routing lemma
accepts a Hoare contract for that next call and preserves its postcondition.
`wp_finite_assume` supplies these contracts from its induction hypothesis or
from the existing logical and numeric execution proofs. Its statement and the
statement of `wp_finite_assume_sound` remain unchanged. The routing contracts
add no axioms and do not replace the concrete child execution proofs.

The seven routing mutation controls target this smaller execution module.
They still regenerate the complete production translation and must fail with
recognized proof diagnostics in `AssumeDispatch.lean`. The three boolean
controls retain their existing `Boolean.lean` target. The ordinary baseline
build and axiom audit still check the full composed theorem. This prevents a
mutation to dispatch alone from repeatedly rebuilding unchanged value parsing
and numeric execution proofs. All ten mutation controls remain enabled.

`validSymbolRef` lives with the snapshot definition. Snapshot ownership and
fact reading do not import recursive symbol execution proofs; modules that
need symbol execution import them explicitly. The reference predicate retains
its original definition.

### Index finding policy

The typed-source adapter requires a recorded upper bound below zero for
`negative_index`, or a recorded range entirely below zero or at/above the literal
extent for `literal_index`. Unknown and partly valid ranges stay quiet. Source
extraction and this stricter finding policy are outside Lean. The existing
interval kernel and its proofs still describe the raw missing-lower-bound query;
that query alone no longer produces a source finding. This trades recall for
fewer unsupported findings and adds no reachability proof for concrete inputs.
