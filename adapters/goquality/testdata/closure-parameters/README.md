# Closure parameter provenance

An unreachable allocation assignment must not make a closure parameter's later caller-state write local. Nested function parameters retain parameter provenance; assignment to the variable itself remains local.

The saved line reduction is 201 bytes from 540 bytes in 117 checks. The Tree-sitter reduction is 205 bytes in 64 checks. The historical reducer checker is retired; both exact Go reductions remain here.

Run the maintained Go regressions:

```sh
go test ./adapters/goquality -run '^(TestReducedClosureParameterMutationIsExternal|TestClosureParametersAndOwnedLocalsKeepTheirProvenance)$' -count=1
```

The tests cover the reduced input, nested closures, pointer/slice parameters, value parameters and owned locals.
