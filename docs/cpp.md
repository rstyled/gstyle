# C++ CPG input

Use `--cpg` to inspect a full C/C++ CPG GraphSON export. Gstyle reads the export
directly; the CPG toolchain and Java are not runtime dependencies of the importer.

Set `CPG_EXPORT` to the full GraphSON exporter from your CPG distribution.
Create the graph with the C/C++ frontend, then export all layers:

```sh
c2cpg --enable-file-content -o cpg.bin src
"$CPG_EXPORT" --repr=all --format=graphson --out cpg-export cpg.bin
gstyle --cpg cpg-export
gstyle search --cpg --format text cpg-export
gstyle patterns --cpg --format json cpg-export
gstyle patterns --cpg --format pdg-json cpg-export > pdg.json
```

The C/C++ frontend may be installed as `c2cpg.sh` or under
`frontends/c2cpg/bin/c2cpg`, depending on the CPG distribution. Use
`--compilation-database build/compile_commands.json` on that frontend when
compiler options are needed. The exporter creates its output directory and
adds missing default/dataflow overlays. Its output directory must not exist.
These commands are verified with CPG frontend/exporter version 4.0.667.

Pass either the export directory or its `export.json` file. One graph is
accepted per invocation. The importer accepts the `tinker:graph` GraphSON
envelope, typed integer IDs, and scalar or singleton-list properties. Metadata
`LANGUAGE` must be `NEWC`, the verified C/C++ frontend.
`VERSION` may identify schema 1.1 or a 4.x CPG producer. Other export dialects,
such as per-method DOT files and the native `cpg.bin`, are not input formats.

## Complexity policy

The `gstyle.cpp.complexity/1` policy scores each non-external method separately,
including lambda methods. Synthetic `<global>` methods are excluded. Scores describe the source
constructs present in the CPG, including unreachable source constructs.

Cyclomatic complexity starts at one. Each `if`, loop, catch, non-default case,
ternary, and built-in logical operator adds one. Switches have no separate
cyclomatic increment.

Cognitive complexity uses source AST nesting. Conditions, loops, switches,
catches, and ternaries add one plus nesting. `else if` and `else` add one without
a nesting penalty. A switch adds one structural increment for all its cases.
Each sequence of like logical operators adds one; changing the operator starts
a new sequence. Each member of a resolved recursive call cycle adds one.
`goto` adds one. Ordinary returns, breaks, and continues add none.

This follows the common rules in the
[Cognitive Complexity white paper](https://www.sonarsource.com/docs/CognitiveComplexity.pdf),
with an explicit method-local policy for lambda bodies. It does not claim exact
Sonar C++ parity. Preprocessor branches, source constructs omitted by the
frontend, unresolved calls, and alternate language frontend conventions can
change the score. The checked fixture covers the C/C++ frontend; other language
frontends need their own source fixtures before claiming comparable scores.

Embedded FILE content supplies source snapshots. Method offsets select the
method body, including lambda source ranges. UTF-16 offsets are converted to
UTF-8 byte spans and checked against the complete method CODE. Truncated or
mismatched CODE does not receive a guessed byte span. Configure the producer to
retain full method CODE when full source display is required.
Without embedded content, method
CODE remains available for display, byte spans remain unavailable, and source
locations stay in the `gstyle.cpp:properties` extension. By default, the importer
reads no local source paths. `--cpg-source ROOT` explicitly permits missing file
content to be read beneath that root. It rejects path and symlink escapes and
requires each supplemented method to match its exported CODE. It does not
replace embedded content or supply compiler flags or missing dependencies.
This option works with TUI, search, and patterns and requires `--cpg`.

## Dependency conversion

Each source method becomes a canonical `domain/pdg.Graph` with the
`gstyle.cpp.import/1` profile. Local declarations and parameters retain symbol
identity through CPG `REF` edges. Direct scalar assignments retain write and
definition facts. Operator calls become computation or assignment nodes;
ordinary calls stay call nodes.
Void method returns become exit nodes and do not add a value-output slot.

Call targets use recorded CALL/REF links and exact full-name lookup. Unresolved
name/signature placeholders never become known callees. Dynamic dispatch is
recorded as indirect; a candidate list alone does not prove a runtime target.
Function-pointer calls remain runtime calls even when the producer gives them
an operator-style name. Dynamic operator calls do not inherit built-in logical
short-circuit complexity rules.
The importer can resolve an exact lambda receiver type or a local callable
with one METHOD_REF initializer, no escaping or other binding uses, and recorded
dominance over the call. Multiple targets, reassignment, missing dominance, and
escapes keep the target unknown. `gstyle.cpp:call-targets` retains candidate IDs,
names, and external-body flags. Macro invocations marked INLINED become
`macro-expansion` computation nodes; their expanded AST remains available and
the wrapper does not count as an unknown runtime call.

Captured locals retain separate capture identities. `gstyle.cpp:captures`
records capture modes and source declarations, including by-value copies.
External declarations use explicit symbol records. Missing REF edges can be
recovered for reads only when every matching recorded reaching definition
identifies the same declaration. Conflicting bindings remain unresolved.
Diagnostics identify unresolved call and binding nodes.

Reference counts now count distinct bound local/parameter read bindings over
the recorded method AST, including unreachable uses. Repeated uses count once;
shadowed declarations count separately. Captures, globals, and member storage
are excluded. Missing read bindings keep the count unknown. The
`gstyle.cpp:reference-count-policy` extension records this policy; the metric
does not count ownership or storage aliases.

`REACHING_DEF` becomes a data edge and `CDG` becomes a control edge. Parallel
edges and original edge IDs survive conversion. Reaching-definition variable
names remain in `gstyle.cpp:variable` annotations, including the current
exporter's generic `EdgeProperty` field. All original edge properties remain
in `gstyle.cpp:edge-properties`. Reaching edges link to a canonical definition
only when their variable matches a declared symbol defined by the source node.
Expression operands feed their
consumers through separate `value-operand` data edges. A simple assignment's
left identifier is a destination, not an input value.
Pointer and field destinations retain a separate `write-address` dependency.

CFG successors remain in `gstyle.cpp:cfg`; they do not become gstyle's
must-precede execution dependencies. Cross-method dependencies remain in
`gstyle.cpp:cross-method-dependencies` for both the source and target method,
including incoming dependencies from excluded global methods. AST containment
does not become control dependence. Other incident relations, including AST,
REF, CONDITION, and CALL, remain in `gstyle.cpp:relations`. A separate validation
pass checks the full method node/property inventory, original relation IDs,
directions and properties, and generated operand roles and positions.
Recorded CFG successors also supply a validated `ValueFlow` context when the
producer includes the method entry. This does not grant native execution-flow
or source-conformance certification.

The C++ adapter supplies plain facts to the existing quality evaluator and
pattern miners. All eight quality rules run: function length, parameter count,
mutation count per target, mutated targets, side-effect density, aggregate
boundaries, repository boundaries, and mutable identities. Bound reference and
pointer writes retain their parameter and field identities. Local helper
summaries use the shared call engine. Opaque calls and object operations keep
conservative effects; they do not disable quality analysis for the file.
`quality.status` is `ready` when evaluation succeeds. A function's `complete`
flag describes effect certainty, separately from which checks run.

Local initialization does not use mutation budgets when a bound local and its
assignment share the same recorded declarator range and scope. Later writes,
compound assignments, captures, static storage, and missing ranges retain their
mutation facts. Initial definitions remain in the PDG. Object lifetime and
unresolved copy/move effects stay conservative; return evidence uses the return
statement, not the METHOD_RETURN metadata location. Unknown effects are estimates
under the configured density policy, not confirmed side effects.

The C++ default effect policy models a bounded set of C/POSIX APIs. Source-backed
functions take precedence over these foreign API models. Project `--config`
policies can add `cpp::` callee prefixes; C++ template specializations retain
distinct names. Go API models and Rust/Go normalization are not applied.
Qualified, single-argument `<cctype>` calls such as `std::isalpha(value)` use a
bounded global-read model for the C locale. The two-argument locale overload and
unqualified unresolved names keep their uncertainty. These API models do not
change an unresolved canonical call target into a resolved target.

Recorded single-target return types and scalar predicate results propagate to
their consumers. `sizeof` produces an integral value even when the operand's
type is unresolved. A synthetic local used only as a scope qualifier, such as
the `Text` in `Text::npos`, does not create an object-lifetime effect. Missing
aliases, receiver types, and possibly overloaded operations remain unknown;
method names such as `size` alone are not evidence of purity.

All 27 pattern analyses run through the shared graph, source-fact, and mining
engines, plus the C++ empty-catch rule. See [pattern groups](patterns.md) for
the registry. Call association mining uses functions whose call targets are
known; an opaque call is never evidence that a required call is absent.
`--threshold` and `--cache` work with C++ mining. Factory suggestions use the
shared detector's review mode, without Go's generated-signature field cap.
The four dependency queries emit context in `measurements`, with graph nodes,
function calls, node-kind counts, and source lines. They produce no refactor
candidate from size alone. TUI markers and SARIF warnings use candidates only.
Inconsistent-clone candidates require measured structural divergence.
Use `--config gstyle.toml` with either patterns or the TUI to tune shared
mining and dependency-measurement thresholds. Explicit `--threshold` overrides
the file's similarity setting. Reports retain the effective `configuration`.
See [codebase settings](patterns.md#codebase-settings) for keys and defaults.

Reports use `imported_graph` coverage. Counts state which input functions pass
each query's prerequisites. Graph queries require their producer overlays;
missing prerequisites produce `unavailable`. Source conformance stays
`not_evaluated`: graph validation does not certify aliases, exception cleanup,
or source rewrites. Findings support review and have no automatic C++ fixes.
No dependency installation or C++ build is required to read an existing export.
Go build tags and Go changed-source selection do not apply to CPG input.
The TUI reuses one verified export snapshot for quality and patterns. Refresh
reads the export again. A prepared pattern scan retains its snapshot across a
refresh, and stale scan results do not replace current findings.

## Fixture regeneration

`adapters/cpp/testdata/sample.cpp` is the source for the checked complexity export.
`adapters/cpp/testdata/patterns.cpp` covers source patterns and graph analyses.
`adapters/cpp/testdata/calls.cpp` covers macro expansion, a resolved local
callback recursion cycle, and an unresolved callback parameter.
`adapters/cpp/testdata/features.cpp` covers shared quality, helper effects,
mining labels, parameter groups, domain facts, and repeated construction.
Generate it with the commands above, using that file as the frontend input.
The checked JSON is minified, with META_DATA.ROOT changed to `.`; all nodes,
edges, IDs, properties, source content, and producer version are preserved.
Normal tests read the checked export and require no CPG toolchain installation.
`zone.graphson.json` is a reduced subgraph of RocketRide's `localZoneOffset`
export; `zone.hpp` preserves the source and license. It covers local
initializations, unresolved calls, and return evidence at its actual source line.
`filepath.graphson.json` is a reduced RocketRide `containsInvalidCharacters`
export. Its embedded file content preserves the source and license. It covers
the qualified C library call, a phantom scope-qualifier local, and unresolved
custom string/view calls that must remain visible.
