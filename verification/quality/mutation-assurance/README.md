# Bounded caller-mutation assurance

The maintained checks are Go tests in `examples/quality-hunt`. They run in the normal test and race jobs.

```sh
go test -buildvcs=false ./examples/quality-hunt -run '^TestMutationAssurance' -count=1 -v
```

`TestMutationAssurance` runs 291 original and 291 instrumented executions. The observer identifies initial caller-owned integer cells by address and records writes, including same-value and restoring writes. It does not consult analyzer ownership facts. Every observed write requires mutation or explicit unknown evidence at the correct case source site. Unrelated effects cannot cover a missing write. Classified and uncertain coverage are reported separately.

`TestMutationAssuranceRejectsUnsafeAnalyzers` builds the current analyzer and three isolated Go-overlay variants. It first requires the unchanged current analyzer to pass the same runtime oracle. Each broken variant must compile, complete all 291 observations, and fail with missing caller-write evidence:

- External mutation effects are dropped.
- Elements read through local containers are treated as locally owned.
- Mixed ownership bindings are treated as locally owned.

Missing mutation anchors, build failures, missing assessments, reduced observation coverage and unrelated failures stop the control test. The overlays never edit production source. Test temporary directories hold builds and reports; failing output identifies the rejected control.

The existing oracle controls reject unrelated IO, wrong-site mutation or uncertainty, effectful places, bad generator expectations and non-equivalent source reductions. Saved-policy checks require the selected analyzer's own policy and reject operational failures or unversioned reports.

## Limits

This is a finite integer-cell contract. Maps, concurrency, newly published cells, library-internal writes, arbitrary destination graphs and multi-target helper effects are outside the runtime observer. Unknown evidence covers a write conservatively; it does not establish precise classification. The observer, source instrumentation, compiler/runtime and source-site matcher remain trusted components checked by controls. A passing run does not prove general soundness or purity.

The [Lean boundary audit](../../lean/MUTATION_ASSURANCE.md) explains which source-bound proofs apply and which ownership connections remain open.

The original frozen-commit study, plan and raw evidence remain in [Git history](https://github.com/rstyled/gstyle/tree/08ea36d3ed8ad554fa1193a0d89849c6aa3e8b72/verification/quality/mutation-assurance). Its one-off runner is retired. Current checks build current source instead of replaying that historical analyzer.
