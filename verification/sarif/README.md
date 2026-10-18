# SARIF support and verification

Gstyle supports SARIF 2.1.0 through `check --format sarif` and
`patterns --format sarif`. The OASIS standard defines a shared JSON format
for static analysis results. This lets CI systems consume Gstyle findings
without a custom parser.

## Contract

- Each report contains one run with the `gstyle` driver.
- Categories `gstyle/check/` and `gstyle/patterns/` distinguish the two analyses.
- `%SRCROOT%` records the current working directory for relative path resolution.
  Run from the repository root, including when loading saved facts.
- Each quality diagnostic becomes one warning with its existing rule ID and message.
- Each published pattern candidate becomes one warning. Its kind is the rule ID;
  its observation, inference, and proposed refactor form the message.
- The first available site is the primary location. Other sites are related locations.
- Paths are URI-escaped. Absolute paths use file URIs; relative paths stay relative.
- Locations use positive source line numbers. Unknown lines and columns are omitted.
  Go byte columns cannot be copied into SARIF's Unicode/UTF-16 column units.
- Empty reports use an empty results array. IO errors propagate to the CLI.
- Existing exit codes and filtering remain in effect. SARIF is an output format,
  so it does not change analysis or authorize fixes.

## Schema provenance

`schema.json` is the full, unchanged OASIS SARIF 2.1.0 Plus Errata 01 schema,
retrieved from the OASIS technical committee repository:

https://github.com/oasis-tcs/sarif-spec/blob/main/sarif-2.1/schema/sarif-schema-2.1.0.json

The schema identifies its official location as:

https://docs.oasis-open.org/sarif/sarif/v2.1.0/errata01/os/schemas/sarif-schema-2.1.0.json

SHA-256: `c3b4bb2d6093897483348925aaa73af03b3e3f4bd4ca38cef26dcb4212a2682e`

Standard: https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html

## Verification

Run `make verify-sarif` with Go and uv installed. The test-only validator is
pinned; the Go application adds no dependencies. CI runs this check.

The script builds the actual CLI, runs both commands against clean and failing
Go source, validates every report with the full Draft 4 schema and URI format
checks, checks rule IDs and escaped paths, and verifies the saved-facts path.
Negative controls confirm the validator rejects invalid versions and missing
driver data. Go unit tests check related locations, unknown lines, empty
reports, preserved messages, and writer errors.

Schema validation proves the format contract. It does not prove that each
analysis finding is correct or that a particular hosted service accepts an
upload; those remain separate analysis and service contracts.

## Review results

The changed-function quality scan flags side-effect density in the CLI output
boundaries. These functions run analysis or write reports; moving their IO into
more wrappers would hide the same work. The SARIF adapter has no mutation or
aggregate-rule findings. No rule suppressions are added. The repository baseline
is regenerated with this change.

## GitHub code scanning

The [GitHub SARIF support guide](https://docs.github.com/en/code-security/reference/code-scanning/sarif-files/sarif-support)
is an additional consumer contract. The CLI test checks the driver, category,
source-root base URI, nonempty IDs and messages, and primary source locations.
These checks cover our output contract; they do not run the complete Microsoft
GitHub ingestion validator or perform a hosted upload.

Run from the repository root. Upload the two reports with distinct categories:

```yaml
- uses: github/codeql-action/upload-sarif@v4
  with:
    sarif_file: quality.sarif
    category: gstyle/check
- uses: github/codeql-action/upload-sarif@v4
  with:
    sarif_file: patterns.sarif
    category: gstyle/patterns
```

The upload action can generate missing `partialFingerprints` from checked-out
source. Gstyle does not emit fingerprints. A direct `/code-scanning/sarifs` API
upload should add them first to avoid duplicate alerts. GitHub has file-size
and result-count limits; Gstyle preserves all findings and does not truncate
reports to fit those limits. A location outside the repository or through a
symlink may need path normalization before GitHub can display it.
