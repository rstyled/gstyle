# Bounded caller-mutation assurance

This assessment tests one contract: within the declared corpus, each executed
write to an initial caller-owned `int` cell must retain mutation or explicit
unknown evidence at the relevant source site. Unrelated effects do not count.
It evaluates semantic effect evidence, not warning thresholds or architecture.

The [experiment plan](plan.md) was committed before implementation. The frozen
analyzer and its own default policy come from
`98ad52cb09ef1a38090edd9cad9c1248a36cca2c` (#238), using Go 1.27.1, linux/amd64.
The final branch also includes #239; the normal tests check that combined code.
No production ownership or classification rule changes in this assessment.
The combined candidate's frozen binary repeats the same 201 classified, 102
unknown, zero missing witnesses; its binary hash and raw evidence are retained.

## Results

| Run | Executions | Classified write witnesses | Covered by unknown | Missing witnesses |
| --- | ---: | ---: | ---: | ---: |
| Frozen baseline | 291 | 201 | 102 | 0 |
| External mutations ignored | 291 | 0 | 102 | 201 |
| Local elements wrongly owned | 291 | 201 | 66 | 36 |
| Mixed bindings wrongly owned | 291 | 201 | 90 | 12 |

There are 303 executed write witnesses in the baseline. These repeat across
syntax and input variants; they are not 303 independent samples. The four
holdout families account for 72 executions and 126 classified writes, with no
missing or unknown witnesses. They are predeclared hand-written idioms, not a
blind external-repository sample or an estimate of population precision/recall.
All three semantic mutants compile and fail through the same runtime oracle.
Declaration-reduced witnesses of 171, 184, and 219 bytes reproduce the mismatch
and preserve final values and the ordered observed targets. Reduction removes
unreachable helpers; it does not claim globally minimal source.

The observer executes the original and traced programs separately (582 baseline
executions total), compares their final cell values and slice lengths, and checks
the expected write counts. Pointer identity decides which cells belong to the
caller. Source instrumentation records writes before execution, preserving line
numbers. Successful execution and restricted, effect-free places are required.
Write-then-restore and same-value assignments are therefore visible even when
final-state snapshots show no change. Analyzer input excludes the trace code.

The first parameter-provenance mutant compiled but failed saved-fact validation
with `invalid call storage ...helperChain`. It is a separate boundary rejection,
not a semantic mutant kill. Its logs/overlay are retained. The documented
replacement policy mutant supplies valid facts and proves that the frozen
binary's own policy is used. An early reduction script rejected a null write
list; that setup error was fixed and the full experiment rerun successfully.

## Confirmed harness defects and corrections

1. `hiddenWrite` previously accepted a complete IO-only report for a program
   that changed caller state. The regression fails before the change. The legacy
   check now requires mutation evidence; the new oracle further requires the
   correct source site and reports uncertainty separately.
2. `--gstyle` previously combined the frozen extractor with the current policy.
   The harness now asks that binary to evaluate its saved facts. Valid finding
   exit 1 is accepted; analysis exit 2 and invalid report schemas are rejected.
   An independent CLI fixture checks that current policy cannot replace it.

The runtime study finds no hidden-write defect in the frozen analyzer within
these bounds. It does expose limited precision: fresh-container/shared-element,
mixed/indirect rebind, borrowed helper-return and inline-callback families often
retain unknown. The private helper-return family has no observed caller writes,
yet remains unknown in all 18 executions. This is a conservative coverage gap,
not a claim of incorrectly classified external mutation. The mixed-rebind case's
effects on a non-writing input are valid static overapproximation because other
inputs do write. No additional analyzer rule is widened to improve these counts.

## Reproduce and inspect

From the repository root, use a new output directory:

```sh
GOTOOLCHAIN=go1.27.1 python3 verification/quality/mutation-assurance/run.py \
  --output "$HOME/.cache/gstyle-mutation-assurance-new"
```

The script archives the pinned source without changing the checkout, builds the
baseline and three Go-overlay mutants, demonstrates the old harness failure,
checks the 126-case legacy corpus and oracle/policy controls, then executes the
fixed matrix and rechecks reduced mutant witnesses. Compilation/analysis errors
are errors, never semantic passes. All builds receive the same original and
instrumented inputs. No dependency is added.

`manifest.json` records toolchain/environment, checker and input hashes, binary
hashes, and results by family. Each run keeps original/traced source, runtime
witnesses and final values, raw facts, the full frozen-policy report, the exact
overlay, and logs. `assessment.json` beside this README summarizes the recorded
run; `evidence.tar.gz` preserves its raw evidence and the initial boundary
rejection. Extract the archive into a new directory with `tar -xzf` to inspect it.
Binaries and copied repository/build caches stay outside the archive; the script
rebuilds them. Absolute paths in historical overlays identify the original run;
the script writes fresh paths when reproducing it.

Normal CI runs the fixed current-code matrix and independent controls through
`go test ./examples/quality-hunt`. The explicit frozen/mutant experiment is a
separate bounded command, not an endless CI fuzzer.

Validation passes: `make check`, `make quality-gate`, `make lint`,
`make cognitive gstyle-check`, and the quality-hunt race tests. The baseline is
regenerated; fixture goldens are unchanged. The new saved-fact IO wrapper retains
one density finding (727 > 500), including conservative serialization evidence.
It is an explicit IO boundary. No suppression or metric-only split hides that
tradeoff; the required quality gate passes on the combined code.

## Claim, evidence, and remaining limits

| Claim | Evidence | Limit |
| --- | --- | --- |
| Caller writes are represented | Runtime address witnesses matched by source site | Only the finite declared integer-cell programs |
| Other effects cannot mask a missing write | IO/wrong-site mutation/unknown controls and policy mutant | One integer target per site; single-target helper calls |
| Transient writes are observed | Restoring and same-value cases, expected event counts | Not a general heap or machine-store tracer |
| Equivalent syntax retains evidence | Existing AST guard and normalized fact/report signatures | Six declared transformations only |
| Unsafe widening is detected | Three successfully built semantic mutants plus reduced rechecks | Selected faults, not all possible analyzer defects |
| Precision remains bounded | Explicit unknown and non-writing-case results | No population precision/recall claim |

The observer, matcher, Go compiler/runtime, and generator remain trusted code.
They are checked by independent controls, not formally proved. Newly published
cells, maps, arbitrary alias graphs, multi-target helper effects, effectful
places, concurrency, unsafe, library-internal writes, and IO absence are outside
this contract. A passing run cannot establish general soundness or absence of
all side effects.

The older `verification/quality/assessment.json` pins and labels a development
self-scan (665 findings; 50 reviewed in its first batch). It does not label
unreported functions and cannot measure missed effects. Goldens and fact replay
check consistency, not independent semantic truth. The relevant formal-proof
audit is kept in [verification/lean](../../lean/MUTATION_ASSURANCE.md).

Stop here: the deliverable is this finite assessment and stronger reusable
checks. A useful next study would target one unknown transfer boundary with the
same independent contract; warning count is not its acceptance criterion.
