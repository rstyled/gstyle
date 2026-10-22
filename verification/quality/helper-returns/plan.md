# Helper-return ownership: fixed study plan

Baseline analyzer and policy: `08ea36d3ed8ad554fa1193a0d89849c6aa3e8b72`.
Register this plan and source excerpts before production edits or evaluation.

## Contract and change boundary

Extend existing typed same-package return certificates for simple pointer
helpers. A proven fresh outer result stays local; an unchanged returned formal
retains the caller argument's provenance; other results remain unknown.
Result ownership never proves the helper or its arguments effect-free. Fresh
outer storage does not establish ownership of referenced children.

Start with direct return expressions and bounded helper composition. Preserve
mixed returns, reassignments, publication/escapes, dynamic calls, shared children,
callbacks, explicit policies, saved facts, recursion and depth-limit boundaries.
Do not introduce a callee-name allowlist or rewrite application code.

## Development matrix and acceptance

Retain the #240 runtime matrix unchanged as the initial safety check. Its
borrowed-helper-return has 18 writes covered only by unknown; private-helper-
return has 18 no-write executions with unknown effects. Require the borrowed
identity helper's writes to become classified mutations and the pure fresh
helper's writes to stay local and complete. Preserve all other caller-write
witnesses as mutation or relevant uncertainty.

Add direct fresh, direct borrowed, helper-composed fresh/borrowed, IO-producing
fresh/borrowed helpers, and negative publication/rebind/mixed/shared-child cases
as development regressions. Demonstrate intended failures before production
edits. Keep call effects and policy overrides separately asserted by kind/site.

## External evaluation, fixed before results

Use the unmodified function excerpts recorded in
`examples/quality-hunt/testdata/helper-return-holdout/sources.json`, with hashes,
upstream URLs, versions and licenses. They come from cached Go/tool dependencies;
no project dependency is added. Only support declarations/imports and caller
wrappers are supplied. Preserve function bodies exactly.

Evaluate these seven callers, on seeds -1, 0 and 7:

- x/tools jsonschema.Ptr[int]: write fresh scalar result (no caller write).
- x/tools jsonschema.Ptr[*State]: write shared pointee (one caller write).
- x/tools guru test-fixture Type.Method: return/write caller int parameter.
- The same Method as a method expression.
- mathutil.MaxPtr: write result selected from two caller cells.
- mathutil.MaxPtr(nil, nil): ignored nil result, no write.
- Go ast.NewIdent: write the fresh record's position, no caller write.

These are a small syntax-selected external-source evaluation set, not a blind
sample of production projects. The method example is explicitly a test fixture.
Do not tune implementation against evaluation failures: preserve the first
after-result and report unsupported cases. Fresh record fields with a named
integer type are a no-caller-write control, not a new runtime-cell claim. Keep
the existing observer's initial built-in-int-cell contract.

## Evidence, mutant controls, and stop

Freeze binaries with their own policies and use identical inputs for before and
after. Preserve raw original/traced executions, facts, reports, source/binary and
checker hashes. Repeat #240's three unsafe variants against the candidate,
updating exact overlay anchors if needed and recording that change. Any failed
build, invalid facts, timeout or observer failure is not a semantic mutant kill.

Require no missing write witnesses, reduced uncertainty for the declared
development helper cases, retained true-positive/call-policy controls, and
detected unsafe variants. Run make check and make quality-gate before pushing;
regenerate the baseline. Update goldens only for expected changed behavior.
Deliver one focused PR and before/after evidence. Keep patrol/timer paused and
stop after the report; no successor worker or automatic fix loop.
