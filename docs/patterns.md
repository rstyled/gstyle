# Pattern groups

Patterns have three groups: `general`, `cpp`, and `go`. The group identifies
the detector's logic or language model. It does not identify the language of
the input or prove that every frontend supports the detector.

## General

Shared code includes structural mining, graph matching, clone comparison,
call co-occurrence mining, dependency slices, and input-to-output paths.
Shared source-fact builders include guard clauses, identical boolean returns,
repeated predicates, parameter groups, and domain design proposals.

The registries live in `domain/patterns/general_patterns.go`. Shared matching
runs in `domain/patterns/report.go`; graph algorithms keep their existing
files. These algorithms consume normalized facts. A frontend must supply
those facts and validate the analysis before enabling it.

## Codebase settings

Use the same explicit policy file for patterns and the TUI:

```sh
gstyle patterns --config gstyle.toml --format json .
gstyle patterns --cpg --config gstyle.toml --format json cpg-export
gstyle --cpg --config gstyle.toml cpg-export
```

```toml
[patterns]
similarity_threshold = 80
min_coverage = 80
max_holes = 3

[patterns.measurements]
thin_slice_min_nodes = 30
barrier_slice_min_nodes = 40
chop_min_nodes = 40
cluster_min_nodes = 20
cluster_percent = 50
min_source_lines = 3
min_function_calls = 2
```

The settings above are an example profile. Omitted values keep defaults:
similarity and alignment coverage are 60%, maximum holes are 3,
thin slices start at 10 graph nodes, barrier slices and chops at 20, and
clusters at 10 nodes or 25% of the function graph. Source-line and
function-call minimums default to zero, so they add no filter.

`similarity_threshold` and `min_coverage` use whole percentages from 0 to 100
and control the shared WL/alignment miner. `80` means 80%. Similarity is a
normalized structural score, not a probability of a bug or a percentage of
equivalent behavior. Internal mining scores retain their existing precision.
These settings do not change the separate CCGraph clone or divergence checks.
`--threshold` overrides `similarity_threshold`, including explicit values of
60 or zero. JSON includes the effective percentage `configuration` after overrides.
The same settings reach Go mining and C++ mining in the TUI.

Measurement thresholds select context records. Source-line minimums count
distinct positive source lines in the dependency set; function-call minimums
count callable invocations, including constructors, excluding operators.
Cluster size and fraction are alternative admission tests; setting
`cluster_percent = 0` disables the fraction test. Node, source-line, and call
thresholds remain counts. These settings never
turn a large dependency set into a defect. The four analyses still run when
their measurements fall below a display threshold.

The file is selected with `--config`; it is not auto-discovered. Unknown keys,
negative limits, and out-of-range similarity or coverage values fail with an
explicit config error. Pattern settings are repository-wide and cannot be set
inside quality `[[overrides]]` blocks.

## Go

`domain/patterns/go_patterns.go` combines the general builders with Go rules
through `RunGo` and `RunGoMining`. The Go group includes type switches,
typed-nil checks, unchecked type assertions, swallowed errors, Go slice/index
checks, and the current Go API taint and resource models. Their graph engines
can be shared; their API and cleanup assumptions belong to the Go provider.

Go extraction remains in `adapters/gopatterns`. Existing `Run` and `RunMining`
entry points retain their Go behavior. Detection and fix ordering remain the
same. Size measurements and statistical similarities retain their existing
publication policy.

## C++

`adapters/cpp/patterns.go` supplies canonical graph facts to `RunCPP` in
`domain/patterns/cpp_patterns.go`. The C++ profile enables 27 analyses:

- Thin slices, barrier slices, input-to-output paths (`chop`), and dependency clusters.
- Structural signatures, trait-method and capability groups, generic-function
  and parameterization candidates, CCGraph clones, and inconsistent clones.
- Call co-occurrence rules and alternative-call rules over known call sets.
- Guard clauses, identical boolean returns, primitive concepts, parameter groups,
  factories, repeated predicates, enum dispatch, and domain design proposals
  for identities, services, models, aggregates, and repositories.
- `cpp_empty_catch` for uncommented, syntactically empty catch bodies.

The adapter supplies C++ types, field bindings, literals, signatures, and AST
counts. It runs the shared miners without Go/Rust normalization. Matching
keeps literal values, opaque call effects, and distinct template signatures.
Call association mining excludes a function with any opaque target, since
such a call could satisfy an apparently missing association. C++ factory
proposals use the shared review-only detector; Go's automatic factory fix
retains its parameter cap. C++ source fixes remain manual.

Dependency sizes appear in the separate `measurements` array. Each record
includes its analysis kind, function and seed location, `graph_nodes`,
`function_calls`, a `node_kinds` breakdown, contributing `source_lines`, and
analysis evidence. Function calls include constructors and indirect calls;
operators, variable reads, literals, and return nodes have their own kinds.
A dependency set can branch, and graph-node count includes the seed. It is
neither a call-chain length nor a source-line count.

Size alone never publishes a C++ refactor candidate, TUI source marker, or
SARIF warning. Text output summarizes these context measurements separately.
The four dependency analyses still run and report coverage with
`"output": "measurement"`. Other analyses use `"output": "candidate"`.
Clone similarity remains a review signal. An inconsistent-clone candidate
also requires the shared divergence check; identical groups are not called
inconsistent. These checks do not establish a bug or prove source equivalence.

The checked C++ patterns fixture includes positive and negative cases. Guard
proposals require at least three top-level body operations and a simple returning else
branch; dispatch ladders are excluded. Boolean-return checks require a `bool`
method and direct literal returns on both branches, or on an if branch and
its immediate continuation. Runtime condition expressions, including C++17
initializers, use the CPG AST; compile-time branches are excluded.
Empty-catch checks require an empty body in both the
exported source code and AST; a body comment suppresses this bounded rule.
Comments outside the body and intentional cleanup still need human review.

Further C++ rules belong in this provider when they need C++ semantics, such as
RAII, destructors, ownership transfer, or exception cleanup. A reusable graph
algorithm stays in the general layer. C++ fixtures and capability evidence
must establish each new analysis's supported scope.

## JSON

Each candidate has a `group` field. A thin-slice result from a C++ graph has
`"group": "general"`; an empty C++ catch has `"group": "cpp"`, and a typed-nil
result has `"group": "go"`. Existing candidate
fields remain available. `signature_groups` contains shared mining results.
`analysis_coverage` continues to state which analyses run and their evidence
gaps. Empty C++ results do not imply that disabled rules found no problems.

C++ coverage has `"scope": "imported_graph"`. `complete` means that the query
runs over an import whose method nodes, properties, incident relations, and
operand projection pass validation. Graph queries also require their producer
overlays. Missing prerequisites produce `unavailable`. This is independent of
C++ source-semantic completeness: graph conformance stays `not_evaluated`, and
alias, effect, ownership, and exception semantics remain uncertified. The
next/score safety checks continue to treat this scope as incomplete source
analysis. A successful import cannot prove that the producer parsed every
source construct or resolved every external call.
