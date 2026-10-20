# Go quality coverage and false-positive roadmap

Draft audit at master `2c275674fb3f0862c1fa844ba002ad3454cf43c1`.
Scope: `gstyle check`, from typed Go source through ownership/call facts,
helper summaries and rule evaluation. The PDG pattern/SAST paths have separate
capabilities; a PDG node alone does not prove quality-consumer support.

This adopts the evidence model in [rstyle's feature coverage roadmap](https://github.com/rstyled/rstyle/blob/6fbf0510cb3e4f234fa6f5273a8b3deecfb703b8/rust-feature-coverage.md).
The unit of progress is a complete real-code function family with checked
positive and adverse cases. A new helper, fixture, or capability row is not a
measured reduction.

## Status and evidence

| Status | Meaning |
| --- | --- |
| Covered | The exact bounded case in the row has connected implementation and recorded regression evidence. Broader compositions remain separate. |
| Partial | A checked subset exists, with a named unsupported or unproved composition. |
| Missing | A source-selected case lacks the required connected proof. |
| Not audited | The current audit has not established a support boundary. |

Diagnostic decisions are separate: confirmed false positive, valid finding
under the current rule, or unresolved. Unknown/incomplete evidence does not by
itself decide that classification. A claim of zero false positives requires a fully reviewed corpus. Unresolved
cases block that claim; report them separately from confirmed false positives.

## Frozen starting inventory

| Item | Count |
| --- | ---: |
| Functions | 5,030 |
| Findings | 665 |
| side_effect_density | 501 |
| mutation_per_target | 76 |
| mutated_targets | 48 |
| fn_params | 23 |
| aggregate | 17 |
| Incomplete functions | 1,040 |

Every finding has a fixed Q-number plus its original function, rule and source
location. The source tree, report and binary hashes are frozen in
[`verification/quality/assessment.json`](../verification/quality/assessment.json);
[`selection.tsv`](../verification/quality/selection.tsv) retains all 665 identities. Preserve the old inventory. Extension fixtures and new implementation
functions have separate measurements. The total is a warning count, not a
false-positive count.

## Seed coverage ledger

These rows start from existing implementation and regression evidence. The
parallel audit must check their compositions before broadening any status.

| ID | Bounded capability | Status | Boundary / evidence |
| --- | --- | --- | --- |
| O01 | Private receiver outer slots | Covered | Closed construction/use proof, excluding publication and address/alias escapes. Referenced storage needs its own proof. `adapters/goquality/closed_receiver_test.go`. |
| O02 | Copies of inline scalar records and arrays | Covered | Reuses copied-value use checks; pointer methods, addresses, array slice views and referenced children retain conservative evidence. `TestClosedReceiverStorageBoundaries`. |
| O03 | Fresh reference-valued receiver fields | Partial | Constructor/field checks exist; passing a private field to an ordinary read-only helper can reject closure. Source-selected private-map chain under audit. |
| O04 | Empty private map of append-only slice entries | Covered | Only same-map append construction establishes owned outer entries. Borrowed insertion, copying, aliases, range publication, captures and defer reject the proof. `TestContainerStorageOwnership`. |
| O05 | Nested references inside map/array/record entries | Partial | Outer allocation does not certify children. SourceFlow entry replacement mutates only the outer map; its helper-retention gap is O03, not a need to certify child writes. General nested child mutation remains separate. |
| O06 | Full-capacity append allocation | Covered | Explicit full-slice bounds, capped helper results and same-block reaching local headers. Nonempty scalar arguments, explicit nonempty slice literals and string constants establish allocation; empty/unknown spreads do not. Existing container and append-budget fixtures. |
| O07 | Fresh helper and tuple results | Partial | Existing fresh-result and recursive forwarding proofs have borrowed, escaped, pointer-element and deferred-result controls. Arbitrary mixed origins/returns remain conservative. `container_test.go`. |
| F01 | Reaching values through branches/loops/closures | Partial | Specific bounded proofs exist. General cross-block capacity propagation and captured/addressed headers are not certified by O06. |
| E01 | Static read-only helper recursion | Covered | Every reachable body and operation must pass; unknown calls, policy overrides, missing bodies and limits retain evidence. `domain/quality/helper_purity_test.go`. |
| E02 | Caller-owned storage through helper parameters | Partial | Storage substitution and context checks exist. Receiver-field non-retention and general nested storage composition have distinct proof boundaries. Context-limit controls in `context_purity_test.go` and `storage_test.go`. |
| E03 | Known callbacks, method values and generic helpers | Partial | Some callback evidence is expanded; exact method-value callback/receiver transport through generic mapItems is source-selected for audit. Arbitrary callable inputs stay unknown. |
| G01 | Read-only package data | Partial | Stable literal/constructed-data proofs exist. The fixed-table milestone handles copied lookup values used as another target's index and exact `slices.Contains` reads, including parentheses. Other queries and broader data shapes remain unproved. Exported/rebound/escaped mutable state stays visible. |
| G02 | Typed standard-library package classification | Covered | Installed stdlib coverage is checked by TestStdlibCoverage. Classification inventory completeness does not establish the correctness of every call-specific effect model. |
| R01 | Call-site density budget identity | Covered | Count each source call once per effect kind; retain raw evidence and assess helpers separately. Legacy missing sites remain independent. `domain/quality/call_density_test.go`. |
| R02 | Mutation budget identity and granularity | Not audited | Audit assigned mutation_per_target and mutated_targets identities against the current rule before changing extraction or counting. |
| R03 | Parameter and aggregate rules | Not audited | Audit actual rule semantics and source. Valid design warnings require code/design work, not false-positive suppression. |
| X01 | Saved-fact replay | Covered | Existing versioned facts and legacy missing-evidence boundaries have validation/round-trip tests. Every changed family must preserve replay behavior. |
| X02 | Build/test variants and source selection | Partial | Typed package errors and conflicting variants are explicit. Broader target/tag combinations require separate pinned contexts. |
| X03 | Resource bounds and incomplete outcomes | Partial | Failed scans are operational failures, not clean results. The audit uses explicit RSS/time/output limits and verified compressed archives. Broader stress coverage is not established. |
| X04 | Held-out false-positive and false-negative quality | Not audited | Pin independent real repositories/build contexts and expected outcomes before evaluating. Own-repository improvement and small fixtures do not establish held-out quality. |

## Work split and integration boundary

| Lane | Assigned findings | Exclusive audit area |
| --- | ---: | --- |
| Ownership | 343 | Adapter allocation, copies, fields, aliases, escapes, receiver/container proofs and mutation budgets. |
| Summaries | 219 | Helper/caller composition, recursion, callbacks, dynamic boundaries and density evidence. |
| Policy/globals | 103 | Global reads, library classifications, parameter/aggregate rules and intended policy. |

Each lane has a separate worktree and a disjoint diagnostic roster. Cross-lane
source reads are allowed. Phase 1 changes no production code. Before phase 2,
the integrator assigns exact file locks and owns shared extraction interfaces,
baselines, golden reports and complete-corpus scans. Agents do not open or merge
PRs independently.

For every selected family, record the original source identities, full blocker
chain, independent semantic expectations, borrowed/published/escaped and actual
side-effect controls, and a small development case. Require source extraction,
summary evaluation, JSON diagnostics and saved-fact replay to compose. Reuse
existing proof mechanisms before introducing a new one.

Rank confirmed safe reductions against net production code, proof complexity
and performance risk. Report affected findings, evidence-only upper bounds and
actual reductions separately. A correctness fix may have zero own-repository
reduction; record that honestly. Reassess after successive zero-gain foundations.

## Completion checks for a milestone

- Preserve all frozen identities, controls, thresholds and suppressions.
- Record valid findings and unresolved cases as well as removed false positives.
- Measure fixed-source removed/added identities separately from the changed repository.
- Record production-code cost, incomplete coverage and remaining blockers.
- Run affected tests, saved-fact checks, required make check and make quality-gate.
- Refresh the baseline and affected golden reports; obtain fresh CI before merge.
- Retain real IO, global writes, borrowed storage, escapes and helper argument effects.
- Bound full scans and record performance/memory evidence appropriate to the change.

The ledger is an evidence index, not a claim of full Go semantics or zero false
positives across all Go projects. Keep historical results and exact boundaries.

## First audit batch and fixed-table milestone

The three agents review 50 distinct frozen identities: seven confirmed false
positives, 43 valid under the current rule, and 615 unresolved. A valid warning
can still contain imprecise evidence; track those evidence bugs separately.
The parameter review confirms counts under the current policy, not that every
signature is poor design.

The fixed-table patch adds 17 net production lines. It removes four warnings on
the unchanged 5,030-function source: **665 to 661**, with zero added warnings
and incomplete coverage unchanged at 1,040. The changed repository separately
reports **661 findings across 5,031 functions**. The source-selected two-warning
ceiling described only the policy lane; full measurement also removes the
projection edge and rule-evaluator warnings caused by the same read proof.
Both extra identities are reviewed and recorded in the assessment.

Real destination writes, key-evaluation effects, mutable/escaped tables,
addresses, callback queries and counterfeit callees retain their evidence.
Independent review checks these controls, including parenthesized generic calls.
Full tests, complexity/cognitive limits, vet and the quality gate pass. Baseline
regeneration changes no threshold or suppression. Both bounded frozen scans
complete below 6 GiB RSS and 120 seconds. Single paired timings do not establish
a speedup.

After this milestone, 52 frozen identities have been reviewed: nine initial
false positives, four fixed, five still confirmed and open, 43 valid findings,
and 613 unresolved. Keep the original 665-warning denominator.

| Next family | Selected warning IDs | Evidence-only ceiling | Production cost / boundary |
| --- | --- | ---: | --- |
| Private map fields consumed by non-retaining helpers | Q0299, Q0301, Q0302 | 3 | Separate prototype uses bounded acyclic typed formal-use certificates; borrowed children and returned aliases stay conservative. Full integration is pending. |
| Fresh nested child construction and private self-slicing | Q0549 | 1 combined; self-slicing alone 0 | Close the whole stronglyConnected chain. Audit whether the existing append-only map-entry proof can be reused before adding another ownership framework. |
| Known method-value callbacks through generic helpers | Q0306 | 1 | A complete general design is estimated at 180–300 production lines. Seek reuse or a narrower sound contract first; keep real writer/control effects and borrowed results. |

These rows do not promise reductions. Callback and nested-storage work remains
open; this milestone does not establish zero false positives.
