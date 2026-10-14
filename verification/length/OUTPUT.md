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
Lean contracts cover all strings. The adapter calls these functions after it
resolves the callee to a function and the writer to a pointer to a named type.

Given those classified facts, `Discards` returns true exactly when all of these hold:

- the call is a typed stdlib `fmt.Fprint`, `fmt.Fprintf`, or `fmt.Fprintln`;
- the writer's static type is not `*bytes.Buffer` or `*strings.Builder`;
- the call is a standalone expression, or its sole result tuple is bound to two
  targets with `_` in the error slot.

The Go adapter supplies type identities and syntax classification. The identity
policies and Boolean decision execute in the proved package.
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
filter exactly when that name is absent. `uncheckedOutputCandidates` calls
`Find` directly; it no longer supplies an unproved suppression Boolean.

The package owns the canonical Site record; `patterns.ErrorCheckSite` is an
alias. Field names, values and JSON stay unchanged. Reflection's defining
package path changes to `domain/patterns/outputcheck`.

## Pipeline and checks

Verification builds pinned dependencies once per run, regenerates the actual
Go translation, compares it with the committed translation, checks all six Lean
theorems and audits their axioms. The allowed axioms are `propext`,
`Classical.choice` and `Quot.sound`. No `sorry`, native decision axioms, or
replacement implementations are admitted by this profile.

Eighteen Go mutants must fail the unchanged proofs: unrecognized function, memory
writer, wrong result count, wrong value count, handled error, missed standalone
call, ignored suppression, inverted selection, skipped last site and duplicated
site, wrong function package, omitted Fprintln, wrong writer package, and a
crossed writer identity, wrong suppression name, inverted suppression matching,
skipped last suppression name, and bypassed suppression lookup. Tool or dependency
errors abort verification instead of passing a mutant.

Go tests cross the syntax, writer and call forms at the extraction boundary;
exercise negative and extreme decision counts; and enumerate all selection
bit sequences through length eight with both suppression values. Selection tests
also check that modifying returned records does not mutate the input. Candidate
tests cover JSON round trips, suppression, occurrence order and source metadata.
Suppression tests include absent names, the exact name in the first/middle/last
slot, repeated names and similar names that must not match. Extraction regressions
include nested parentheses around a blank error target.

## Remaining boundaries

This is a proof of the pattern's decision and site selection, not complete
source-to-finding soundness. AST/type extraction and candidate rendering remain
outside Lean. The adapter intentionally omits
indirect function calls, deferred calls, goroutines and nested closures in the
current function. Named error variables that are later ignored are outside this
syntactic rule. Existing Boolean facts loaded from JSON remain trusted inputs.

The Lean theorem assumes the generated package representation and unfolding
hypotheses, the pinned Go semantics, and the 64-bit model. Translation, the Go
compiler, standard-library behavior and the Lean kernel remain trust boundaries.
The Iris weakest-precondition proof establishes safety and the return contract;
it is not a separate total-correctness theorem. The finite slice loop's progress
follows from its increment and bounds, but no total-correctness claim is made.
