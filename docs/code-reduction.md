# First behavior-preserving code reduction

Base: master `dc1e449b4b2d54ce99b7de7378ab00d924f85c5b`.
Implementation: `d49a4edb144a75ff817fd973a9827e21e4a18444`.

The command statechart separates quality evaluation, pattern mining and fix
application. Source review found an obsolete AST differ beside the active
anti-unification path, and two copies of the extraction-to-miner conversion.
The quality and targeted pattern reports locate the review area; neither is
used as permission to delete behavior or weaken proof checks.

## What changes

- Remove 47 private functions unreachable from all repository programs and test
  entry points, plus two private record types left by the older state helpers.
  The unused AST differ accounts for 461 lines. Retain exported helpers even
  when the command reachability report does not use them.
- Share report/fix fact construction at the adapter boundary. Preserve their
  different evidence profiles, nil/empty array shapes, graph identity and copied
  hit storage. Adapter hit names remain distinct Go types with the domain's
  field shapes; their public type identities are tested.
- Keep source-revision, compiler-overlay, comment, parameterization and write
  restoration checks. No command, rule limit, suppression or dependency changes.

## Size and behavior

Tracked production Go decreases **78,332 → 77,623 physical lines** (709 removed)
and **71,515 → 70,859 nonblank lines** (656 removed). These counts include
examples and Go verification commands, and separate `_test.go` and `testdata`.
The existing test cases remain; 67 lines of boundary tests are added. This is
about a 0.9% production reduction, not a claim that most remaining code is dead.

On identical frozen master source, the old and new binaries produce byte-for-
byte identical quality reports (5,227 functions / 650 diagnostics) and targeted
repository pattern reports (861 functions in coverage / 14 candidates).
The finite CLI corpus also matches all **55 command results**, including JSON
and graph export, `next`, `score`, dry-run and applied fixes, rewritten files,
the next apply run, and invalid-source exit 2. This checks actual outputs rather
than assuming that fewer self-findings means a better analyzer.

On the smaller changed repository, the quality scan sees 5,178 functions and
649 findings. Density excess drops 258,744 → 258,444. Excess rates can rise when
pure/dead functions leave the denominator; no tolerance is widened. The new
conversion code has no quality diagnostics. The baseline is regenerated.

## Verification and limits

`deadcode -test -json ./...` from x/tools v0.50.0 supplies reachability evidence;
source references, compilation and tests check the private deletion set. A
tool-only telemetry dependency is resolved with a separate cached modfile;
the project's module files are unchanged. No exported API is removed based
only on command reachability.

Required local checks are `make check`, `make quality-gate`, `make lint cognitive
gstyle-check`, and affected adapter/application/domain race tests. Existing
golden and caller-mutation runtime harness source are retained.

The evidence archive records the differential corpus, deletion lists, hashes,
test logs and projection profiles. Full self-source quality reports remain in
the task cache because each is 521 MB; their identical SHA-256 is in the
manifest. The targeted pattern report is preserved. Full repository pattern
mining did not finish within the prior audit budget, so no full-pattern clean
result is claimed. Finite fixtures and reachability are not a soundness proof.

To repeat the CLI comparison, build the base and implementation with
`go build -trimpath -buildvcs=false`, then run the archived `compare_cli.py`
once with `before BINARY` and once with `after BINARY`. Its `ROOT` and `OUT`
constants identify the frozen source and scratch evidence directories; adjust
those two paths for another machine. The script modifies only scratch fixtures.

The false-positive patrol and recovery timer remain paused. Ownership PR #242
remains separate and unmerged. Larger API pruning needs an explicit compatibility
boundary; it is not smuggled into this first reduction.
