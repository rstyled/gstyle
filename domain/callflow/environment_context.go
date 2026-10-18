package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (evaluation *exitEvaluation) concreteValue(value exitValue) bindingflow.Value {
	if result, found := evaluation.concrete[value]; found {
		return result
	}
	result := concreteValue(value)
	if evaluation.concrete == nil {
		evaluation.concrete = map[exitValue]bindingflow.Value{}
	}
	evaluation.concrete[value] = result
	return result
}
func (evaluation *exitEvaluation) concreteValues(values map[pdg.Ref]exitValue) map[pdg.Ref]bindingflow.Value {
	result := make(map[pdg.Ref]bindingflow.Value, len(values))
	for ref, value := range values {
		result[ref] = evaluation.concreteValue(value)
	}
	return result
}
func (evaluation *exitEvaluation) concreteResults(values map[pdg.Ref]string) map[pdg.Ref][]bindingflow.Value {
	result := make(map[pdg.Ref][]bindingflow.Value, len(values))
	for ref, tuple := range values {
		slots := decodeTuple(tuple)
		items := make([]bindingflow.Value, len(slots))
		for i, value := range slots {
			items[i] = evaluation.concreteValue(value)
		}
		result[ref] = items
	}
	return result
}
func (evaluation *exitEvaluation) concreteHeap(values exitValues) map[string]bindingflow.Value {
	result := map[string]bindingflow.Value{}
	for address, cell := range values.heap {
		result[address] = evaluation.concreteValue(cell.Value)
	}
	return result
}
