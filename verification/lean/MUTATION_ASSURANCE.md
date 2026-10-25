# Proof boundary for the mutation assurance experiment

The bounded Go tests documented in `verification/quality/mutation-assurance/` do not
claim a Lean proof of Go ownership or side-effect classification.

The existing profiles in [README.md](README.md) bind specific translated Go
functions or extracted records to their theorems. They include source/hash
checks, axiom audits, and deliberate mutation controls. Those are evidence for
their stated hypotheses and pinned translation/model, not for unrelated code.

No existing profile establishes the end-to-end contract from arbitrary Go
source, through `adapters/goquality` reference bindings and ownership inference,
to `domain/quality` mutation evidence. In particular, a theorem about function
length, PDG queries, or a pattern-check decision does not establish absence of
hidden external mutation in those ownership transfer rules.

The tests therefore use executable Go witnesses for this one claim and
reports its input/observer bounds. Go compiler/runtime behavior, the restricted
instrumentation, pointer-identity observer, and source-site matcher remain its
trusted components. Observed mismatches falsify the bounded contract; absence
of mismatches is not a universal soundness proof. No Lean setup, dependency,
theorem, or trusted assumption changes in this work.
