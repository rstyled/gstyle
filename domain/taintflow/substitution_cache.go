package taintflow

import "github.com/rstyled/gstyle/domain/pdg"

type substitution struct {
	Input  value
	Result value
}

func (e *evaluation) substitute(item value, call pdg.Ref) value {
	key := valueIdentity(item)
	if cached, found := e.substitutions[key]; found {
		return cached.Result
	}
	result := e.expandSubstitution(item, call)
	if e.substitutions != nil && e.substitutionArgumentsKnown(item) {
		e.substitutions[key] = substitution{Input: item, Result: result}
	}
	return result
}

func (e *evaluation) substitutionArgumentsKnown(item value) bool {
	for dep := range item.Data {
		if !e.substitutionArgumentKnown(dep.Input) {
			return false
		}
	}
	for root := range item.Objects {
		if !e.substitutionArgumentKnown(root.Input) {
			return false
		}
	}
	return true
}

func (e *evaluation) substitutionArgumentKnown(input int) bool {
	if input == 0 {
		return true
	}
	_, found := e.arguments[pdg.Position(input)]
	return found
}
