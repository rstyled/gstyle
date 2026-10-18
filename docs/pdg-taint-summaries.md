# Taint summaries over canonical PDGs

The native pattern pipeline calls `patterns.QueryTaintFlows`. The reusable
`domain/taintflow` query validates supplied graphs and summarizes symbolic input,
result-slot, and direct field dependencies. It uses `bindingflow` operands,
origins, selector indexes, allocation identities, compiler memory-kind facts,
and `pdg.BodyFlow`; it does not walk the AST or name application helpers.

A summary can carry an input through a helper return, carry a source discovered
inside a helper back to its caller, or carry an input into a callee sink. Direct
field writes and reads connect builder setters, getters and fluent receiver
returns. Actual object identities keep different callers and builder instances
separate. Field effects are evaluated from the same pre-call store. A strong
update requires one actual target; joins preserve the unchanged value from
returning paths that do not write it. Empty constructors can return fresh
object identities. Compiler-proved immutable scalars do not acquire object
aliases.

Caller and body worklists solve dependencies without a fixed pass count.
Only expression values used across blocks survive a block transfer. Value maps
are immutable after publication; joins can share an existing superset.
Recursive field paths widen after eight indexes. A wildcard reads reachable
stored descendants and cannot justify a strong clear.

Source and sink models remain explicit standard-library API contracts. Tuple
slots stay separate; variadic formals receive the union of actual elements.
Unknown external calls conservatively propagate their argument dependencies.
Graphs without body flow retain the existing local graph query. Invalid graphs
passed to the summary API return an error. Native reports expose that error as
not-evaluated taint coverage with a reason. Cross-function findings retain the
actual source path/function and separate source-node evidence from sink-node
evidence.

## Scope and checks

All taint coverage remains partial. The query does not prove arbitrary
aggregate initialization, collection elements, dynamic targets, closure cells,
validation, numeric/string encoding, deferred effects, or exceptional heap
updates. A reported dependency is not proof of a feasible exploit. Missing
findings are not proof of safety.

Typed-source checks cover helper results and sources, sink helpers, distinct
result slots, recursion, variadics, direct and nested receiver fields, aliases,
fluent calls, conditional writes, safe overwrites, separate factory calls,
separate callers, and cross-package calls. An independent two-object oracle
checks all 512 four-write/target combinations. A recursive twelve-link case
checks field widening. Seven compiling fault controls exercise source seeding,
result substitution and slot choice, field transfer and identity, strong clears,
and variadic mapping.

The source corpus now detects `IndirectSQL`: 35 of 36 unsafe cases are reported,
with the existing one flagged safe case unchanged. Request-object URL flow and
branch validation remain recorded gaps.

## External evidence

The later Miniflux sort hardening commit `7628a214` is not a confirmed unsafe/safe
benchmark: the pre-fix API handler already validates sort columns and direction.

The older [search-ranking SQL fix](https://github.com/miniflux/v2/commit/1b86913c005c082ed27fd7f1f42fc6ae9bdcbb9b)
changes raw search-text interpolation into a SQL parameter. Static scans of the
actual `storage`, `ui`, and `http/request` packages now connect request-helper
sources to SQL consumers. Both revisions still report four findings, including
numeric/length-derived dependencies. This demonstrates that summaries cross the
builder boundaries; it does not establish that the scanner distinguishes the
fix. Encoding and value-shape precision remains separate work. No target code
is executed, and no application-specific model is added for this experiment.

The first broad query took about 49 seconds on a local Miniflux package set.
Immutable set sharing, affected-caller worklists and live expression retention
reduce the measured query to about 2 seconds. These are local measurements,
not a general performance guarantee.
