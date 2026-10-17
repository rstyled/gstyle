package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (q *nilErrGraph) lostReturn(e nilErrEvidence, ref pdg.Ref) bool {
	if len(e.positions) == 0 {
		return false
	}
	values := make([]nilErrValue, len(e.results))
	for i, result := range e.results {
		values[i] = q.returnValue(e, ref, i)
		if !nilErrAssignableZero(values[i], result.Type()) {
			return false
		}
	}
	for _, position := range e.positions {
		if !values[position].Nil {
			return false
		}
	}
	return true
}
func (q *nilErrGraph) returnValue(e nilErrEvidence, ref pdg.Ref, position int) nilErrValue {
	slot := pdg.Position(position + 1)
	source := q.copies.Operand(ref, slot)
	if source != 0 {
		return q.assignedSource(e, source, e.results[position].Type())
	}
	// General bare returns carry slot-specific direct reads on the return node.
	// A missing explicit operand never falls back to the source return list.
	symbol := q.copies.ReadSymbol(ref, slot)
	if symbol == 0 {
		return nilErrValue{}
	}
	return e.values[symbol]
}
func (q *nilErrGraph) assignedSource(e nilErrEvidence, source pdg.Ref, target types.Type) nilErrValue {
	if expression, ok := q.source[source].(ast.Expr); ok {
		return nilErrAssigned(e.valueRef(source), e.info.TypeOf(expression), target, e.info.Types[expression].IsNil())
	}
	// Scalar/binding bare returns use derived read entities, without AST nodes.
	symbol := q.readSymbol(source)
	object := q.objects[symbol]
	if object == nil {
		return nilErrValue{}
	}
	return nilErrAssigned(e.valueRef(source), object.Type(), target, false)
}

// Compiler types determine representation; they do not select dependencies.
func nilErrAssigned(value nilErrValue, source, target types.Type, untypedNil bool) nilErrValue {
	if target == nil {
		return nilErrValue{}
	}
	if untypedNil {
		return value
	}
	if _, iface := target.Underlying().(*types.Interface); !iface {
		return value
	}
	if !nilErrInterfaceSource(source) {
		return nilErrValue{}
	}

	return value
}

func nilErrInterfaceSource(source types.Type) bool {
	if source == nil {
		return false
	}
	if _, generic := types.Unalias(source).(*types.TypeParam); generic {
		return false
	}
	_, iface := source.Underlying().(*types.Interface)
	return iface
}
