# Arbitrary-depth branch result checking

`make verify-branches` checks a structural theorem, collects actual production
reports from supplied Go source, checks those reports with the theorem, and runs
twelve fault controls. It does not enumerate branch trees. The existing finite
program certification in [PROGRAMS.md](PROGRAMS.md) remains available.

## Exact guarantee

For every finite `Program` tree, every recorded interval, and every signed
64-bit input:

```text
checkReport(program, recordedInterval) = true
    implies recordedInterval contains Program.eval(program, input)
```

The same check also requires exact bounds on a direct literal result. Returning
unknown for every program does not satisfy the complete check. Unknown bounds
remain permitted for branches; this is not a theorem that the analyzer always
finds the tightest range.

`checkTree_sound` proves the structural part by induction on the tree. It has no
syntax-depth, finite constant alphabet, or premise that a child report is sound.
Input values and condition/literal constants are not enumerated in that theorem.
`checkReport_sound` adds signed input-domain coverage and literal precision.
`checked_interval_sound` composes the result with the existing production
interval membership contract. `certificateOfCheck` constructs the existing
`Certificate` type from a successful computation of the checked condition.

This is a theorem about a checked-result pipeline. **It is not an unconditional
correctness theorem for the production extractor or CFG worklist.** It does not
establish that every production answer will pass the check. Source lowering,
transfer/worklist composition and termination still need implementation proofs
to establish that stronger claim. The distinction is part of the result, not a
premise assumed away in the proof.

## Source-to-report connection

Use the complete verification command on another admitted source file:

```sh
python3 -B verification/lean/verify_branches.py --source /absolute/path/input.go.txt
```

The input is a canonical Go file in package `generated`, beginning with the
empty `observe(value int64)` helper. Every remaining declaration is a unique,
exported function taking `x int64` and returning `int64`. Its wrapper and body
must match the canonical branch grammar byte for byte. See
`verification/programs/testdata/branches.go.txt` for a complete input.

Leaves assign `result = x` or a signed decimal `int64` literal. Branches compare
`x` against a signed literal with `<`, `<=`, `>`, `>=`, `==`, or `!=`. Each arm
contains one leaf or another branch. Every function observes and returns its
result. Input mutation, extra statements, calls in expressions, shadowing,
noncanonical wrappers and unsupported syntax are rejected. There is no configured
enumeration depth or constant list. Go parsing, resources and Lean elaboration
still impose operational limits; setup or resource failures are not verified
results.

The connection is checked as follows:

1. The Go parser reifies each supplied function into the independent branch
   tree. The renderer must reproduce every byte of the supplied file, including
   all function declarations and wrappers. The input is not silently rewritten.
2. Real Go executions at signed endpoints and around each supplied threshold
   check the reified meaning against the compiled source. These executions are
   supporting checks, not the universal-input proof.
3. The same source bytes go through production `gopatterns.Extract` and
   `runtimeguard.Analyze`. Missing or duplicate functions/observations fail. The
   captured `FirstArgument` range is from the exact typed observation call.
4. The generated Lean certificate computes `checkReport` for the captured raw
   range and proves its source rendering equals the captured function text.
   Every checked-range and per-function source theorem has an axiom audit.
5. The driver emits a separate binding theorem from the captured manifest. Lean
   checks that each actual `Certificate` has the captured name, program tree,
   raw range and source. Numeric and Boolean fields must be literals. The driver
   audits each actual certificate, including its soundness and precision proofs,
   and the binding theorem; a valid but unrelated named theorem is insufficient.
   Source hashes, exact whole-file concatenation and before/after input
   fingerprints are also checked. `result.json` requires all checks and controls.

`go run ./verification/programs --source PATH --output DIR` only collects and
writes a certificate. It does not run Lean and must not be described as successful
verification. `--structural` selects this checker for the older generated corpus;
`--source` selects it automatically.

## Checker design and limits

The initial input domain is the full signed 64-bit range. Each true/false branch
restricts that domain independently. Mathematical integers represent endpoints,
so `MinInt64 - 1` and `MaxInt64 + 1` cannot wrap into feasible domains. A leaf
checks that its result is covered by the captured report. Both arms are checked;
empty domains are permitted only when their bounds contradict.

The `!=` true arm conservatively retains the incoming interval. This keeps the
checker simple and sound, but can reject a correct report that relies on more
precise exclusion of a point. A rejected certificate means this check did not
establish the claim; it does not by itself prove that gstyle is wrong.

The reviewed meaning of the canonical renderer, the Go parser/compiler,
collector, serialization and driver remain trusted boundaries. Independent
subject binding now rejects a certificate for an unbounded report paired with a
manifest and unused declaration claiming `[1,1]`. It also audits the actual
certificate's proof dependencies. This is a consistency check on collected
artifacts, not a sandbox for arbitrary Lean code or a proof of the collector.
This work does
not prove official Go operational semantics, arbitrary Go syntax, aliases,
concurrency, CLI finding projection or automatic rewrite correctness. The
existing signed-word interval assumptions remain unchanged. Supported source
contains no loops or side-effecting user code; resource exhaustion and package
initialization are outside the scalar meaning.

## Remaining implementation obligations

The requested unconditional source-to-range theorem is still unfinished. In
particular, a successful `checkReport` is not a substitute for these obligations:

1. Prove that the admitted, typed AST lowers to the production PDG with the same
   branch meaning. The relevant producer is `irBuilder.runtimeFlow`, including
   `runtimeFlowCFG`, `runtimeBlock`, `runtimeCondition`, operand edges and the
   scalar/control CFG builders in `adapters/gopatterns`. Trusting the Go parser
   does not establish this lowering theorem.
2. Connect the proved scalar `analysis.transfer` and solver-mode
   `analysis.block` execution contracts to the admitted source. These contracts
   now execute the actual bodies for zero writes or a tracked single signed
   write, including field invalidation, RHS evaluation and scalar helper guards.
   Establish their snapshot, reference, metadata, scalar-state and finite-value
   derivation premises from the source. The existing ParseInt and newly explicit
   empty-input ParseUint execution contracts remain standard-library boundaries. Do not assume the
   conclusion as a sound-child or sound-result premise.
3. Compose the proved `analysis.successors` execution with concrete source
   meaning, then prove `solve` preserves that invariant for the production FIFO
   queue. The successor-array prefix/metadata/queue proof is complete; it does
   not establish that its snapshot and decision data follow from the source.
   The post-constructor cursor proof now includes re-enqueue, reachable queued
   inputs, joins, queue exhaustion and a weighted acyclic termination argument.
   Source/constructor data and graph ranks still need establishment, with no
   fixed syntax-depth limit. Then connect `analysis.report`/`analysis.call` to
   the final typed observation and its `FirstArgument` range.

[Worklist proofs](WORKLIST.md) cover actual scalar block and successor execution,
concrete range/identity preservation, and the post-constructor cursor loop with
acyclic progress. Constructor/source data and rank establishment, full source-path
meaning and report execution remain open. These component proofs do not remove
this checker's success premise or prove correctness for arbitrary Go features.

## Controls and CI

Production Go overlays that forge a lower bound or erase bounds must fail the
new checker. Three isolated checker mutations must fail the general theorem:
wrong comparison inverse, using the wrong coverage endpoint, and omitting
negative inputs from the initial domain. Compiler/setup/type failures do not
count as those logical rejections. Artifact controls independently change source,
raw report fields or program inventory and must fail their named binding check.
Four shared controls keep the original generated claims compiling while changing
the manifest/report declaration together, substituting the program or source, or
replacing actual certificate soundness with an unapproved axiom. The independent
subject theorem or actual dependency audit must reject them. Setup errors do not
count as rejection.

Go tests cover independently written source with extreme and nonenumerated
constants, 32 nested comparisons, actual production collection, and rejection of
unmodeled effects and changed wrappers. Python tests reject mixed logical/setup
failures and failures in the wrong theorem. Lean audits allow only the existing
standard axioms. No `sorry`, new trusted axiom or native proof shortcut is added.

CI runs `make verify-branches` beside `make verify-programs` and retains both
artifact directories. These checks provide reusable validation for arbitrary
admitted branch trees. Removing the checker premise from the theorem remains
the next implementation-composition proof obligation.
