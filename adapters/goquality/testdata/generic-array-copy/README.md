# Generic array copy falsely classified as external

`Copy[T ~[1]int]` modifies its array value copy while the caller's array stays unchanged. Array-only constraints must retain value ownership; constraints permitting slices or pointers remain conservative.

The saved Tree-sitter reduction is 111 bytes from 320 bytes in 26 checks. The historical reducer checker is retired; the exact Go inputs remain here.

Run the maintained Go regressions:

```sh
go test ./adapters/goquality -run '^(TestGenericArrayStorageUsesValueOwnership|TestReducedGenericArrayCopyIsEffectFree)$' -count=1
```

The tests cover array parameters, copies, slices of copies, unions, named and intersected constraints, and pointer/slice/mixed constraints. The generated quality harness adds six syntax variants for each supported generic-array family.
