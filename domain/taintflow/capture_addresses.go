package taintflow

import "github.com/rstyled/gstyle/domain/pdg"

func (e *evaluation) memoryValue(ref pdg.Ref, attrs pdg.Attributes) (value, bool) {
	switch e.query.graph.Tables.Text(attrs.Operator.Value) {
	case "field":
		return e.fieldValue(ref), true
	case "&":
		return e.captureAddress(ref)
	default:
		return value{}, false
	}
}
func (e *evaluation) captureAddress(ref pdg.Ref) (value, bool) {
	source := e.query.paths.Operand(ref, 1)
	path := e.query.paths.Path(source, e.query.facts)
	if len(path.Fields) != 0 {
		return value{}, false
	}
	key, found := e.query.cells[path.Root]
	if !found {
		return value{}, false
	}
	item := emptyValue()
	item.Data = e.symbolValue(path.Root).Data
	item.Objects[key.Object] = true
	return item, true
}

// Passing a captured binding by address retains every possible old/new input
// dependency. Arbitrary pointer writes cannot establish a clean overwrite.
func (e *evaluation) havocAddressArguments(ref pdg.Ref) {
	positions, _ := e.query.paths.RequiredOperands(ref)
	incoming := emptyValue()
	for _, position := range positions {
		incoming = joinValue(incoming, e.input(ref, position))
	}
	for key, old := range e.state.Memory {
		if key.Fields != "value" || !incoming.Objects[key.Object] {
			continue
		}
		next := joinValue(old, incoming)
		next.Opaque = true
		e.state.Memory[key] = next
	}
}
