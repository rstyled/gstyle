# Function-length proof pilot

This project uses Lean 4, Iris and Goose/Perennial. The Go runtime has no new
external dependencies. Lean dependencies are development-only and pinned in
`lakefile.toml`, `lake-manifest.json` and `lean-toolchain`.

The production metric counts physical lines containing Go tokens. Blank lines,
comments and inserted semicolons do not count. The signature and braces count;
every physical line inside a multiline string counts. `//line` directives do not
change physical line numbers.

`domain/quality/linecount/counter.go` contains the production accumulator and
strict threshold comparison. Goose generates `Gstyle/Code` from that actual Go
package; proofgen generates `Gstyle/GeneratedProof`. Neither generated directory
is a hand-written model. `Gstyle/Proof/Length.lean` specifies the new physical
lines as a duplicate-free range and verifies the Go accumulator's boundary
checks, exact increment, unchanged rejection, signed-integer bounds and strict
threshold comparison.

The specification assumes spans arrive in source order, with the next start at
or after the preceding span's last line. The counter checks this at runtime.
Invalid spans fail source extraction instead of producing a clean report.

## Scope and trust

This is a proof of the accumulator and comparison, not the complete analyzer.
Go's scanner, token-to-span extraction, configuration validation, suppression and
diagnostic assembly remain outside this proof. Go regression tests cover that
integration. The trusted translation and Go semantics come from the pinned
Goose/Perennial revision. Its 64-bit platform model applies. Package unfolding,
struct representation, layout and Go semantics are explicit theorem hypotheses;
the audit of global axioms does not remove these hypotheses.

No new axioms, `sorry`, native decision procedures or trusted replacement
implementations are permitted in this pilot. `Audit.lean` lists the global axioms
of the production-code proofs.

## Run

Install elan and Go, then run `make verify-length`. Put elan's `bin` directory on
`PATH`. The pinned Lean version installs through elan. The first build fetches
and compiles Perennial, Iris, Batteries and Qq; later builds reuse Lake's cache.

The command checks regenerated output against the checked-in translation, checks
all nine theorem axiom lists, and retranslates four deliberately broken Go
variants. Each variant must fail in our proof module. Tool/setup failures do not
count as successful mutation rejection. Logs and the success report go to
`dist/verification/function-length/`; stale success is removed before a run.

Dependencies are checked once before mutation tests. A mutation counts as
rejected only for a recognized proof failure in its expected proof module.
Missing modules, syntax errors, unrelated diagnostics and mixed setup/proof
failures abort verification. No unchanged build is added per mutation.
