package callflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
)

// assumeCondition records only direct whole-binding facts at a selected edge.
// Subsequent stores use the normal transfer/invalidation rules. Heap locations
// and unknown expressions do not acquire facts from a source-text guess.
func (evaluation *exitEvaluation) assumeCondition(point exitPoint, condition pdg.Ref, truth bool) exitPoint {
	context := evaluation.valueContext(point)
	value := evaluation.body.paths.Evaluate(condition, context)
	if value.Boolean != 0 {
		bound := falseValue
		if truth != value.Negated {
			bound = trueValue
		}
		return evaluation.assumeBinding(point, value.Boolean, bound)
	}
	subject, notNil := evaluation.body.paths.NilPredicate(condition, evaluation.body.facts)
	symbol := evaluation.conditionSymbol(subject)
	if symbol == 0 {
		return point
	}
	bound := nilValue
	if truth == notNil {
		bound = nonNilValue
	}
	return evaluation.assumeBinding(point, symbol, bound)
}

func (evaluation *exitEvaluation) conditionSymbol(subject pdg.Ref) pdg.Ref {
	if subject == 0 {
		return 0
	}
	graph := evaluation.body.graph
	node := graph.Nodes[subject-1]
	attrs := graph.NodeAttributes(node)
	if node.Category != pdg.Read || len(attrs.Reads) != 1 || !attrs.Reads[0].DirectSymbol() {
		return 0
	}
	return attrs.Reads[0].Symbol
}

func (evaluation *exitEvaluation) assumeBinding(point exitPoint, symbol pdg.Ref, bound exitValue) exitPoint {
	values := evaluation.values[point.values]
	bindings := make(map[pdg.Ref]exitValue, len(values.bindings)+1)
	maps.Copy(bindings, values.bindings)
	bindings[symbol] = bound
	point.values = evaluation.internValues(exitValues{bindings: bindings, nodes: values.nodes, tuples: values.tuples, cells: values.cells, unorderedReads: values.unorderedReads})
	return point
}
