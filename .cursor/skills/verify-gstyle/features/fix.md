# Pattern autofix

`gstyle fix` applies deterministic AST transforms for the mechanical pattern
kinds found by `gstyle patterns`. Dry-run by default (prints a diff); `--apply`
writes the changes. No LLM, no tokens — pure static analysis. `--phased`
re-mines between phases to a fixpoint; `--changed` scopes to git-touched
functions; `--kind` selects one kind.

## Sub-features

- `fix-dry-run` prints the diff without writing; exit 0 even when candidates exist.
- `fix-apply` writes the transform; the result is gofmt-clean and the package still builds.
- `fix-kind` selects one kind: `--kind specification`, `--kind trait_method`, etc.; `--kind all` is the default.
- `fix-phased` runs mine → apply phase → re-mine to a fixpoint (max 3 cycles); phases order guard/value-object work first, structural patterns next, parameterize last.
- `fix-changed` with `--apply` fixes only candidates in functions the git diff touches (`--base REF`, default merge-base with main/master).
- `fix-idempotent` — re-running `fix --apply` after a successful fix is a no-op (the fixer skips already-applied transforms).

## How to get to it (user POV)

- Run `gstyle fix --kind specification .` to preview one kind.
- Run `gstyle fix --kind specification --apply .` to write it.
- Run `gstyle fix --phased --apply .` for interacting fixes.
- Run `gstyle fix --changed --apply .` in a git repo to fix only your diff.

## Driving it with verify-gstyle

Preconditions: `verify-gstyle build "$RUN_DIR"` and `verify-gstyle doctor "$RUN_DIR/gstyle"` report ok; `go` on PATH. Run fixture commands from the fixture directory; absolute paths outside the current Go module do not load as packages.

- Craft a fixture with a repeated boolean rule in two `if` conditions (see `patterns.md`). Save under `$RUN_DIR/scratch/fix/`.
- Dry-run: `$RUN_DIR/gstyle fix --kind specification $RUN_DIR/scratch/fix` — assert stdout shows the diff, the fixture file is byte-identical afterward, exit 0.
- Apply: `$RUN_DIR/gstyle fix --kind specification --apply $RUN_DIR/scratch/fix` — assert the named predicate method `IsAgeOver18AndActive` was added to `User`, both conditions call it, and exit code is 0. No Specification struct is generated.
- Build check: `go build ./...` in the fixture dir must succeed after apply.
- Idempotency: run the apply command again — assert exit 0 and the file is byte-identical (no-op).
- For `trait_method`, first require an actual candidate from mining. Parallel methods alone do not guarantee one. When admitted, assert the generated interface is gofmt-clean and the package builds.
- For `--changed`: init a temp git repo in `$RUN_DIR/scratch`, commit a baseline, add an unfixed pattern in an uncommitted edit, run `$RUN_DIR/gstyle fix --changed --apply` — assert only the touched function was fixed.
- Save the before/after diff to `$RUN_DIR/evidence/fix-<id>.diff` with the command and exit code.

## Gotchas

- The fixer inserts new types after the import *declaration*, not the last import spec — a past bug inserted inside parenthesized import blocks and broke the file. If `go build` fails after apply with import errors, check the insertion point.
- `specification` extraction renames the condition's variable to the `IsSatisfiedBy` parameter; two-variable rules are rejected at mine time, but verify the generated body compiles.
- `--changed` needs a real git repo with at least one commit; untracked files count as fully changed.
- Detection-only kinds (`domain_service`, `aggregate`, `repository`, `mutable_identity`) never appear in fix output — asserting their absence is correct.
- Always `go build` the fixture after `--apply`; a gofmt-clean file can still have scoping errors (e.g. a referenced variable that lived outside the extracted rule).
