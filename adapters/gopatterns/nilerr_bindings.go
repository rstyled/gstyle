package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// Compiler objects supply types at the boundary. State keys and write/copy
// selection use canonical symbols, accesses and operand positions.
func (q *nilErrGraph) indexTypedBindings(fn *ast.FuncDecl, info *types.Info) {
	for ref, source := range q.source {
		name, ok := source.(*ast.Ident)
		if !ok {
			continue
		}
		symbol := q.readSymbol(ref)
		if symbol != 0 {
			q.bindObject(symbol, info.ObjectOf(name))
		}
	}
	signature := info.Defs[fn.Name].Type().(*types.Signature)
	for i, output := range q.graph.Function.Outputs {
		q.bindObject(output.Symbol, signature.Results().At(i))
	}
}

func (q *nilErrGraph) bindObject(symbol pdg.Ref, object types.Object) {
	if symbol == 0 || object == nil {
		return
	}
	q.objects[symbol] = object
}

func (q *nilErrGraph) readSymbol(ref pdg.Ref) pdg.Ref {
	if ref == 0 {
		return 0
	}
	attrs := q.graph.NodeAttributes(q.graph.Nodes[ref-1])
	if len(attrs.Reads) != 1 || attrs.Reads[0].Location != 0 {
		return 0
	}
	return attrs.Reads[0].Symbol
}

func (q *nilErrGraph) recordOperation(e *nilErrEvidence, ref pdg.Ref) {
	writes := q.copies.Writes(ref)
	values := make([]nilErrValue, len(writes))
	// Every RHS is evaluated before any binding update, including parallel copies.
	for i, write := range writes {
		values[i] = q.writeValue(*e, write)
	}
	for i, write := range writes {
		if write.Symbol == 0 {
			continue
		}
		e.invalidateSymbol(write.Symbol)
		e.values[write.Symbol] = values[i]
	}
}

func (q *nilErrGraph) writeValue(e nilErrEvidence, write bindingflow.Write) nilErrValue {
	object := q.objects[write.Symbol]
	if object == nil {
		return nilErrValue{}
	}
	if write.Zero {
		return nilErrZeroType(object.Type())
	}
	return q.assignedSource(e, write.Source, object.Type())
}

func (e nilErrEvidence) resultSymbol(position int) pdg.Ref {
	if e.graph == nil || position >= len(e.graph.graph.Function.Outputs) {
		return 0
	}
	return e.graph.graph.Function.Outputs[position].Symbol
}

func (e *nilErrEvidence) invalidateSymbol(symbol pdg.Ref) {
	var retained []nilErrCheck
	for _, fact := range e.guards {
		if fact.path.Root != symbol {
			retained = append(retained, fact)
		}
	}
	e.guards = retained
}
