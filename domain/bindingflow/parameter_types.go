package bindingflow

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (q *Query) validateParameterTypes() error {
	for _, group := range [][]pdg.Parameter{q.graph.Function.Inputs, q.graph.Function.Outputs} {
		for i, parameter := range group {
			if _, err := q.parameterPrimitive(parameter); err != nil {
				return fmt.Errorf("parameter %d primitive fact: %w", i+1, err)
			}
		}
	}
	return nil
}

func (q *Query) parameterPrimitive(parameter pdg.Parameter) (string, error) {
	primitive, err := q.symbolPrimitive(parameter.Extensions)
	if err != nil {
		return "", err
	}
	if primitive == "bool" && nilableKind(q.symbolKind(parameter.Extensions)) {
		return "", fmt.Errorf("Boolean parameter has a nilable memory kind")
	}
	return primitive, nil
}
