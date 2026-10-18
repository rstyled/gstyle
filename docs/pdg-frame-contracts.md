# PDG call-frame effects

The callflow query summarizes explicit frame-sensitive calls in canonical PDGs.
It separates a known observation from missing evidence. It does not summarize
implicit panic paths, arbitrary memory effects, termination, scheduling or all
ways that a caller can observe an added frame. Those remain in the completion
work plan.

A direct call uses its canonical Callee declaration identity. An indirect call
uses its call-target edge, actual reaching definitions and strong value copies.
A stored free function or static method expression has one compiler-owned
gstyle.go:function-value string fact. Bound method values, unresolved interface
dispatch, closures and callback parameters remain unknown unless their
implementation can be established by a later query. A function value's creation
does not execute its body.

Package summaries converge over the actual target relation without a pass limit.
Local recursion propagates seeded observations and uncertainty; recursion alone
does not invent either. The query conservatively includes every canonical call,
including deferred invocations and calls outside admitted reachable flow.
Missing body flow is unknown. The adapter checks every contributing graph against
its source snapshot; a stale transitive body cannot authorize a rewrite.

## External contracts

The following identities are explicit observations. A supplied body cannot replace
this classification:

- recover and panic
- runtime.Caller, runtime.Callers and runtime.Stack
- runtime/debug.Stack and runtime/debug.PrintStack

The builtin contracts in domain/callflow/contracts.go do not invoke user callbacks
or explicitly inspect stack frames. They do not establish successful completion.
Only exact compiler builtin identities match; a package function named len,
append or recover keeps its package identity.

Three narrow standard-library contracts exclude explicit frame inspection and
user callbacks:

| Identity | Reason | Reviewed source |
| --- | --- | --- |
| errors.New | Constructs an errorString containing the argument | Go 1.27.1 src/errors/errors.go |
| strings.TrimSpace | Scans text and uses the fixed unicode.IsSpace predicate | Go 1.27.1 src/strings/strings.go and src/unicode/graphic.go |
| path.Clean | Scans text with the package's internal lazy buffer | Go 1.27.1 src/path/path.go |

This is a reviewed contract boundary, not a generated proof or a broad contract
for those packages. The reviewed file SHA-256 values are:

- errors/errors.go: 865b9cce008bf68d4d2cc4146455a9c5692756bbe57be29ee1ef0de345dafe1e
- strings/strings.go: a81bdf73a308277c322e35d0c602e3813f57b6f769c091f8f8962c8884a45ea3
- path/path.go: 0a5024c364387eeea9ab5fe0b285f0d62355e8cd181cdbddfdfa227f6916d94c
- unicode/graphic.go: 46a43107a87c88cf17a05a557e00dbf6b0f4bb53ebcd7d3035e02f1e67c428be

Other external calls remain unknown, including callback-based string operations,
reflection and formatted output. A known local body takes precedence over a
neutral external contract. Explicit observation identities retain their effect.

## Helper extraction

Parameterization proof policy 0.5.0 requires a known, non-observing summary for
both selected functions, in addition to source, type, binding and dependency
checks. A known observation or unknown call effect removes the automatic fix.
Matching syntax cannot restore a missing target or function-value fact.
This closes the explicit transitive-call gap; it does not establish the remaining
frame, exception or general transformation contracts.
