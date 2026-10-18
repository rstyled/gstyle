# Taint summaries over canonical PDGs

The native pattern pipeline calls `patterns.QueryTaintFlows`. The reusable
`domain/taintflow` query validates supplied graphs and summarizes symbolic input,
result-slot, and direct field dependencies. It uses `bindingflow` operands,
origins, selector indexes, allocation identities, compiler memory-kind facts,
`callflow.InvocationTargets`, canonical capture bindings, and `pdg.BodyFlow`; it does not walk the AST or name application helpers.

A summary can carry an input through a helper return, carry a source discovered
inside a helper back to its caller, or carry an input into a callee sink. Direct
field writes and reads connect builder setters, getters and fluent receiver
returns. Actual object identities keep different callers and builder instances
separate. Field effects are evaluated from the same pre-call store. A strong
update requires one actual target; joins preserve the unchanged value from
returning paths that do not write it. Empty constructors can return fresh
object identities. Compiler-proved immutable scalars do not acquire object
aliases.

The native consumer supplies `Extraction.Callables` separately from declaration
matching. Function values use the shared canonical target resolver. Summaries
specialize on finite callable-name sets in input and capture slots, so a callback
returning a constant does not inherit the payload of an ignored argument. Unknown
target alternatives keep their conservative argument transfer.

Closure values reference environment objects whose fields link to live lexical
cells. Returns and helper calls rebase fresh cells and preserve shared aliases;
capture reads observe call-time stores, not construction snapshots. Distinct
factory sites stay separate. Repeated allocation families join weakly, including
later setter calls, and cannot prove a clean overwrite. Passing a captured cell
by address also retains possible old/new argument dependencies; arbitrary pointer
writes do not prove a clean overwrite.

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
aggregate initialization, collection elements, unresolved targets, bound method
receivers, validation, general numeric/string decoding, deferred effects, or
exceptional heap updates. Captured strings and callable values use the admitted
live-cell model; general closure heap containers remain incomplete. A reported dependency is not proof of a feasible exploit. Missing
findings are not proof of safety.

Typed-source checks cover helper results and sources, sink helpers, distinct
result slots, recursion, variadics, direct and nested receiver fields, aliases,
fluent calls, conditional writes, safe overwrites, separate factory calls,
separate callers, and cross-package calls. An independent two-object oracle
checks all 512 four-write/target combinations. A recursive twelve-link case
checks field widening on an executable depth-based walk. Thirty-two independent
two-cell closure traces check shared writes, safe clears and separate factory
instances. Source/CLI cases cover returned closures, live callable reassignments,
nested relay environments, callback contexts, explicit nil and address escapes.
Seven earlier compiling fault controls exercise source seeding,
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
sources to SQL consumers. The parent `3e1e0b60` reports one SQL finding from
`request.QueryStringParam` to `EntryQueryBuilder.GetEntries`; the fixed
`1b86913c` reports none. The earlier query reports four on each revision.
The difference is decimal-format evidence: typed primitive `%d` output cannot
carry SQL syntax, even when its numeric value depends on input or slice length.
Raw `%s` text and `%c` remain tainted. Named types, interfaces, unknown formats,
and missing or conflicting type evidence do not prove decimal output.
Encoding facts apply only to SQL; numeric SSRF targets retain their dependency.
Unknown external calls discard encoding evidence because they can decode text.
No target code is executed and no Miniflux-specific model is added.

The first broad query took about 49 seconds on a local Miniflux package set.
Immutable set sharing, affected-caller worklists and live expression retention
reduce the measured query to about 2 seconds. These are local measurements,
not a general performance guarantee.

Callable summaries use the same finite receiving-site history as the PDG closure
model. Distinct factory instances remain separate through helpers. Repeated
sites represent allocation families and keep weak writes. The independent
finite-site oracle covers 364 traces, opaque site IDs and owned result sets;
a concrete-cell oracle checks projected writes for unique and repeated cells.
Unknown callback calls retain reachable capture dependencies and may-writes,
including effects forwarded through a known helper. They do not prove a clean
overwrite. Eleven compiling fault controls cover callback context, capture
arguments, live writes, family clears, opaque targets, unknown-call effects and
allocation history, pending-effect boundaries, captured callback returns and encoding loss at unknown calls.

Pending callback reads and writes are distinct from ordinary heap edges. They
project onto captured cells when a caller supplies an environment. Interface
arguments retain possible callback effects without turning every argument into
a general heap mutation. Boundary tests check that effect payloads do not become
heap edges and that ordinary objects do not receive captured-cell writes.
