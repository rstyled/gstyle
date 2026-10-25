# Promoted-pointer quality false negative

Gstyle at commit `4fa9285` classifies a promoted-field write through a local value wrapper as local even when an embedded pointer reaches caller-owned state. The saved reducer input is 25 lines / 541 bytes; the result is 6 lines / 134 bytes after 111 checks.

The historical reducer checker is retired. The exact Go inputs remain here and the maintained regression runs in Go:

```sh
go test ./adapters/goquality -run '^(TestReducedPromotedPointerMutationHasQualityEvidence|TestSelectorMutationProvenanceRespectsPointerIndirection)$' -count=1
```

The tests require mutation evidence for shared pointer paths and cover nested promotion, addresses, explicit pointer fields, aliases, value copies and pointer parameters. Embedded values and owned storage retain local provenance.
