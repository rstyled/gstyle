package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
)

// A family includes all cells allocated at a lexical binding site in this
// activation. History joins never discard an older cell retained by a closure.
func (evaluation *exitEvaluation) nextCellValues(old map[pdg.Ref]exitValue, writes []bindingflow.ValueWrite) map[pdg.Ref]exitValue {
	result := maps.Clone(old)
	if result == nil {
		result = map[pdg.Ref]exitValue{}
	}
	for _, write := range writes {
		if write.UnknownTarget() {
			return unknownCellValues(result)
		}
		if _, tracked := result[write.Symbol]; write.Location != 0 || !tracked {
			continue
		}
		value := abstractValue(write.Value)
		result[write.Symbol] = joinCellValue(result, write.Symbol, value)
	}
	return result
}

func unknownCellValues(old map[pdg.Ref]exitValue) map[pdg.Ref]exitValue {
	result := map[pdg.Ref]exitValue{}
	for symbol := range old {
		result[symbol] = unknownValue
	}
	return result
}

func (evaluation *exitEvaluation) initialCellValues(bindings map[pdg.Ref]exitValue) map[pdg.Ref]exitValue {
	result := map[pdg.Ref]exitValue{}
	for symbol := range evaluation.body.facts.CapturedSymbols {
		if evaluation.body.graph.Symbols[symbol-1].Role != pdg.CaptureRole {
			continue
		}
		result[symbol] = bindings[symbol]
	}
	return result
}

func joinCellValue(values map[pdg.Ref]exitValue, symbol pdg.Ref, next exitValue) exitValue {
	previous, found := values[symbol]
	if found && previous != next {
		return unknownValue
	}
	return next
}

// Local families become observable only when a closure retains a cell. Earlier
// writes cannot affect a closure that does not yet exist.
func (evaluation *exitEvaluation) captureValues(point exitPoint, ref pdg.Ref) exitPoint {
	if len(evaluation.body.facts.CapturedSymbols) == 0 {
		return point
	}
	attrs := evaluation.body.graph.NodeAttributes(evaluation.body.graph.Nodes[ref-1])
	if evaluation.body.graph.Tables.Text(attrs.Construct) != "closure" {
		return point
	}
	values := evaluation.values[point.values]
	cells := maps.Clone(values.cells)
	if cells == nil {
		cells = map[pdg.Ref]exitValue{}
	}
	for _, symbol := range attrs.Symbols {
		cells[symbol] = joinCellValue(cells, symbol, values.bindings[symbol])
	}
	point.values = evaluation.internValues(exitValues{bindings: values.bindings, nodes: values.nodes, tuples: values.tuples, cells: cells})
	return point
}
