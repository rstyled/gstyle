# Program range verification

These commands certify captured production reports. They do **not** prove the
analyzer implementation correct without a successful certificate check. See
[WORKLIST.md](WORKLIST.md) for implementation coverage, premises and open work.
Use Go and the pinned Lean toolchain on `PATH`.

| Command | Programs | Fault controls | Evidence |
| --- | --- | ---: | --- |
| `make verify-programs` | Exhaustive bounded grammars below | 8 | `dist/verification/programs/` |
| `make verify-branches` | Supplied canonical source; arbitrary finite branch trees | 12 | `dist/verification/branches/` |

A failed run removes any earlier `result.json`. Collection alone is not
verification: `go run ./verification/programs --source PATH --output DIR` writes
artifacts but does not run Lean. `--source` selects structural checking;
`--structural` also selects it for the generated corpus.

## Common guarantee and source grammar

For each certified program, its captured `FirstArgument` interval contains
`Program.eval program x` for **every signed 64-bit input** `x`. Input values are
universally quantified, not sampled or bounded to small integers. Direct literal
results must have exact bounds; an analyzer returning unknown for every program
cannot pass. Branch reports need not be the tightest possible intervals.

`Gstyle/Proof/Programs/Contract.lean` defines the independent grammar and meaning.
It does not call the analyzer's value resolver or follow its CFG. Functions take
one `int64` parameter `x`, choose a result through comparisons against signed
constants, call the empty `observe(result)` helper, and return the same result.
Leaves select `x` or an integer literal; comparisons use `<`, `<=`, `>`, `>=`,
`==` or `!=`.

Supplied source must be a canonical file in package `generated`, starting with
`observe(value int64)`. Remaining declarations are unique exported functions with
that wrapper and grammar, reproduced byte for byte by the renderer. Leaves
assign `result = x` or a signed decimal literal. Each branch arm contains one
leaf or another branch. Input mutation, extra statements, expression calls,
shadowing, changed wrappers and unsupported syntax are rejected. See
[`branches.go.txt`](../programs/testdata/branches.go.txt) for a complete input.

```sh
python3 -B verification/lean/verify_branches.py --source /absolute/path/input.go.txt
```

The wrapper's `_ = *new(int)` selects production runtime-fact extraction; it
cannot change the numeric result on normal execution. Allocation failure and
resource exhaustion are outside the scalar meaning.

## Exhaustive bounded programs

`make verify-programs` collects these complete alphabets:

- **Boundaries:** depth at most one; six comparison operators; thresholds
  `MinInt64`, -1, 0, 1 and `MaxInt64`; leaves `x`, -1, 0 and 1: 484 cases.
- **Nested:** depth at most two; `x < 0`, `x == 0` and `x > 0`; leaves `x` and 0:
  590 cases. Both children may branch, including contradictory and repeated tests.

The grammars overlap; 1,074 is the total number of collected functions. Syntax
bounds do not restrict the universal input quantifier.

`enumerate_complete` proves enumeration equals structural admission for any
alphabets and depth, independent of analysis success. Each generated
`soundCase...` proves the captured interval contains the program result for every
signed input. `PreciseLiteral` enforces literal precision. `complete0`,
`complete1` and `certified_admitted` compose those certificates over the admitted
grammars. This certifies collected answers, not arbitrary program sizes.

## Arbitrary-depth checker

`make verify-branches` does not enumerate trees. `Structural.lean` proves:

```text
For every finite Program tree, recorded interval and signed 64-bit input:
  checkReport(program, recordedInterval) = true
    implies recordedInterval contains Program.eval(program, input)
```

`checkTree_sound` uses structural induction, without a syntax-depth bound,
constant alphabet or assumed child-report soundness. `checkReport_sound` adds
signed input coverage and literal precision. `checked_interval_sound` connects
this to production interval membership; `certificateOfCheck` builds a
`Certificate` from a successful check. This does not establish that every
production answer passes.

The checker starts with the full signed 64-bit domain and refines true and false
arms separately. Mathematical endpoints prevent `MinInt64 - 1` or `MaxInt64 + 1`
from wrapping into feasible domains. Both arms are checked; an empty domain
requires contradictory bounds. The `!=` true arm keeps the incoming interval,
so a correct report relying on point exclusion can be rejected. Rejection means
this check did not establish the claim, not necessarily that gstyle is wrong.
There is no configured tree-depth or constant-list limit, but parsing, memory
and Lean elaboration still impose operational limits. Setup/resource failure is
not a verified result.

## Shared source and certificate binding

1. The Go parser reifies functions into independent program trees. The renderer
   must reproduce every source byte, wrapper and declaration; input is not
   silently rewritten. Coverage compares the full inventory, not just its size.
2. Compiled Go executions at signed endpoints and threshold neighbors must equal
   the independent interpreter. These samples test the source-meaning boundary;
   Lean supplies the all-input guarantee.
3. The same bytes pass through production `gopatterns.Extract` and
   `runtimeguard.Analyze`. The collector reads the range at the exact typed
   observation call. Missing/duplicate functions, missing observations or
   multiple observations fail collection; manufactured PDG fixtures are not used.
4. Each certificate proves its rendering equals captured function text.
   `source_file` checks ordered segments, including the helper. The driver
   checks their exact concatenation, raw reports, source hashes and before/after
   input fingerprints. `reported_interval` connects certificate membership to
   production intervals under signed-word bounds, preserving unknown payloads.
5. The driver requires a typed `Certificate` subject, then emits a theorem binding
   its name, program, raw range and source to captured manifest literals.
   Numeric and Boolean fields must be literals. Kernel-checked reflexivity
   proves these equalities without recomputing decidable tree/string equality.
   The theorem, typed subject and original certificate receive axiom audits,
   including soundness and precision fields. Unused declarations or unrelated named
   theorems cannot stand in for the certificate. `result.json` requires every
   applicable check and control.

## Controls and CI

Both commands use isolated production overlays that forge lower bound 1 or erase
bounds. The first must fail soundness; the second must fail literal precision.
Artifact controls omit a program or change captured source. Four shared controls
keep the original generated claims compiling while substituting report, program
or source, or adding an unapproved axiom to actual certificate soundness. Subject
binding or dependency audits must reject them. A small unchanged corpus passes
before certificate mutations run; logs and overlay sources remain for review.

The branch command also mutates the comparison inverse, coverage endpoint and
initial signed domain. These must fail the general theorem. Artifact mutations
must fail their named binding check. Compiler, setup, syntax/type, unrelated or
mixed diagnostics do not count as logical rejection. Dependency revisions,
foundation and generated claims, source identity and actual certificate proofs
receive their applicable checks/audits. Only existing standard axioms are allowed;
no `sorry`, new trusted axiom or native proof shortcut is added.

Go tests cover independently written source, extreme/nonenumerated constants,
32 nested comparisons, real collection and rejection of unsupported effects or
wrappers. Python tests reject mixed failures and failures in the wrong theorem.
CI runs both commands and retains both evidence directories.

## Trust and open implementation boundary

Source equality binds the grammar's renderer to analyzed bytes; it does not
prove official Go semantics refine `Program.eval`. Renderer meaning, Go parser
and compiler, collector, serialization, driver, pinned dependencies and Lean
kernel remain trusted. Runtime samples test those boundaries but do not prove
them. Binding rejects an unrelated unbounded certificate paired with a manifest
and unused declaration claiming `[1,1]`; it is artifact consistency, not a proof
of collection or a sandbox for arbitrary Lean code.

This work does not prove arbitrary Go syntax, pointers/aliases, arbitrary
assignments, loops, concurrency, operational convergence/termination, CLI finding
projection or automatic rewrites. The signed-word interval assumptions remain;
package initialization and resource exhaustion are outside scalar meaning.

[WORKLIST.md](WORKLIST.md#coverage-and-premise-producers) is the implementation
obligation map. Its O1–O3 rows track actual setup, typed source-to-PDG/CFG lowering
and metadata/primitive/rank establishment, full source-path composition, and
non-nil report/call projection. Existing component proofs and the independent
corpus support that future work. They do not remove checker acceptance, establish
source-produced premises or close the unconditional source-to-report theorem.
