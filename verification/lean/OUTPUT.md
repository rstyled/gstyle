# Unchecked-output proof

Run `make verify-output`. Go, Python 3 and elan are required. It uses the same
pinned Lean / Goose / Iris / Perennial dependencies as the other profiles.
The former Gobra pilot and Docker job are removed.

## Production contract

`domain/patterns/outputcheck` contains the translated production code.

`Formatted` accepts exactly package `fmt` and one of `Fprint`, `Fprintf`, or
`Fprintln`. `Memory` accepts exactly the pairs `(bytes, Buffer)` and
`(strings, Builder)`. These policies operate on separate package and type/name
strings; they do not accept a matching short name from another package. Their
Lean contracts cover all strings. The adapter supplies a callee identity only
for a resolved function, and a writer identity only for a pointer to a named
type. Unsupported shapes become empty identities.

Given those classified facts, `Discards` returns true exactly when all of these hold:

- the call is a typed stdlib `fmt.Fprint`, `fmt.Fprintf`, or `fmt.Fprintln`;
- the writer's static type is not `*bytes.Buffer` or `*strings.Builder`;
- the call is a standalone expression, or its sole result tuple is bound to two
  targets with `_` in the error slot.

`Classify` composes the identity policies and discard decision over those raw
identities and consumption facts. It also checks the actual error-target name
against `_`. Its theorem covers all identity/name strings, consumption flags and
64-bit counts. The adapter no longer assembles the Formatted, Memory or ErrorBlank
policy Booleans. It supplies type identities, syntax shape, counts and target
name; the rule's classification composition executes in the proved package.
The Lean decision theorem covers every Boolean input and every 64-bit count,
including negative and extreme counts, without assuming inputs are valid counts.
Two results means the selected fmt functions' `(int, error)` tuple; it is not
a general rule for every Go function that returns an error.

`Select` returns no sites when suppressed. Otherwise it returns the exact stable
filter of the supplied sites by `DiscardedOutput`. The loop proof covers any
owned input slice, preserves that input, and returns each selected occurrence
once, with all fields unchanged. Equal occurrences remain separate. The proof
tracks the processed prefix, proves each index is in bounds, and establishes
that the loop counter does not overflow. It covers the loop, not a replacement
model of selection.

`hasSuppression` returns true exactly when `unchecked_output` occurs in the
suppression-name slice. Its loop proof checks every name through the last slot,
keeps indexes in bounds and preserves that slice. `Find` composes the lookup and
selection proofs over the actual two borrowed slices. It returns the stable
filter exactly when that name is absent. `Report` calls `Find` directly; the
candidate adapter no longer supplies an unproved suppression Boolean.

`describe` builds the exact advisory payload: the owning function's path, ID and
name, the call's line in both line fields, kind `unchecked_output`, score 800,
the callee-bearing observation, inference and counter-evidence text. `Report`
composes `Find` with a proved mapping loop. Its output is exactly the ordered map
of those payloads over the selected records, with both inputs preserved. The
payload type has no automatic-fix field. Production candidate construction calls
`Report` for each function. The outer corpus traversal, per-function input
projection and projection into the common Candidate type remain outside Lean.

The package owns the canonical Site record; `patterns.ErrorCheckSite` is an
alias. Field names, values and JSON stay unchanged. Reflection's defining
package path changes to `domain/patterns/outputcheck`.

## Pipeline and checks

Verification builds pinned dependencies once per run, regenerates the actual
Go translation, compares it with the committed translation, checks all nine Lean
theorems and audits their axioms. The allowed axioms are `propext`,
`Classical.choice` and `Quot.sound`. No `sorry`, native decision axioms, or
replacement implementations are admitted by this profile.

Thirty-four Go mutants must fail the unchanged proofs. They cover identity and
discard policies, suppression lookup and composition, skipped/duplicated selected
sites, every location/owner field, kind, score, missing callee text, and
skipped/duplicated report payloads, crossed classifier identities, swapped
consumption counts, lost consumption form and incorrect blank-target handling.
Tool or dependency errors abort verification
instead of passing a mutant.

Go tests cross the syntax, writer and call forms at the extraction boundary;
exercise negative and extreme decision counts; and enumerate all selection
bit sequences through length eight with both suppression values. Selection tests
also check that modifying returned records does not mutate the input. Candidate
tests cover JSON round trips, suppression, occurrence order and source metadata.
Suppression tests include absent names, the exact name in the first/middle/last
slot, repeated names and similar names that must not match. Extraction regressions
include nested parentheses around a blank error target and `//line` directives
that must not replace a call's physical source line. Projection tests compare
the common Candidate against the proved payload and check that it adds no fix,
refactor, analysis or unrelated evidence.

## Remaining boundaries

This is a proof of the pattern's decision and fact-to-advisory pipeline, not
complete source-to-finding soundness. AST/type extraction, outer corpus traversal,
input and common Candidate projections, and final presentation remain outside
Lean. The adapter intentionally omits
indirect function calls, deferred calls, goroutines and nested closures in the
current function. Named error variables that are later ignored are outside this
syntactic rule. Existing Boolean facts loaded from JSON remain trusted inputs.

The Lean theorem assumes the generated package representation and unfolding
hypotheses, the pinned Go semantics, and the 64-bit model. Translation, the Go
compiler, standard-library behavior and the Lean kernel remain trust boundaries.
The Iris weakest-precondition proof establishes safety and the return contract;
it is not a separate total-correctness theorem. The finite slice loop's progress
follows from its increment and bounds, but no total-correctness claim is made.

Each mutant builds only its affected proof module. The full unchanged project is
checked before mutation tests, so unrelated pattern proofs are not rebuilt for
every mutant.

The outer input projection matches a two-statement function body: a discarded
output call followed directly by a return of the universe nil in the function's error result. Unknown callback
outcomes and functions without an error result stay quiet. Named results with
deferred calls are excluded because a defer can replace nil. Raw discard facts
remain available. This source filter and its occurrence-index metadata are tested
in Go; they are outside the Lean proof.
