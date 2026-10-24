# Program-level range certificates

Run `make verify-programs` with Go and the pinned Lean toolchain available on
`PATH`. The command collects real analyzer results, proves their numeric meaning,
and checks eight negative controls. Evidence stays in
`dist/verification/programs/`; a failed run removes any earlier `result.json`.

## Claim

For every admitted program in each of the two finite grammars, the captured
production result interval contains the program's result for **every signed
64-bit input**. Input values are universally quantified in Lean; they are not a
sample, a small-integer bound, or a list of fuzz seeds.

The source functions take one `int64` parameter `x`, choose a result through a
finite tree of comparisons against constants, call an empty `observe(result)`
function, and return that same result. Leaves select `x` or an integer literal.
The report is the `FirstArgument` range at that exact, typed observation call.
Missing functions, missing observations, duplicate functions, and multiple
observations fail collection. The collection path uses production
`gopatterns.Extract` and `runtimeguard.Analyze`, not manufactured PDG fixtures or a
replacement implementation of the analyzer.

Two alphabets are enumerated completely:

- **Boundaries:** depth at most one; six comparison operators; thresholds
  `MinInt64`, -1, 0, 1, and `MaxInt64`; leaves `x`, -1, 0, and 1. There are 484 cases.
- **Nested:** depth at most two; comparisons `x < 0`, `x == 0`, and `x > 0`;
  leaves `x` and 0. There are 590 cases. Both children may branch, so this includes
  trees with three conditionals, contradictory paths, and repeated tests.

The two grammars overlap; 1,074 is the total number of collected functions.
Depth bounds constrain program syntax, never the input-value quantifier.

## Proof connection

`Gstyle/Proof/Programs/Contract.lean` defines the expression and program meaning
independently of the analyzer. It does not call `valueResolve`, follow gstyle's
CFG, or compute expected bounds using the analyzer's transfers.

- `enumerate_complete` proves enumeration equals structural admission for any
  alphabets and depth. Admission does not depend on a successful analysis.
- Each generated `soundCase...` theorem proves the captured range contains
  `Program.eval program x` for every signed 64-bit `x`. Standard Lean arithmetic
  tactics construct proofs checked by the kernel.
- `PreciseLiteral` requires exact bounds on direct literal results. An analyzer
  returning unknown for every case cannot pass the complete contract.
- Each certificate proves its rendered Go function equals the captured source.
  `source_file` checks the ordered source segments, including the helper. The
  driver checks that their exact concatenation is the analyzed file, and that
  certificate bounds equal the captured raw reports. Coverage equality checks
  the full program inventory, not merely its size.
- A separate driver-emitted theorem binds each actual certificate's name,
  program tree, raw range and source to manifest literals. The actual certificate
  and this theorem receive axiom audits, including soundness and precision fields;
  unused report declarations and unrelated named proofs cannot replace them.
- `certified_admitted` composes these certificates into the program-level claim.
- `reported_interval` proves that certificate membership is exactly the interval
  membership used by the existing production interval proofs, under signed-word
  bounds. Raw unknown-bound payloads keep their existing meaning.

This is **result certification**: the analyzer is not assumed correct. Its actual
answers become proof obligations. Existing internal proofs remain useful without
being premises that assume the whole analysis is sound.

## Trust and limits

The proof quantifies the independent scalar grammar's meaning. The equality
checks bind that grammar's renderer to the analyzed bytes. They do not prove a
refinement theorem from the official Go operational semantics to `Program.eval`.
The small renderer and its Go meaning remain reviewed and tested boundaries.
The harness compiles and executes every generated function at signed endpoints
and around every threshold, checking those real Go results against its scalar
interpreter. These executions check the boundary; the universal input guarantee
comes from Lean, not those executions.

A harmless `_ = *new(int)` in each wrapper selects the production runtime-fact
extraction path. It cannot change the numeric result on normal execution. Memory
allocation failure and resource exhaustion are outside the scalar meaning.
`observe` is defined in the captured file with an empty body.

The collector, raw-report serialization, command runner, Go compiler/toolchain,
pinned Lean dependencies, and Lean kernel remain trusted. Source hashes are
checked before and after the run. No unproved axioms, `sorry`, native decision
procedures, or accepted setup failures are added.

This does not prove the analyzer implementation correct for arbitrary program
sizes or other Go syntax. It does not prove convergence, operational termination,
CLI finding projection, pointer/alias semantics, arbitrary assignments, loops,
concurrency, or automatic rewrites. It closes a bounded source-to-range evidence
gap without claiming those larger results.

## Negative controls

The verifier builds two isolated Go overlays of the production observation code.
One forges lower bound 1; the other erases all bounds. The first must fail numeric
soundness, and the second must fail the literal precision contract. The originals
are never edited. Two certificate controls omit a program or change the captured
source. They must fail coverage and source equality respectively.
Four more controls preserve compilation of the original generated claims but
substitute the report, program, or source, or put an unapproved soundness axiom in
the actual certificate. Independent subject binding and dependency audits reject
these. This checks artifact consistency; it does not prove collection correct or
sandbox arbitrary Lean code.

A negative result counts only for an expected logical diagnostic in the named
contract. Compiler failures, missing dependencies, syntax/type errors, and mixed
setup/proof failures abort verification. A small unchanged control corpus passes
before the certificate mutations run. All logs and overlay sources survive for
review. Pinned dependency revisions and five foundation theorem axiom lists are
also checked; complete generated claims and source identity have their own audits.

## Proof map

The user promise is that reported ranges include every possible result.
`verification/programs/collect.go` binds that promise to production extraction
and the typed `observe` call in `domain/runtimeguard/flow.go`. Generated
`complete0` and `complete1` discharge the claim over the two admitted grammars.

Existing `Interval`, `ValueEvaluation`, `Comparison`, and `Assume` proofs cover
internal execution and preservation contracts. This certificate layer checks the
composed producer, transfer, traversal, and observation result on whole source
programs. A future implementation-level composition proof can use those internal
contracts and use this corpus as an independent check. The two remaining global
bridges are official Go source meaning and an implementation theorem for all
admitted program sizes; neither is hidden as a discharged assumption here.
