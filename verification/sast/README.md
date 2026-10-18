# SAST source baseline

Run `make verify-sast` to build the real CLI, extract typed Go AST/PDG facts,
and compare JSON and SARIF security results with the labeled corpus. The corpus
is copied into a temporary module and is never executed. The test uses no new
application or test dependencies.

`cases.json` supplies reviewed source expectations, separate from the detector.
`baseline.json` records observed rule IDs, safe/unsafe counts, known misses and
false alarms, and analysis coverage. `support` rows are helper declarations and
do not count as safe controls. CI fails for changed results, missing security
node evidence, JSON/SARIF disagreement, missing corpus functions, or stale gap
labels. Inspect a result change before running `make sast-baseline` to refresh.
Improving a known gap requires changing its expectation and removing its gap
label in the same PR. No security finding is suppressed to make the baseline pass.

## Supported boundaries

The models use compiler-resolved standard-library identities and argument
positions, not source-text matches. SQL DB/Tx/Conn Context APIs and prepare calls
track the query argument. Query parameter data is allowed. HTTP form/query/header
and selected OS/IO inputs seed per-function PDG data flow. JSON findings retain
`taint_flow` as the candidate kind and add a stable security rule ID and CWE.
SARIF uses that security ID so weakness families can be filtered separately.
These IDs intentionally replace the generic SARIF `taint_flow` ID.

## Limits

This is a small, project-owned regression corpus, not an industry benchmark or
a measured real-world detection rate. It does not execute exploits, compare
other scanners, or establish that all recorded paths are feasible. The reports
retain partial/unknown coverage; a quiet scan does not prove security.

Known gaps stay visible in the manifest: cross-function taint, request-object
URL propagation, branch allowlist validation, HTTP URL sinks, and command/template
context precision. Further coverage needs reviewed unsafe examples and safe
controls. Framework models, aliases and dynamic dispatch, authentication rules,
crypto/TLS checks, secrets, and dependency vulnerability databases are outside
this first baseline.

The changed-function guardrail scan reports global-read density at catalog
lookups and report conversion. These are read-only model tables; the code adds
no suppressions or mutation exceptions. Production complexity ceilings remain
in force.
