package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (evaluation *exitEvaluation) callCellValues(values exitValues) map[pdg.Ref]bindingflow.Value {
	result := evaluation.concreteValues(values.cells)
	for symbol := range values.cells {
		if evaluation.body.facts.SingleCellSymbols[symbol] {
			result[symbol] = evaluation.concreteValue(values.liveBindings()[symbol])
		}
	}
	return result
}
