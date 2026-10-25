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
- Medium/high risk or unresolved review holds merge and automatic chaining until operator disposition. No-fix alone is not a hold.

## Done check

Reconcile the state file (`~/.cache/gstyle-patrol-chief-state.json`), durable patrol threads, linked/actual PRs, and owned processes. Save state atomically. Review and refresh existing work first.

Before launch, record a current-source finding/unknown or missing-evidence witness, its expected contract, and a concrete suspected mismatch. Check exclusions/history; reuse matching evidence. Rewrite audits require an active rule and nonempty normal CLI preview. Dormancy, valid advice, no-op previews, and partial coverage alone are not leads.

With a qualified lead, no hold, a free slot, and fewer than two patrol PRs, launch one worker promptly. After no-fix, require a different qualified lead. Chief may triage cheaply within a recorded bound; no speculative census-worker chain. With no lead, record exhaustion and end until new evidence or direction. Recovery preserves rejections and exhausted scope; lack of eligible work is not a permission hold.

Launch implementation with `t3_thread_launch`, not delegation: unique UTC title/branch and explicit `{type:"worktree", baseRef:"master", branch:uniqueBranch, startFromOrigin:true}`. Reserve first. After a lost reply, reconcile threads before retrying. Failed runs retain capacity until owned work is quiescent.

Merge only with operator authority, independent LOW risk, passing required checks on the exact reviewed head, no objection/conflict, and an expected-head guard. Verify the merged master tree against the tested tree.

## Evidence

Review the diff and five risk categories: Alignment, Impact, Verification, Rollback cost, and Error detection. Analyzer work needs semantic red/green, retained raw effects, replay/policies; refreshes need own/upstream preservation, actual-master overlay red/upstream green, and combined gates. Maintenance needs acceptance checks/applicable gates. Preserve immutable originals privately; state scope/cost limits. Count false positives corrected, direct/inherited unknown changes, attribution repairs, and restored true effects separately. No-fix reduces nothing; propagated copies are not bugs. Overall false-positive rates require reviewed labels and a denominator.

## Handoff

Use the [worker brief](references/worker-brief.md) with mode, grants/holds, coordinator, base/head, exclusions, and candidate history. Queue refreshes and maintenance to the existing PR owner with a stable request ID; maintenance changes only the requested material.

Start eligible work before app CI watching; then end without polling. Unwatch superseded/completed PRs or handback. Keep PR links and state current; leave unrelated timers unchanged.
