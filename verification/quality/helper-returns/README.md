# Bounded helper-return ownership study

Typed same-package pointer helpers now preserve outer result identity. A fresh
result stays local. An unchanged returned formal keeps its actual argument's
provenance. Call effects remain separate. Fresh outer storage does not prove
ownership of children.

The [plan](plan.md) and external excerpts were committed before production edits
or evaluation. The complete external-source freeze is commit `3a0d3bb291e6533bc51c6899d4fb22c7074d7ce4`.
The fixed comparison uses baseline `08ea36d3ed8ad554fa1193a0d89849c6aa3e8b72`
and candidate `cfb7871933789293fd74b77539efffda553edc49`, Go 1.27.1,
linux/amd64. The final branch also includes master's independent #241 call-flow
memory bound; the combined branch receives the normal required checks.

## Results

| Corpus | Executions per phase | Classified before → after | Unknown before → after | Missing before → after |
| --- | ---: | ---: | ---: | ---: |
| Existing mutation matrix | 291 | 201 → 219 | 102 → 84 | 0 → 0 |
| Helper development cases | 21 | 0 → 6 | 14 → 8 | 0 → 0 |
| Frozen external-source cases | 21 | 0 → 6 | 12 → 6 | 0 → 0 |

These are executed write witnesses, repeated across inputs and syntax variants,
not independent statistical samples. Each phase executes 333 cases as original
and instrumented programs, for 666 program runs. Inputs, traced sources and
observer results match across phases. The existing borrowed-helper family now
classifies all 18 writes. Its fresh-helper family has no effects and is complete
in all 18 executions. No caller-write witness disappears.

The external set has seven wrappers on three fixed seeds. Exact function bodies,
versions, URLs, hashes and licenses are in
[`helper-return-holdout`](../../../examples/quality-hunt/testdata/helper-return-holdout/).
Bodies are compiled in an isolated same-package fixture with support types and
imports. This does not test cross-package summaries or whole upstream projects.

| External body / caller | Before | After |
| --- | --- | --- |
| x/tools `Ptr[int]`, private scalar write | Unknown, no observed caller write | Complete, no effects |
| x/tools `Ptr[*State]`, shared child write | Three unknown witnesses | Three unknown witnesses |
| x/tools guru fixture `Type.Method`, direct call | Three unknown witnesses | Three classified mutations |
| Same method, method expression | Three unknown witnesses | Three classified mutations |
| mathutil `MaxPtr`, two possible caller cells | Three unknown witnesses | Three unknown witnesses |
| `MaxPtr(nil, nil)`, unused result | Complete, no effects | Complete, no effects |
| Go `ast.NewIdent`, private record field write | Complete, no effects | Complete, no effects |

The method body is an upstream tool test fixture, not production application
code. This small syntax-selected set cannot estimate population precision or
recall. The first after-result is preserved separately and has the same binary,
source and trace hashes as the reproducibility run. No implementation change
follows that evaluation. The external reports have zero diagnostics in both
phases: warning count alone would miss the evidence improvement.

## Safety checks and scope

The regression fails on unchanged production for both fresh and borrowed
results, then passes with the fix. Typed controls retain mixed returns,
reassignment, publication, alias escape, address exposure, closure capture,
defer/go calls, dynamic calls, shared children, generic children, and unsupported
named results. Helper composition uses the existing eight-pass bound;
unresolved cycles and long chains stay unknown. Existing literal-constructor
certificates are unchanged. This is not a proof for all pointer-returning code.

IO, global writes, argument effects and callbacks have separate assertions.
Saved facts replay identically. Explicit IO/global/none helper policies retain
the caller's direct mutation. The golden changes only for a proven fresh
temporary: its mutation record stays present as local; tuple uncertainty and
shared-slice effects remain.

All three deliberately unsafe variants compile and fail through the unchanged
runtime oracle: ignoring external mutation loses 219 witnesses, wrongly owning
referenced elements loses 36, and wrongly owning mixed bindings loses 12.
Declaration-reduced witnesses reproduce each failure. Build errors, invalid
facts, analysis failures and observer failures do not count as semantic kills.

The observer covers executed writes to **initial caller-owned built-in `int`
cells**, including same-value and restored writes. It checks original/traced
final values, lengths and write counts. It does not prove behavior for all inputs,
newly published cells, all scalar types, concurrency, IO or callbacks. Typed
controls cover the new publication and effect boundaries; they are not a runtime
proof of those excluded behaviors. Named `token.Pos` is only a no-caller-write
control. The observer and checker are part of the trusted test code; raw sources
and hashes make that boundary inspectable.

## Cost and reproduction

Two sequential profile pairs use the same archived source, `GOMAXPROCS=2`, and
three extraction iterations per build. Time changes from 1.454 to 1.547 seconds
in one pair and 1.503 to 1.409 seconds in the other. This short sample does not
establish a speed change. Allocated bytes rise from 717.9 to 731.4 MB and from
718.6 to 729.4 MB, about 1.5–1.9%. CPU and allocation profiles are retained.
Earlier overlapping exploratory runs are not used for a timing claim.

From the repository root, with a new output directory:

```sh
GOTOOLCHAIN=go1.27.1 python3 verification/quality/helper-returns/run.py \
  --output "$HOME/.cache/gstyle-helper-return-reproduction"
```

The script rebuilds both frozen analyzers with their own policies, demonstrates
the intended red/green regression, executes the same inputs, checks unsafe
variants and records sequential profiles. The [assessment](assessment.json)
binds the [raw evidence archive](evidence.tar.gz), binary hashes, checker hashes
and results. The archive omits rebuildable executables and checkout copies.
The existing mutation-assurance runner's new optional `--revision` supports
candidate testing; its original frozen default is unchanged. No dependency is
added. `make check`, `make quality-gate`, lint, cognitive checks and affected
race tests validate the combined branch before push; the baseline is regenerated. Across the combined source, density excess rises
from 258,744 to 259,244 as the implementation adds functions; its rate falls
slightly across 5,246 functions. This study claims better ownership evidence,
not a lower repository warning count.

Keep the patrol and recovery timer paused. This is one bounded improvement;
the next task is a self-audit for code reduction, with no automatic fix chain.
