package taintflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
	"strings"
)

// An opaque call can return reachable inputs and can invoke supplied closures.
// Wildcard writes are may-effects; they never establish a clean overwrite.
func (e *evaluation) unknownCall(ref pdg.Ref) value {
	target := e.targetValue(ref)
	result := joinValue(e.externalCall(ref), target)
	result = joinValue(result, encodeValue(callbackValue(e.state, e.callableReferences(ref)), ""))
	e.havocCallableInputs(ref, result)
	return result
}
func (e *evaluation) callableReferences(ref pdg.Ref) value {
	references := e.targetValue(ref)
	positions, _ := e.query.paths.RequiredOperands(ref)
	for _, position := range positions {
		item := e.input(ref, position)
		if e.callableInput(ref, position, item) {
			references = joinValue(references, item)
		}
	}
	return references
}
func (e *evaluation) havocCallableInputs(ref pdg.Ref, incoming value) {
	reachable := reachableStore(e.state, e.callableReferences(ref))
	for root := range reachable.Objects {
		key, affected := callbackCell(root)
		if !affected {
			continue
		}
		next := joinValue(readCell(e.state, key), incoming)
		next.Opaque = true
		e.state.Memory[key] = next
	}
}
func (e *evaluation) callableInput(ref pdg.Ref, position pdg.Position, item value) bool {
	if len(item.Functions) != 0 {
		return true
	}
	kind := e.query.paths.MemoryKind(e.query.paths.Operand(ref, position))
	return kind == "function" || kind == "interface"
}

// A pending callback effect is not a general heap write. Formal callable values
// retain it until a caller supplies an environment with captured cells.
const callbackEffect = "callback:*"

func callbackCell(root object) (cell, bool) {
	if root.Captured {
		return cellKey(root, "*"), true
	}
	if root.Input != 0 {
		return cellKey(root, callbackEffect), true
	}
	return cell{}, false
}
func (e *evaluation) callbackTargets(base value, prefix string) []cell {
	reachable := reachableStore(e.state, pathValue(e.state, base, prefix))
	var result []cell
	for root := range reachable.Objects {
		if target, affected := callbackCell(root); affected {
			result = append(result, target)
		}
	}
	return result
}

func callbackValue(store state, base value) value {
	reachable := reachableStore(store, base)
	result := emptyValue()
	for root := range reachable.Objects {
		if root.Captured {
			maps.Copy(result.Data, readCell(store, cellKey(root, "value")).Data)
			continue
		}
		if root.Input != 0 {
			result.Data[atom{Input: root.Input, Fields: appendFields(root.Fields, callbackEffect)}] = true
		}
	}
	return result
}

func callbackMarker(fields string) bool {
	return fields == callbackEffect || strings.HasSuffix(fields, "/"+callbackEffect)
}
