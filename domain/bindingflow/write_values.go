package bindingflow

import (
	"encoding/json"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/constant"
)

type ValueWrite struct {
	Write
	Value Value
}

// WriteValues transfers whole-value nilness and boolean facts. It reads every
// RHS from the same pre-store context. Other scalar values remain unknown.
func (q *Query) WriteValues(owner pdg.Ref, context Evaluation) []ValueWrite {
	writes := q.Writes(owner)
	result := make([]ValueWrite, len(writes))
	for i, write := range writes {
		result[i] = ValueWrite{Write: write, Value: q.writeValue(owner, write, context)}
	}
	return result
}

// ZeroValue proves nil only for an explicit compiler nilable-type fact.
func (q *Query) ZeroValue(symbol pdg.Ref) Value {
	if q == nil || symbol == 0 || int(symbol) > len(q.graph.Symbols) {
		return Value{}
	}
	if nilableKind(q.symbolKind(q.graph.Symbols[symbol-1].Extensions)) {
		return Value{Nil: true}
	}
	return Value{}
}

// InputValue evaluates an assigned input only when its canonical inventory is
// complete. Representation conversion belongs to the receiving slot.
func (q *Query) InputValue(owner pdg.Ref, position pdg.Position, context Evaluation) Value {
	if q == nil {
		return Value{}
	}
	if _, known := q.InputValues(owner); !known {
		return Value{}
	}
	if q.tupleProducer(owner) != 0 {
		return q.tupleInputValue(owner, position, context)
	}
	return q.writeValue(owner, Write{Source: q.Operand(owner, position), Position: position}, context)
}

func (q *Query) writeValue(owner pdg.Ref, write Write, context Evaluation) Value {
	if write.Zero {
		return q.InitialValue(write.Symbol, context.Facts)
	}
	if write.Source == 0 {
		return Value{}
	}
	if q.tupleProducer(owner) != 0 {
		return q.tupleInputValue(owner, write.Position, context)
	}
	switch q.inputRepresentation(owner, write.Position) {
	case UnchangedRepresentation:
		return shapeValue(q.Evaluate(write.Source, context))
	case BoxedRepresentation:
		return Value{NonNil: true}
	default:
		return Value{}
	}
}

func (q *Query) symbolKind(extensions []pdg.Extension) string {
	var kind string
	found := false
	for _, extension := range extensions {
		if q.graph.Tables.Text(extension.Name) != "gstyle.memory:kind" {
			continue
		}
		if found || json.Unmarshal([]byte(q.graph.Tables.Text(extension.Value)), &kind) != nil {
			return ""
		}
		found = true
	}
	return kind
}

func shapeValue(value Value) Value {
	if value.Nil || value.NonNil {
		return Value{Nil: value.Nil, NonNil: value.NonNil}
	}
	if truth, known := value.Bool(); known {
		return Value{Scalar: constant.MakeBool(truth)}
	}
	return Value{}
}

func nilableKind(kind string) bool {
	switch kind {
	case "pointer", "interface", "slice", "map", "channel", "function":
		return true
	default:
		return false
	}
}

// InitialValue adds the compiler-owned Boolean zero to the existing nilable
// zero contract. Missing type evidence does not establish another zero shape.
func (q *Query) InitialValue(symbol pdg.Ref, facts Facts) Value {
	if q == nil || symbol == 0 || int(symbol) > len(q.graph.Symbols) {
		return Value{}
	}
	if facts.BooleanSymbols[symbol] {
		return Value{Scalar: constant.MakeBool(false)}
	}
	return q.ZeroValue(symbol)
}
