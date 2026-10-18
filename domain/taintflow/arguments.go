package taintflow

import "github.com/rstyled/gstyle/domain/pdg"

// A variadic formal receives the union of its actual elements. This is a
// collection summary, not an index-sensitive slice-content proof.
func (e *evaluation) actual(call pdg.Ref, position pdg.Position) value {
	if item, found := e.arguments[position]; found {
		return item
	}
	callee := e.catalog[e.target(call)]
	if !variadicPosition(callee, position) {
		return e.input(call, position)
	}
	result := emptyValue()
	positions, _ := e.query.paths.RequiredOperands(call)
	for _, actual := range positions {
		if actual >= position {
			result = joinValue(result, e.input(call, actual))
		}
	}
	return result
}
func variadicPosition(callee *query, position pdg.Position) bool {
	if callee == nil || len(callee.graph.Function.Inputs) == 0 {
		return false
	}
	inputs := callee.graph.Function.Inputs
	last := inputs[len(inputs)-1]
	return int(position) == len(inputs) && last.Variadic.Status == pdg.Known && last.Variadic.Value
}
