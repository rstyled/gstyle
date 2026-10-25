---
name: gstyle-patrol
description: Coordinate the authorized gstyle self-scan patrol, its T3 workers, PR reviews, refreshes, and recovery events.
---

## Goal

Keep the authorized patrol moving by completion events. Fix real analyzer defects in gstyle's current source.

## Constraints

- Existing grants and holds govern. This skill adds no authority. Chief manages; bound workers implement. Read-only tasks stay read-only.
- Allow one implementation/refresh worker, including maintenance, and two open patrol PRs. Exclude unrelated PRs from capacity; avoid their work.
- Preserve real effects, proof bindings/premises/audits/fault controls, and current repository gates. No suppression, rule weakening, metric-only splits, application rewrites, or proof expansion. Ask before heavy dependencies or speculative guards.
- Medium/high risk or unresolved review holds merge and automatic chaining until operator disposition. No-fix alone is not a hold.

## Done check

Reconcile the state file (`~/.cache/gstyle-patrol-chief-state.json`), durable patrol threads, linked/actual PRs, and owned processes. Save state atomically. Review and refresh existing work first.

With no hold, a free worker slot, and fewer than two patrol PRs, launch exactly one worker after a fix or no-fix. After no-fix, choose a different candidate/angle and retain rejection history.

For a new implementation worker, use `t3_thread_launch`, never delegation: unique UTC title `gstyle false-positive patrol — …` and branch `fix/self-scan-…`, with explicit `{type:"worktree", baseRef:"master", branch:uniqueBranch, startFromOrigin:true}`. Reserve before launch. On a lost reply, list threads; recover and reuse the thread, or prove noncreation before retrying. Failed runs retain the slot until owned work is quiescent.

Merge only with operator authority, independent LOW risk, passing required checks on the exact reviewed head, no objection/conflict, and an expected-head guard. Verify the merged master tree against the tested tree.

## Evidence

Review the diff and all five merge-risk categories. Analyzer work needs semantic red/green, retained raw effects, and replay/policies; refreshes also need own/upstream preservation and actual-master overlay red/upstream green with combined gates. Maintenance needs its requested acceptance checks and applicable gates. Keep original evidence immutable in private cache; state scope and performance limits honestly.

## Handoff

Use the [worker brief](references/worker-brief.md) with mode, grants/holds, coordinator, base/head, exclusions, and candidate history. Queue refreshes and maintenance to the existing PR owner with a stable request ID; maintenance changes only the requested material.

Start eligible work before app CI watching; then end without polling. Unwatch superseded/completed PRs or handback. Keep PR links and state current; leave unrelated timers unchanged.
