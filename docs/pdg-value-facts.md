# Canonical value facts

`gstyle.go:value-fact` records a compiler-owned leaf value or conversion
contract on an existing canonical node. Each record contains exactly one field:
`nil: true`, `non_nil: true`, `boolean: true|false`, or `conversion` with `identity`, `nil`,
`boxed`, `nilness`, or `unknown`. Duplicate records, duplicate fields, invalid
JSON, nulls, unknown fields and invalid enum values fail the binding query.

The binding query reads these facts for call exits, resource paths and nil-error
checks. Legacy runtime nil/boolean records remain readable when a canonical
record is absent. Identical legacy records are accepted; conflicting values fail.
Compiler constants are leaf facts, not another dependency graph. Conversion
inputs still come from canonical operand edges. Identity transfers the actual
operand; nil conversion requires its nil constant; boxing proves a non-nil
interface; nilness transfers only whole-value nilness. Storage identity does not
follow boxing or nilness-only conversions.

Safe identity, nil and nilness-preserving conversions have no local panic effect.
This says nothing about operand evaluation. Slice-to-array conversions can panic
and stay outside that set. Generic, unsafe and nonpreserving conversion behavior
remains conservative. Scalar constants and fixed aggregate shapes used only by
the nil-error adapter retain their typed leaf boundary.

The source regressions cover folded and named constant branches, recovered
interface values, nil interfaces, boxed nil pointers, named pointer/slice
conversions, and a nonpreserving slice/string conversion. These tests establish
this contract; they do not establish complete Go execution semantics or finish
the full PDG migration.

Conditional constructor non-nilness and payload dependency rules are recorded in
`pdg-call-packing.md`. A known non-nil shape is not a complete constant payload.
