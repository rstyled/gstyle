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
	bounded := boundSourceHelpers(index)
	for name, function := range index {
		function.Calls = sourceCalls(function.Calls, bounded)
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
		helpers = append(helpers, quality.Helper{Name: target.Name, Params: target.Params, HasSelf: target.HasSelf, Mutations: target.Mutations, Calls: target.Calls, Effects: target.Effects})
		pending = append(pending, target.Calls...)
	}
	slices.SortFunc(helpers, func(a, b quality.Helper) int { return strings.Compare(a.Name, b.Name) })
	return helpers
}

// Cross-package bodies retain panics, known writes, and writes bound to formals.
// Complete evidence stays available for policy reevaluation.
func sourceCalls(calls []quality.Call, available map[string]bool) []quality.Call {
	result := slices.Clone(calls)
	for i, call := range result {
		found := available[quality.NormalizeCallee(call.Callee)]
		if found && !call.Dynamic {
			result[i].Local = true
		}
	}
	return result
}

// Bounded recursive groups retain their complete source evidence. The quality
// evaluator still checks effects and project policy before proving read-only.
func boundSourceHelpers(index map[string]quality.Function) map[string]bool {
	proven := map[string]bool{}
	for name := range index {
		proven[name] = true
	}
	for changed := true; changed; {
		changed = false
		for name, body := range index {
			if !proven[name] || boundSourceBody(body, proven) {
				continue
			}
			delete(proven, name)
			changed = true
		}
	}
	return proven
}

func boundSourceBody(body quality.Function, proven map[string]bool) bool {
	for _, effect := range body.Effects {
		if effect.Kind != quality.Panic {
			return false
		}
	}
	if unboundMutations(body.Mutations) {
		return false
	}
	return boundSourceCalls(body.Calls, proven)
}
func unboundMutations(mutations []quality.Mutation) bool {
	for _, mutation := range mutations {
		if mutation.Provenance == quality.Unknown && mutation.Parameter == nil {
			return true
		}
	}
	return false
}
func boundSourceCalls(calls []quality.Call, proven map[string]bool) bool {
	for _, call := range calls {
		if !boundSourceCall(call, proven) {
			return false
		}
	}
	return true
}

func boundSourceCall(call quality.Call, proven map[string]bool) bool {
	if quality.ClassifyCallee(call.Callee) == quality.None || call.LocalMemoryRead {
		return true
	}
	if call.Dynamic {
		return false
	}
	return proven[quality.NormalizeCallee(call.Callee)] || sourceContainerCall(call) || call.HasPlainJSON()
}

func sourceContainerCall(call quality.Call) bool {
	if quality.ContainerClone(call.Callee) {
		return true
	}
	if call.MutationTarget == "" || len(call.StorageArgs) == 0 {
		return false
	}
	return call.HasLocalContainerTarget() || call.StorageArgs[0].Parameter != nil
}
