package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (evaluation *exitEvaluation) returnAt(point exitPoint, ref pdg.Ref) completions {
	returned := evaluation.returnTuple(point, ref)
	point = evaluation.assignReturnSlots(point, returned)
	return completions{{values: point.values, returned: returned, mode: Returns, recovered: point.recovered}: true}
}

func (evaluation *exitEvaluation) returnTuple(point exitPoint, ref pdg.Ref) string {
	outputs := evaluation.body.graph.Function.Outputs
	values := make([]byte, len(outputs))
	context := evaluation.valueContext(point)
	for i, output := range outputs {
		value := evaluation.returnSlot(ref, pdg.Position(i+1), output.Symbol, context)
		values[i] = byte(abstractValue(value))
	}
	return string(values)
}

func (evaluation *exitEvaluation) returnSlot(ref pdg.Ref, position pdg.Position, symbol pdg.Ref, context bindingflow.Evaluation) bindingflow.Value {
	inputs, known := evaluation.body.paths.InputValues(ref)
	if !known {
		return bindingflow.Value{}
	}
	if len(inputs) != 0 {
		return evaluation.body.paths.InputValue(ref, position, context)
	}
	if symbol != 0 && evaluation.body.paths.ReadSymbol(ref, position) == symbol {
		return context.Bindings[symbol]
	}
	return bindingflow.Value{}
}

func (evaluation *exitEvaluation) initialBindings(inputs string) map[pdg.Ref]exitValue {
	bindings := inputBindings(evaluation.body, inputs)
	for _, output := range evaluation.body.graph.Function.Outputs {
		if output.Symbol == 0 {
			continue
		}
		value := abstractValue(evaluation.body.paths.InitialValue(output.Symbol, evaluation.body.facts))
		if value != unknownValue {
			bindings[output.Symbol] = value
		}
	}
	return bindings
}
