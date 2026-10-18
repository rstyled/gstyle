package bindingflow

import "github.com/rstyled/gstyle/domain/pdg"

// CallValues returns assigned formal values, including the packed slice. A
// complete shape and canonical expression inventory are required for expansion.
func (q *Query) CallValues(owner pdg.Ref, inputs []pdg.Parameter, context Evaluation) ([]Value, bool) {
	if q == nil {
		return nil, false
	}
	fixed, variadic, known := CallSignature(inputs)
	if !known {
		return nil, false
	}
	shape, known := q.receivingShape(owner, fixed, variadic)
	if !known {
		return nil, false
	}
	values := make([]Value, len(inputs))
	for i := range fixed {
		values[i] = q.InputValue(owner, pdg.Position(i+1), context)
	}
	if variadic {
		values[fixed] = q.variadicValue(owner, shape, context)
	}
	return values, true
}

func (q *Query) receivingShape(owner pdg.Ref, fixed int, variadic bool) (CallShape, bool) {
	operands, known := q.callOperands(owner)
	if !known {
		return CallShape{}, false
	}
	shape, found := q.callShapes[owner]
	if !found {
		if variadic {
			return CallShape{}, false
		}
		shape = CallShape{Fixed: uint32(fixed)}
	}
	if !shape.matches(fixed, variadic) {
		return CallShape{}, false
	}
	count, known := q.argumentCount(owner, shape, operands)
	return shape, known && shape.accepts(count)
}

func (q *Query) argumentCount(owner pdg.Ref, shape CallShape, operands []pdg.Ref) (uint64, bool) {
	if shape.Tuple == 0 {
		return uint64(len(operands)), q.tupleProducer(owner) == 0
	}
	tuple := q.tupleProducer(owner)
	if len(operands) != 1 || tuple == 0 {
		return 0, false
	}
	return uint64(shape.Tuple), q.tupleArities[tuple] == shape.Tuple
}

func (q *Query) variadicValue(owner pdg.Ref, shape CallShape, context Evaluation) Value {
	if shape.Spread {
		return q.InputValue(owner, pdg.Position(shape.Fixed+1), context)
	}
	values, _ := q.InputValues(owner)
	count, _ := q.argumentCount(owner, shape, values)
	return Value{Nil: count == uint64(shape.Fixed), NonNil: count > uint64(shape.Fixed)}
}

func (q *Query) tupleInputValue(owner pdg.Ref, position pdg.Position) Value {
	tuple := q.tupleProducer(owner)
	if uint64(position) > uint64(q.tupleArities[tuple]) {
		return Value{}
	}
	if q.inputRepresentation(owner, position) == BoxedRepresentation {
		return Value{NonNil: true}
	}
	return Value{}
}

func (q *Query) callOperands(owner pdg.Ref) ([]pdg.Ref, bool) {
	values, known := q.InputValues(owner)
	if !known {
		return nil, false
	}
	positions, _ := q.RequiredOperands(owner)
	for i, position := range positions {
		if uint64(position) != uint64(i+1) {
			return nil, false
		}
	}
	return values, true
}

func (shape CallShape) accepts(count uint64) bool {
	if shape.Spread {
		return count == uint64(shape.Fixed)+1
	}
	if shape.Variadic {
		return count >= uint64(shape.Fixed)
	}
	return count == uint64(shape.Fixed)
}

func (shape CallShape) matches(fixed int, variadic bool) bool {
	return uint64(shape.Fixed) == uint64(fixed) && shape.Variadic == variadic
}

// CallSignature admits canonical free-function receiving slots. Receiver
// wrappers retain their separate contract; only the final slot can be variadic.
func CallSignature(inputs []pdg.Parameter) (int, bool, bool) {
	for i, input := range inputs {
		if !callFormal(input, i, len(inputs)) {
			return 0, false, false
		}
	}
	variadic := len(inputs) > 0 && inputs[len(inputs)-1].Variadic.Value
	fixed := len(inputs)
	if variadic {
		fixed--
	}
	return fixed, variadic, true
}

func callFormal(input pdg.Parameter, index, count int) bool {
	return uint64(input.Position) == uint64(index) && input.Role == pdg.ParameterRole && input.Variadic.Status == pdg.Known && (!input.Variadic.Value || index == count-1)
}
