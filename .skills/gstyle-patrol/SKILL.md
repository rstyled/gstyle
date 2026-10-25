---
name: gstyle-patrol
description: Coordinate the continuous gstyle self-scan false-positive and quality patrol, including T3 workers, PR review, branch refreshes, guarded merges, and recovery events. Use for this patrol, not unrelated repository work.
---

# gstyle patrol

Run the authorized patrol by completion events. Find real analyzer precision defects in gstyle's own current source, preserve real findings, and keep one implementation worker moving when capacity permits.

## Scope and authority

- Reuse the operator's existing authorization and preferences. Loading this skill does not grant indefinite execution, merges, messages, or schedules. Do not ask again for routine work already authorized.
- The chief thread manages work; it does not edit implementation, root, or another worker's checkout. Implementation uses ordinary top-level T3 threads with bound worktrees.
- Keep at most **one active implementation or refresh worker** and **two open patrol PRs**. Read-only review is separate from implementation. Unrelated PRs do not consume patrol capacity, but inspect their scope to avoid duplicate or dependent work.
- Preserve the operator's paused proof work, source bindings, theorem premises, audits, fault controls, and strict failure filters. Do not restart disabled monitors or change unrelated timers.
- No suppressions, broad rule weakening, correct application rewrites, metric-only helper splits, or synthetic-only model additions. Legitimate IO or serialization density is not itself a false positive.
- Obtain operator disposition for medium/high merge risk or unresolved review objections. Ask before heavy dependencies or speculative race/security guards; explain the concrete failure, likelihood, and added complexity in plain words. A demonstrated ordinary fix needs no repeated approval.

## Durable state

Use the configured state file; the default is `$HOME/.cache/gstyle-patrol-chief-state.json`. Keep live details there, not in this skill: coordinator ID, repository and root, operator grants, active worker/run/worktree, open PR heads and reviews, watchers, pending refreshes, actual master, last scanned base, outcomes, holds, candidate history, and evidence paths.

Read it at each event. Reconcile it with durable T3 threads and GitHub. Write it atomically and preserve fields owned by other workflows. Do not copy old scan totals onto a new master before measuring it. Record superseded heads and their evidence instead of overwriting history.

## Handle each completion, PR event, or recovery wake

1. Reconcile T3 threads whose titles start `gstyle false-positive patrol —`, their linked PRs, actual open repository PRs, and the state file. Include any other implementation task that owns the single slot. Inspect recent activity of failed/stopped runs; silence alone is not failure. A completion message can precede the run's terminal status: verify the run is no longer active before reusing the slot. A failed/stopped run does not free the slot while an owned extraction session/process can still run. Inspect its actual session, deadline, and owned process state; preserve edits/evidence and finish or clean up only that owned work before a same-checkout recovery. If ownership or quiescence remains unknown, reserve the slot and report the concrete blocker; do not start an overlapping replacement.
2. Resume existing reviews and required branch refreshes before new implementation. A changed head invalidates prior-head validation. A changed master requires conflict and integration checks; it does not permit merging a conflicted branch whose old checks passed.
3. Review a completed fix on its exact head using the procedure below. For pending CI, use the app watcher instead of polling or a custom sleep loop.
4. When no implementation/refresh worker is active, process the next queued refresh. Refreshes may run while another PR's CI is pending. Serialize refresh workers; use the latest actual master at each start.
5. If no refresh has priority, fewer than two patrol PRs are open, and no applicable hold exists, launch exactly one new worker after a valid fix **or a no-fix**. Do not wait for a recovery timer.
6. Save state, verify links/watchers, and end the event turn. After calling `watch_pull_request`, end without polling.

**A no-fix is not a hold.** Save inspected/rejected candidates and their evidence, then assign a different candidate or concrete new analysis angle. Do not rerun the same unchanged scan or rejection without new evidence. A candidate-specific blocker leads to another candidate. Pause only for an external blocker that prevents meaningful authorized progress, or medium/high risk or an unresolved review needing operator disposition. A medium/high-risk result or unresolved review objection holds that merge **and automatic implementation/refresh chaining** until the operator dispositions it; do not relabel it candidate-only to launch another worker. Record the concrete reason and release condition. Authorized read-only evidence review may continue.

## Launch and manage a worker

Use `t3_thread_launch`, not delegated tasks, native subagents, or `create_threads`, for a new patrol implementation worker. Use a unique UTC timestamp in the title and branch:

```json
{
  "title": "gstyle false-positive patrol — <UTC timestamp>",
  "workspaceStrategy": {
    "type": "worktree",
    "baseRef": "master",
    "branch": "fix/self-scan-<unique UTC timestamp>",
    "startFromOrigin": true
  },
  "message": "<self-contained worker brief and current handoff>"
}
```

Inherit the project/provider/model/options unless the operator requests otherwise. Reserve the implementation slot in durable state before launch. Retain the returned thread ID and verify its bound path, branch, run, and top-level relationship. Launch has no idempotency key. If launch errors or its response is lost, keep an `unknown launch` reservation and inspect `t3_thread_list`, matching title, branch, creation time, and bound run before any retry. An empty early listing alone does not prove noncreation. Use bounded reconciliation; retry only after evidence establishes the prior launch failed without a live thread/workspace preparation. If the outcome stays unknown, preserve the reservation and report it rather than risk a second worker.

Read [the worker brief](references/worker-brief.md) and send it with mode (`new implementation` or `refresh`), current coordinator/repository/state, authorized task and write/push/PR/report scope, applicable holds and forbidden actions, actual base or reviewed starting head, excluded open PR scopes, rejection history, and a new candidate/angle where needed. Carry existing grants forward; do not ask for them again. The brief must not turn a read-only request into implementation or infer a missing grant. Do not send stale PR IDs or totals as facts. Workers report here only when that coordination is authorized and never start a successor.

Use `t3_thread_send` with a stable, task-specific `clientRequestId` for follow-ups to these ordinary threads: `queue` for a new refresh turn, `steer` for an in-flight update. Keep each refresh in that worker's existing bound checkout. Do not discard useful work when master changes during a push; preserve the first red and original base, then arrange normal integration and combined validation.

## Review, refresh, and merge

Immediately `link_pull_request` for every PR the chief works on. Read the actual diff and relevant source/contract, original semantic red, fixed green, true-positive controls, raw self-scan evidence, replay/policy checks, and validation logs. Verify current exact head, checks, comments, reviews, unresolved threads, conflicts, and merge state. Do not accept the worker's label as the review.

Assess all five merge-risk categories: **Alignment, Impact, Verification, Rollback cost, Error detection**. Use the available `merge-risk` skill. Meaningful performance changes need before/after profiling; retain slower/noisy results and allocation costs. A small source diff can affect many callers. Trace large effect deltas and existing summary-budget crossings; effect records are not bug counts.

For refreshes, require a clean reviewed starting head, fresh master, normal merge, and normal push. Regenerate a conflicted baseline from combined code; never choose a stale side. Compare reviewed own code and incoming source/tests. When a shared file has nonoverlapping upstream edits, prove unchanged own hunks and unchanged upstream content outside them; do not demand impossible whole-file equality or hide substantive changes behind line normalization. Verify actual-base semantic red, upstream controls under that same overlay, combined controls, fresh scans, and required gates before push. A stale combined test expectation may need a narrow reviewed update; preserve negative controls and real evidence rather than freezing an obsolete false effect. Document that exception explicitly.

Auto-merge only when the operator has authorized it, final overall risk is **low**, required checks pass on the **exact reviewed head**, and there is no blocking review, conflict, or verification gap. Use an expected-head guard, for example:

```sh
gh pr merge <number> --repo <owner/repo> --merge --match-head-commit <reviewed-sha>
```

Never admin-bypass, force-push, rebase shared branches, overwrite edits, or merge unverified code. Verify the merge commit and actual master tree against the tested tree; if another merge intervenes, inspect its provenance and integration instead of asserting equality. Stop the completed PR's watcher and update state. Reconcile capacity and pending refreshes immediately.

For pending checks, first start any eligible queued refresh or new worker. Then call `watch_pull_request` and end the turn. A watcher does not occupy the implementation slot. Watcher wakes are news, not merge approval: recheck exact readiness. Unwatch a superseded head, completed PR, or PR handed back to the operator. Before finishing PR work, verify `list_thread_pull_requests` and register any missing PR from this work; do not link unrelated background PRs.

## Evidence and limits

Keep artifacts in task-specific home-cache directories, outside the repository. Record commands, exits, source/binary identities, exact base/head, and process bounds. Keep original reports and first-red artifacts immutable; compare normalized deep copies. Match analysis scope: file commands may expand to packages, partial native coverage is not complete coverage, and selected replay equality is not cross-package or full-report correctness.

Follow the fetched repository's actual `AGENTS.md`, tool `--skill`, Make, CI, guard, and verification contracts; do not hard-code an old contract from a prior run. Improvement PRs regenerate their quality baseline and pass `make check` and `make quality-gate` before push, plus affected checks. Preserve upstream feature work even when it changes these commands.

Bound expensive scans by owned PID and deadline, allow graceful termination, then clean up only owned processes. A known independent solver/cancellation or hosted-test issue is neither automatically fixed nor an automatic patrol-wide hold. Record its actual evidence and scope. Report failures honestly; do not weaken deadlines, proofs, policies, or tests to obtain green checks.

Report concise outcomes with exact PR/head and worker links (`t3-thread://v1/<exact threadId>`). State what is active and what happens next. Do not call a capacity wait or a no-fix an operator hold.
