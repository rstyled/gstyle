# Mutation assurance: bounded experiment plan

Registered before implementing or running the new oracle, 2026-10-22.
Frozen development analyzer and policy: `98ad52cb09ef1a38090edd9cad9c1248a36cca2c`.

## Claim and observation boundary

In the declared corpus, each executed write to an integer cell reachable from
the caller at function entry must have a mutation effect or explicit unknown
effect at the enclosing case's write/call site. An unrelated IO, mutation, or
unknown effect does not discharge this obligation. Unknown is reported
separately from a classified mutation. Density and diagnostic thresholds are
outside this claim.

The runtime observer identifies caller cells by actual `*int` identity. It
instruments addressable integer assignments and increments before execution;
it does not read Gstyle facts or infer ownership. Places must have no calls or
other evaluation effects. Original line numbers are retained. Successful
execution is required; compile failures, timeouts, and oracle errors are not
semantic passes. Record transient and same-value writes, not just final state.
Analyze the uninstrumented source. Check original versus instrumented final
state as an additional instrumentation control, not as proof of equivalence.

This claim excludes maps, concurrency, unsafe, assembly, publication of new
reachable cells after entry, arbitrary destination graphs, and library-internal
writes. It covers one integer write per statement/site and single-target
helper calls. It is not a proof of all Go ownership or side-effect analysis.

## Fixed matrix and budget

Twelve development families: borrowed pointer, private pointer, borrowed slice,
cloned scalar slice, fresh container/shared element, copied container/shared
element, mixed rebind, indirect rebind, borrowed helper return, private helper
return, inline callback, and write-then-restore with unrelated IO.

Four holdout families, specified before results and excluded from oracle tuning:
reverse two caller slice cells (standard slice-reversal idiom), swap caller
fields (ordinary struct update), update through a saved alias after detaching
the field, and a helper that writes and restores the same caller cell. They
are independently written idioms outside the repaired self-scan cases, not a
statistical sample of external repositories. Report them separately. If a
holdout fails, retain that first result and stop treating it as held out.

Cross each family with the existing six syntax transformations and three
initial integers (-1, 0, 7): 96 functions, 288 original executions and 288 traced
executions. Verify syntax equivalence with the existing AST guard. Add a
same-value write control. Run the existing 126-case corpus as a regression.
No open-ended fuzzing or corpus expansion in this assessment.

Three unsafe analyzer builds, using Go overlays rather than editing production:
discard external parameter provenance; treat referenced local elements as
owned; and discard mixed-binding uncertainty. Require successful compilation
and the same runtime corpus, followed by an observed semantic mismatch. Record
surviving mutants honestly; setup failure does not kill a mutant.

## Evidence and completion

Keep original and traced inputs, runtime witnesses/final states, complete frozen
JSON reports and facts, toolchain/configuration, source/binary hashes, and each
mutant's exact overlay and result. Evaluate frozen binaries with their own
classification policy. Report per-case matched mutation, covered uncertainty,
missing evidence, and private-storage false effects; do not call these population
precision or recall. The frozen quality assessment labels findings from a
development corpus and does not measure unreported defects. Existing Lean
theorems have their documented source bindings and scope; they do not prove
this ownership contract end to end.

Stop after the fixed matrix, controls, three mutants, reduction of any confirmed
counterexample, and required repository checks. Fix only confirmed defects with
focused regression tests. Report remaining scope and verification gaps.
