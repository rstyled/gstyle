# Parenthesized assignment targets

`((local)) = shared` must update the pointer's binding history. A later caller-state write must retain mutation or unknown evidence. `((_)) = 42` discards a value and must not create a temporary mutation.

The saved hidden-write reduction is 151 bytes from 379 bytes in 47 Tree-sitter checks. The blank-target reduction is 68 bytes from 257 bytes in 16 checks. The historical reducer checkers are retired; the exact Go inputs remain here.

Run the maintained Go regressions:

```sh
go test ./adapters/goquality -run '^(TestReducedParenthesizedAssignmentKeepsMutationEffect|TestParenthesizedPlacesMatchUnwrappedPlaces)$' -count=1
```

The tests cover pointer and slice assignment, unreachable branches, multiple assignment, owned storage and blank targets mixed with real writes. Normal repository vet checks remain enabled.
