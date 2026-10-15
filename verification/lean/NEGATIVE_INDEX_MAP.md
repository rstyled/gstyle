# Negative-index proof map

Reviewed at `9bf8562` (merged #103 and #104). This is a source and theorem review,
not a new proof stage. The complete negative-index pipeline is not formally verified.

## The guarantee to work toward

The current rule is an advisory: upper-check evidence exists, but the analysis
has not established a nonnegative index. It does not establish an executable
negative-index witness. Conversely, silence does not establish index safety:
`return s[i]` without an upper-check trigger intentionally produces no finding.
`UpperChecked` is trigger metadata, not proof that every path has an upper bound.

Keep three obligations separate:

| Obligation | Required result |
| --- | --- |
| Advisory fidelity | The output accurately describes the supplied analysis evidence and identifies the correct source site. It does not claim a concrete failing execution. |
| Analysis soundness | Every concrete value in the supported source subset remains represented through the production analysis. A possible negative value is not incorrectly discarded. |
| Implementation correctness | The actual Go path implements the stated abstract operations and reporting policy. |

Use one bounded source-to-diagnostic target: a 64-bit signed local copied by a
single plain assignment, followed by a length comparison and collection access.
For example:

```go
func At(s []int, i int64) int {
    j := i
    if j < int64(len(s)) { return s[j] }
    return -1
}
```

The proposed target is: if analysis completes successfully, every negative input
reaching this triggered access yields an advisory for the correct binding and
source site, with no automatic fix. Prove this for the actual extraction,
analysis and reporting path. A real negative input is an antecedent; a finding
is not itself proof such an input is reachable in the caller's program.
A corresponding early lower-bound rejection must avoid a negative-index advisory.
Neither direction is established end to end by the current theorems.

The initial target excludes loops, heap/field-derived index state, aliases,
non-builtin calls, globals,
concurrency, compound arithmetic, narrow/unsigned conversions, suppression and
32-bit targets. These are proof exclusions, not claims that production rejects
such programs. Unknown fallbacks and missing proof certificates must never be
interpreted as verified safety. Scope admission itself needs a checked boundary;
there is no production verified-subset admission mechanism today.

## Production path and current evidence

Links to Go start at the repository root; proof links are local to this project.
“Conditional” means the theorem applies only under its stated input assumptions.

| Boundary | Production code | Existing proof and remaining obligation |
| --- | --- | --- |
| Typed source to facts, operands and control flow | [ir.go](../../adapters/gopatterns/ir.go), [ir_runtime_facts.go](../../adapters/gopatterns/ir_runtime_facts.go), [ir_runtime_flow.go](../../adapters/gopatterns/ir_runtime_flow.go) | Go tests. No proof that emitted facts, widths, binding identities, operand order or CFG outcomes mean what the source program means. Structural validation alone does not establish this. |
| Snapshot and RHS queries | [graph.go](../../domain/runtimeguard/graph.go), [value.go](../../domain/runtimeguard/value.go) | [ValueEvaluation](Gstyle/Proof/Transfer/ValueEvaluation.lean), [ValueResolution](Gstyle/Proof/Transfer/ValueResolution.lean): conditional production execution for finite queries. Snapshot construction, successful witness availability and source-value meaning remain obligations; `ParseInt` has an explicit contract. |
| Assignment and state updates | [flow.go](../../domain/runtimeguard/flow.go), [increment.go](../../domain/runtimeguard/increment.go), [pointer.go](../../domain/runtimeguard/pointer.go) | [Assignment](Gstyle/Proof/Transfer/Assignment.lean), [LocalInvalidation](Gstyle/Proof/Transfer/LocalInvalidation.lean), [Step](Gstyle/Proof/Transfer/Step.lean): proved pieces. No composed `analysis.transfer` theorem; tracking, guard clearing, increment/range fallbacks and caller input conditions remain. |
| Branch filtering | [condition.go](../../domain/runtimeguard/condition.go) | [Assume](Gstyle/Proof/Transfer/Assume.lean), `wp_finite_assume_sound`: preserves represented environments satisfying the condition, given finite certificates, `AssumeMeaning`, value/symbol meaning and valid snapshots. These source-meaning assumptions are not discharged by the PDG producer. |
| Join and worklist | [flow.go](../../domain/runtimeguard/flow.go) | [StateMap/Join](Gstyle/Proof/StateMap/Join.lean), `wp_join_sound`, and equality/copy proofs. Conditional interval union and exact map behavior; no proof of the complete worklist, reachable block states, block execution or convergence. The bounded straight-line target still needs its actual traversal proved. |
| State to access record | `analysis.access`, `report`, `publicAccess` in [flow.go](../../domain/runtimeguard/flow.go) | Integration tests. No composed proof that the correct index value, binding, trigger and source node reach each report record. |
| Decision and deduplication | [interval.go](../../domain/runtimeguard/interval/interval.go) | [Risk](Gstyle/Proof/Interval/Risk.lean), `wp_NegativeIndex` and `wp_NegativeDetected`; [Select](Gstyle/Proof/Interval/Select.lean), `wp_SelectNegative`: exact decision and first-risky-record policy. A represented negative value with both triggers yields a selected record for that binding, possibly an earlier risky occurrence. Not a concrete reachability theorem. |
| Candidate and final output | [negativeindex.go](../../adapters/gopatterns/negativeindex.go), [nilguard.go](../../adapters/gopatterns/nilguard.go), [patterncheck/cli.go](../../application/patterncheck/cli.go), [patterns/report.go](../../domain/patterns/report.go) | [Advisory](Gstyle/Proof/Interval/Advisory.lean), `wp_Describe`, and generic `pdg/query.At` prove helpers. Node filtering, symbol-name lookup, payload-to-Candidate copying, suppression, report assembly and rendering are not composed into a production theorem. |

Thus “no false-safe interval decision under supplied triggers” is proved locally.
“Every warning is a real bug,” “no warning means safe,” and the bounded
source-to-diagnostic target above are not proved.

## A production counterexample to the current wording

At the reviewed commit, `gstyle patterns --format json .` reports this function:

```go
func LowerCheckThenAddZero(s []int, i int) int {
    if i < 0 { return -1 }
    i += 0
    if i < len(s) { return s[i] }
    return -1
}
```

Its observation says “upper-bound check on i has no lower-bound check; i indexes
a collection.” The source has a lower check. Adding zero cannot change `i`, and
the later length check protects the access. The warning follows the conservative
compound fallback, not an executable negative index. The current payload theorem
proves the exact message is constructed; it does not prove the message's claim
about source checks. This example is outside the proposed plain-assignment subset.

A four-function production CLI check confirms: upper-check-only warns; complete
lower/upper checks do not; lower-check-then-add-zero warns; no upper check does
not warn. Existing [adapter regressions](../../adapters/gopatterns/negativeindex_test.go)
and [application regression](../../application/patterncheck/negativeindex_test.go)
exercise the source pipeline, but are tests rather than universal proofs.

## One recommended next PR

Align the advisory text with the established evidence: report that nonnegativity
was not established after upper-check evidence, rather than asserting that the
source contains no lower-bound check. Keep the decision, score, sites and
no-auto-fix policy unchanged. Update the exact payload contract and add the
source counterexample as a regression beside the positive and negative controls.
Require the existing source binding, axiom audit, relevant semantic mutations and
application checks to pass. Do not introduce another transfer proof in that PR.

This fixes a demonstrated user-visible overclaim and makes the contract honest.
It does not improve interval precision or prove source-to-diagnostic soundness.
After review, choose the next proof by which boundary in the table it closes for
the bounded target. Helper count is not the acceptance criterion.

## Trust and independent checks

Existing proofs depend on the pinned Goose translation and Go semantic model,
64-bit widths, explicit layout/unfolding/initialization hypotheses, the parser
contract where used, and Lean's kernel. The permitted global axioms are
`propext`, `Classical.choice` and `Quot.sound`; that audit does not discharge
explicit semantic hypotheses. Iris WP establishes partial correctness, not a
whole-analyzer termination theorem. Compiling/running the tool adds the normal
compiler and runtime trust boundary. A practical milestone must also decide
whether Go's parser/type checker is trusted. If so, label the result as a
contract from a supplied typed AST to diagnostics; do not silently present
assumed frontend truth as proved source semantics.

For the bounded target, independent evidence must include kernel-checked
production composition with an assumption list, regenerated source bindings,
source-level positive/negative examples and mutations spanning extraction,
transfer, access projection and final site/payload copying. Tests and mutation
rejection support the contract; they do not replace the missing theorems.
