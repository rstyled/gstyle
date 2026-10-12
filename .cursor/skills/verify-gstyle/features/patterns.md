# Pattern mining

`gstyle patterns` mines Go code for 18 kinds of mechanical patterns — structural
issues with deterministic AST fixes, from guard clauses and value objects to
DDD patterns like specifications and domain services. Text output lists one
candidate per block with its observation, inference, and suggested refactor;
exit code is always 0 (informational, never a gate).

## Sub-features

- `patterns-text` reports candidates as `#N kind score [support, lift, holes, coverage]` with observation, inference, possible refactor, sites, and counter evidence.
- `patterns-kinds` covers all 18 kinds: guard_clause, value_object, parameterize, anemic_model, primitive_obsession, type_switch, enum_dispatch, trait_method, capability_set, generic_fn, entity_identity, missing_identity, mutable_identity, aggregate, repository, factory, specification, domain_service.
- `patterns-detection-only` — aggregate, repository, mutable_identity, and domain_service are detection-only (nil FixSpec); they inform but never auto-fix.
- `patterns-suppression` — domain_service suppresses the corresponding anemic_model suggestions for stateless multi-type functions.

## How to get to it (user POV)

- Run `gstyle patterns .` for the current directory.
- Run `gstyle patterns ./domain` for one package.
- Pipe to `grep` for one kind: `gstyle patterns . | grep -A5 trait_method`.

## Driving it with verify-gstyle

Preconditions: `verify-gstyle build "$RUN_DIR"` and `verify-gstyle doctor "$RUN_DIR/gstyle"` report ok; `go` on PATH.

- Craft a repeated business rule in two `if` conditions (direct boolean return
  expressions do not exercise this extractor):
  ```go
  package example

  type User struct { Age int; Active bool }

  func Allowed(u User) bool {
      if u.Age > 18 && u.Active { return true }
      return false
  }

  func Eligible(u User) bool {
      if u.Age > 18 && u.Active { return true }
      return false
  }
  ```
  Save under `$RUN_DIR/scratch/patterns/` with a `go.mod` (`module fixture`,
  `go 1.26.0`). Run from that directory; package loading uses the current module.
- Run `(cd "$RUN_DIR/scratch/patterns" && "$RUN_DIR/gstyle" patterns --format json .)`.
  Exit code is 0. The `candidates` array contains `Kind: "specification"`,
  with both functions in `Sites`. Text output also names `specification`.
- Parallel methods alone do not guarantee a `trait_method` candidate. Admission
  thresholds and current grouping rules affect that result; do not use the old
  Dog/Cat example as a fixed positive expectation.
- For a two-variable rule (`c.x != first.x`), assert no `specification` candidate (would extract with a dangling reference).
- Save stdout to `$RUN_DIR/evidence/patterns-<id>.txt` with the command and exit code.

## Gotchas

- The miner needs real type information: fixtures must be valid Go packages with a `go.mod`, or `patterns` reports load errors instead of candidates.
- `trait_method` requires 2+ distinct self types *in the same package*; the SelfTy field must be populated by the extractor (a past bug left it empty and the miner silently found nothing — if `patterns` reports 0 candidates on a fixture that should match, check SelfTy first).
- `specification` only fires on single-subject rules (one variable); multi-variable conditions are skipped by design.
- Detection-only kinds appear in `patterns` output but never in `fix` dry-runs.
