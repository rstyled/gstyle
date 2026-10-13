package gopatterns

// NativePolicy registers the extraction contract in each conformance run.
// The root profile, not per-record evidence, selects these semantics.
type NativePolicy struct {
	ID, Version, CountingRule, TypeTransport, NormalFlow, ExceptionalFlow string
	RequiredValidation                                                    []string
}

func nativePolicy() NativePolicy {
	return NativePolicy{
		ID: ScalarNormalProfileID, Version: NativeExtractionVersion,
		CountingRule:       "Distinct local/parameter symbol IDs read by retained read operations; writes alone do not count.",
		TypeTransport:      "gstyle.go.scalar-normal:resolved-type on symbols and interface elements, checked against go/types; aliases/named types resolve to underlying primitive types. Native width comes from the loaded package target.",
		NormalFlow:         "Acyclic integer/bool scalar operations, actual inputs and meaningful output entities; explicit branches for short circuit; reject unspecified sibling-operand evaluation orders. Preserve source operations and nonexecuting value joins without collapsing bindings.",
		ExceptionalFlow:    "Panic/unwinding omitted under a bounded normal-flow policy; paired CFG evidence stays partial.",
		RequiredValidation: []string{"source snapshot identity", "source inventory/kinds/operators/literals", "binding scopes/roles/reads/writes/types", "source/CFG entries/transitions/outcomes", "execution/control dependency completeness", "reaching definitions/value joins", "operand/return dependencies", "complete resolved interface", "projection/reference-count agreement", "known build/target"},
	}
}
