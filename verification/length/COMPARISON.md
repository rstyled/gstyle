# Numeric comparison transfers

The comparison proofs use the selected production `graphView.comparison` body. Operand and symbol reads use the shared PDG snapshot. Both operand values are evaluated against the input state before any constraint writes to it.

The execution contracts cover true and false branches with neither binding, a left binding, a right binding, or two valid numeric bindings. Two-sided comparisons allow the same symbol ID on both sides. They preserve the PDG snapshot and borrowed operator tables. If the second constraint rejects, the original allocated map retains the first update; rejection does not imply rollback.

`wp_comparison_both_numeric_true_sound` and `wp_comparison_both_numeric_false_sound` prove that a feasible numeric comparison returns the original non-nil map location and keeps every represented numeric valuation in the resulting intervals. They connect the actual method body to `comparisonResultMap` and its all-key preservation theorem.

Input assumptions remain explicit:

- The state map is allocated and non-nil. Owned-map premises imply this. Separate nil-input proofs cover zero operands on both truth branches; they do not assume ownership of a nil map.
- Snapshot tables satisfy `ValueSnapshotValid`, and the tracked table has the same length as the kind table. Constraint metadata either resolves the kind, unstable, and tracked entries or proves that the ID is absent and tracking is disabled. Zero and out-of-range IDs short-circuit before state access.
- Operand value and symbol queries have successful finite model results. `ParseInt` execution retains the value-stage contract premise.
- The initial binding intervals and both operand intervals represent the supplied numeric valuation. Kind and instability interpretations agree with the snapshot entries used by the method.
- The concrete values of both operands lie in the signed 64-bit range. Indexing uses the existing 64-bit Go execution model.
- Borrowed global tables supply the operators read by the method. The soundness contracts also require those selected entries to implement the proved inversion and reversal functions. Package initialization has not established these table contents yet.
- The supplied branch condition holds for the numeric valuation. Unsupported operators impose no numeric relation and remain conservative under the specified operator-table contents.
- Bound values have `Nil = false`. Pointer/nil constraints and pointer propagation remain a separate stage.

The original all-key representation wrappers cover two nonzero binding IDs. [One-sided wrappers](ONE_SIDED.md) now also connect the left-only and right-only execution contracts to all-key numeric interval soundness. The representation theorem concerns numeric interval bounds. It does not establish the program meaning of pointer flags, length/check metadata, PDG producer facts, or the Go-to-PDG mapping. It does not prove worklist convergence or universal query termination. Cached out-of-range binding IDs are covered as inactive constraints, on either or both sides.

Go regressions cover all six operator-table entries across signed endpoints, unsupported names, nil versus empty maps, both binding sides, both truth values, repeated IDs, untracked bindings, and partial state changes when the second constraint rejects. Another 9,600 concrete cases check feasible-branch interval preservation across bound presence and tracking choices, including an unrelated binding.

`make verify-comparison` audits 34 theorem dependencies and checks 40 Go/model mutations. CI partitions the controls across four shards, with two isolated workers per shard. Each shard retains the canonical translation check, dependency checks, strict proof-failure classification, private build directories, and source hashes checked before and after verification. Thirty Go mutations also have independent regression-test controls.
