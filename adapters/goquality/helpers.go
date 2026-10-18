package goquality

import (
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain/quality"
)

func attachHelpers(facts, available []quality.Function) []quality.Function {
	index := map[string]quality.Function{}
	for _, function := range available {
		index[quality.NormalizeCallee(function.Name)] = function
	}
	readOnly := readOnlyHelpers(index)
	for name, function := range index {
		function.Calls = sourceCalls(function.Calls, readOnly)
		index[name] = function
	}
	for position, function := range facts {
		body := index[quality.NormalizeCallee(function.Name)]
		facts[position].Calls = body.Calls
		facts[position].Helpers = reachableHelpers(body, index)
	}
	return facts
}

func reachableHelpers(function quality.Function, index map[string]quality.Function) []quality.Helper {
	seen := map[string]bool{quality.NormalizeCallee(function.Name): true}
	pending := slices.Clone(function.Calls)
	var helpers []quality.Helper
	for len(pending) > 0 {
		call := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if !call.Local || call.Dynamic {
			continue
		}
		name := quality.NormalizeCallee(call.Callee)
		target, found := index[name]
		if !found || seen[name] {
			continue
		}
		seen[name] = true
		helpers = append(helpers, quality.Helper{Name: target.Name, Mutations: target.Mutations, Calls: target.Calls, Effects: target.Effects})
		pending = append(pending, target.Calls...)
	}
	slices.SortFunc(helpers, func(a, b quality.Helper) int { return strings.Compare(a.Name, b.Name) })
	return helpers
}

// Cross-package calls use only source bodies proven read-only. Evidence stays
// intact for policy reevaluation. Other package calls remain conservative.
func sourceCalls(calls []quality.Call, readOnly map[string]bool) []quality.Call {
	result := slices.Clone(calls)
	for i, call := range result {
		available := readOnly[quality.NormalizeCallee(call.Callee)]
		if available && !call.Dynamic {
			result[i].Local = true
		}
	}
	return result
}

// The finite fixed point admits only bodies with no observable mutations,
// intrinsic effects, dynamic calls or unresolved callees. Cycles stay unknown.
func readOnlyHelpers(index map[string]quality.Function) map[string]bool {
	proven := map[string]bool{}
	for changed := true; changed; {
		changed = false
		for name, body := range index {
			if proven[name] {
				continue
			}
			if !readOnlyBody(body, proven) {
				continue
			}
			proven[name] = true
			changed = true
		}
	}
	return proven
}
func readOnlyBody(body quality.Function, proven map[string]bool) bool {
	if len(body.Effects) != 0 {
		return false
	}
	if observableMutations(body.Mutations) {
		return false
	}
	return readOnlyCalls(body.Calls, proven)
}
func observableMutations(mutations []quality.Mutation) bool {
	for _, mutation := range mutations {
		if mutation.Provenance != quality.Local {
			return true
		}
	}
	return false
}
func readOnlyCalls(calls []quality.Call, proven map[string]bool) bool {
	for _, call := range calls {
		known := proven[quality.NormalizeCallee(call.Callee)] || quality.ContainerClone(call.Callee) || call.HasLocalContainerTarget()
		if call.Dynamic || !known {
			return false
		}
	}
	return true
}
