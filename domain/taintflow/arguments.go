package taintflow

import "github.com/rstyled/gstyle/domain/pdg"

type argumentPath struct {
	Input  int
	Fields string
}

func (e *evaluation) argumentPathValue(call pdg.Ref, input int, fields string) value {
	actual, found := e.arguments[pdg.Position(input)]
	if !found {
		return pathValue(e.state, e.actual(call, pdg.Position(input)), fields)
	}
	key := argumentPath{Input: input, Fields: fields}
	if item, found := e.argumentPaths[key]; found {
		return item
	}
	item := pathValue(e.state, actual, fields)
	if e.argumentPaths != nil {
		e.argumentPaths[key] = item
	}
	return item
}

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
