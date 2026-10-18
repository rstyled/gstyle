package bindingflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/constant"
)

// ParameterZero requires resolved compiler facts, including for unnamed outputs.
func (q *Query) ParameterZero(parameter pdg.Parameter) Value {
	if q == nil {
		return Value{}
	}
	primitive, err := q.parameterPrimitive(parameter)
	if err != nil {
		return Value{}
	}
	if primitive == "bool" {
		return Value{Scalar: constant.MakeBool(false)}
	}
	return Value{Nil: nilableKind(q.symbolKind(parameter.Extensions))}
}
