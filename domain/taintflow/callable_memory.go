package taintflow

import "github.com/rstyled/gstyle/domain/pdg"

func (e *evaluation) mutableCallableReferences(ref pdg.Ref) value {
	if !e.sortsSlice(ref) {
		return e.callableReferences(ref)
	}
	references := e.targetValue(ref)
	positions, _ := e.query.paths.RequiredOperands(ref)
	for _, position := range positions {
		item := e.input(ref, position)
		if position == 2 && e.callablePreservesMemory(item) {
			continue
		}
		if e.callableInput(ref, position, item) {
			references = joinValue(references, item)
		}
	}
	return references
}

func (e *evaluation) sortsSlice(ref pdg.Ref) bool {
	switch e.target(ref) {
	case "sort.Slice", "sort.SliceStable":
		return true
	default:
		return false
	}
}

func (e *evaluation) callablePreservesMemory(item value) bool {
	if item.Opaque || len(item.Functions) == 0 {
		return false
	}
	for name := range item.Functions {
		target := e.catalog[name]
		if target == nil || !target.memoryPreserved {
			return false
		}
	}
	return true
}
