# Real rstyle capture fixture

Generated from `../rstyle-module` using the revisions in `../../rstyle-versions.json`.
The module has two functions and a real mutation-per-target finding: two updates
of `y`, against the fixture's configured limit of one. Its detail lines are 6 and 7.

From the repository root, regenerate into a new directory:

```sh
rstyle --cpg-compression none go \
  --source "$PWD/tools/pdg-viewer/testdata/rstyle-module" \
  --out /tmp/new-rstyle-viewer-capture --format json > /tmp/rstyle-viewer-report.json
# Exit 1 is expected: the fixture has one finding.
```

Copy the sealed `documents/` directory and the stdout report into this fixture.
Remove the report's machine-specific `run` field. Keep document bytes unchanged:
the manifest checksums cover the exact bytes. The full report intentionally
retains runtime bounds even though the viewer requests only check, patterns and
flow in production. Tests read only the requested fields.
