# Aliasing density false negatives

The saved inputs change caller-owned state through indirect pointer replacement, range assignment and an allocated wrapper with an embedded pointer. The affected analyzer incorrectly reports complete coverage with zero density and no effects.

Original inputs are 533, 536 and 528 bytes. Their saved reductions are 161, 164 and 156 bytes, from 135, 117 and 117 checks respectively. The historical reducer checker is retired; exact Go inputs remain here.

Run the maintained Go regressions:

```sh
go test ./adapters/goquality -run '^(TestReducedAliasingMutationsRemainVisibleInQuality|TestAliasingProvenancePreservesOwnedStorage)$' -count=1
```

The tests retain uncertainty for exposed or mixed references and verify owned-storage controls. These are bounded cases, not an interprocedural ownership proof.
