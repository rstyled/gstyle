---
name: gstyle-patrol
description: Coordinate the authorized gstyle self-scan patrol, its T3 workers, PR reviews, refreshes, and recovery events.
---

## Goal

Fix demonstrated analyzer errors in current gstyle source. Measure corrected findings, uncertainty, and missing evidence separately; fewer reports alone are not success.

## Constraints

- Existing grants and holds govern. This skill adds no authority. Chief manages; bound workers implement. Read-only tasks stay read-only.
- Allow one implementation/refresh worker, including maintenance, and two open patrol PRs. Exclude unrelated PRs from capacity; avoid their work.
- Preserve real effects, proof bindings/premises/audits/fault controls, and current repository gates. No suppression, rule weakening, metric-only splits, application rewrites, or proof expansion. Ask before heavy dependencies or speculative guards.
- Non-low risk-gate results or unresolved review hold merge and automatic chaining until operator disposition. Analysis errors require resolution; pending CI blocks merge. No-fix alone is not a hold.

## Done check

Reconcile the state file (`~/.cache/gstyle-patrol-chief-state.json`), durable patrol threads, linked/actual PRs, and owned processes. Save state atomically. Review and refresh existing work first.

Before launch, record a current-source finding/unknown or missing-evidence witness, its expected contract, and a concrete suspected mismatch. Check exclusions/history; reuse matching evidence. Rewrite audits require an active rule and nonempty normal CLI preview. Dormancy, valid advice, no-op previews, and partial coverage alone are not leads.

With a qualified lead, no hold, a free slot, and fewer than two patrol PRs, launch one worker promptly. After no-fix, require a different qualified lead. Chief may triage cheaply within a recorded bound; no speculative census-worker chain. With no lead, record exhaustion and end until new evidence or direction. Recovery preserves rejections and exhausted scope; lack of eligible work is not a permission hold.

Launch implementation with `t3_thread_launch`, not delegation: unique UTC title/branch and explicit `{type:"worktree", baseRef:"master", branch:uniqueBranch, startFromOrigin:true}`. Reserve first. After a lost reply, reconcile threads before retrying. Failed runs retain capacity until owned work is quiescent.

Use the [repo risk gate](../../docs/change-risk.md): require `gstyle risk --base BASE_SHA --head HEAD_SHA --check`, with analyzer and policy from that exact trusted base, plus matching hosted `risk-gate`. Only LOW/exit 0 is eligible; medium/high/unknown/unrated require operator review, exit 2 is failure, and advisory exit 0 proves nothing. Never substitute candidate-built analysis or subjective scores; do not bypass the gate. Also require operator authority, independent semantic review, required reviews/all exact-head checks, no objection/conflict, and an expected-head guard. Static LOW does not cover CLI/runtime/test contracts or authorize merging. Verify merged master against the tested tree.

## Evidence

Review the diff and gate report: code impact, dependencies, API changes, coverage, and reasons. Keep semantic/test evidence separate. Do not run the `merge-risk` skill or produce its five-category score. Analyzer work needs semantic red/green, retained raw effects, replay/policies; refreshes need own/upstream preservation, actual-master overlay red/upstream green, and combined gates. Maintenance needs acceptance checks/applicable gates. Preserve immutable originals privately; state scope/cost limits. Count false positives corrected, direct/inherited unknown changes, attribution repairs, and restored true effects separately. No-fix reduces nothing; propagated copies are not bugs. Overall false-positive rates require reviewed labels and a denominator.

## Handoff

Use the [worker brief](references/worker-brief.md) with mode, grants/holds, coordinator, base/head, exclusions, and candidate history. Queue refreshes and maintenance to the existing PR owner with a stable request ID; maintenance changes only the requested material.

Start eligible work before app CI watching; then end without polling. Unwatch superseded/completed PRs or handback. Keep PR links and state current; leave unrelated timers unchanged.
