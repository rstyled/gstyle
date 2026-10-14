# Unchecked-output proof

Run `make verify-output`. Go, Python 3 and elan are required. It uses the same
pinned Lean / Goose / Iris / Perennial dependencies as the other profiles.
The former Gobra pilot and Docker job are removed.

## Production contract

`domain/patterns/outputcheck` contains the translated production code.

`Discards` returns true exactly when all of these hold:

- the call is a typed stdlib `fmt.Fprint`, `fmt.Fprintf`, or `fmt.Fprintln`;
- the writer's static type is not `*bytes.Buffer` or `*strings.Builder`;
- the call is a standalone expression, or its sole result tuple is bound to two
  targets with `_` in the error slot.

The first two facts and the syntax classification come from the Go adapter.
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
model of selection. `uncheckedOutputCandidates` uses this selector directly.

The package owns the canonical Site record; `patterns.ErrorCheckSite` is an
alias. Field names, values and JSON stay unchanged. Reflection's defining
package path changes to `domain/patterns/outputcheck`.

## Pipeline and checks

Verification builds pinned dependencies once per run, regenerates the actual
Go translation, compares it with the committed translation, checks both Lean
theorems and audits their axioms. The allowed axioms are `propext`,
`Classical.choice` and `Quot.sound`. No `sorry`, native decision axioms, or
replacement implementations are admitted by this profile.

Ten Go mutants must fail the unchanged proofs: unrecognized function, memory
writer, wrong result count, wrong value count, handled error, missed standalone
call, ignored suppression, inverted selection, skipped last site and duplicated
site. Tool or dependency errors abort verification instead of passing a mutant.

Go tests cross the syntax, writer and call forms at the extraction boundary;
exercise negative and extreme decision counts; and enumerate all selection
bit sequences through length eight with both suppression values. Selection tests
also check that modifying returned records does not mutate the input. Candidate
tests cover JSON round trips, suppression, occurrence order and source metadata.

## Remaining boundaries

This is a proof of the pattern's decision and site selection, not complete
source-to-finding soundness. AST/type extraction, suppression-name lookup and
candidate rendering remain outside Lean. The adapter intentionally omits
indirect function calls, deferred calls, goroutines and nested closures in the
current function. Named error variables that are later ignored are outside this
syntactic rule. Existing Boolean facts loaded from JSON remain trusted inputs.

The Lean theorem assumes the generated package representation and unfolding
hypotheses, the pinned Go semantics, and the 64-bit model. Translation, the Go
compiler, standard-library behavior and the Lean kernel remain trust boundaries.
The Iris weakest-precondition proof establishes safety and the return contract;
it is not a separate total-correctness theorem. The finite slice loop's progress
follows from its increment and bounds, but no total-correctness claim is made.
