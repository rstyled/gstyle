package goquality

import (
	"go/ast"
	"go/token"
	"go/types"
)

// Observe the outer header without exposing rows, or prove each ranged row's
// uses separately. The outer container is never passed to an iterator.
func (x *extractor) privateMapObservation(node, parent ast.Node) bool {
	if call, ok := parent.(*ast.CallExpr); ok {
		return tableLength(call, node, x.pkg.TypesInfo)
	}
	return x.readOnlyRowRange(node, parent)
}

// Range keys do not expose rows. A newly declared row used only for metadata
// or immediate sorted key collection cannot publish or change row storage.
func (x *extractor) readOnlyRowRange(node, parent ast.Node) bool {
	statement, ok := parent.(*ast.RangeStmt)
	if !ok {
		return false
	}
	if statement.X != node {
		return false
	}
	if statement.Value == nil {
		return true
	}
	id, ok := ast.Unparen(statement.Value).(*ast.Ident)
	if !ok || statement.Tok != token.DEFINE {
		return false
	}
	return x.readOnlyRowUses(x.pkg.TypesInfo.ObjectOf(id))
}

func (x *extractor) readOnlyRowUses(object types.Object) bool {
	if object == nil {
		return false
	}
	proof := receiverProof{parents: x.fields.parents}
	for _, use := range x.fields.uses[object] {
		node, parent := proof.parent(use)
		if proof.function(use) != nil || !x.rowObserver(node, parent) {
			return false
		}
	}
	return true
}

func (x *extractor) rowObserver(node, parent ast.Node) bool {
	call, ok := parent.(*ast.CallExpr)
	return ok && (tableLength(call, node, x.pkg.TypesInfo) || x.sortedRowKeys(call, node))
}

// maps.Keys retains the row in an iterator. Only a direct slices.Sorted call
// consumes it here: it copies ordered scalar keys into fresh slice storage,
// invokes no user callback, and neither retains nor writes the map or values.
// An escaping iterator, another consumer, or asynchronous use proves nothing.
func (x *extractor) sortedRowKeys(call *ast.CallExpr, row ast.Node) bool {
	if !x.rowObserverCall(call, row, "maps.Keys") {
		return false
	}
	proof := receiverProof{parents: x.fields.parents}
	node, parent := proof.parent(call)
	consumer, ok := parent.(*ast.CallExpr)
	if !ok || !x.rowObserverCall(consumer, node, "slices.Sorted") {
		return false
	}
	_, parent = proof.parent(consumer)
	_, asynchronous := parent.(*ast.GoStmt)
	return !asynchronous
}

func (x *extractor) rowObserverCall(call *ast.CallExpr, argument ast.Node, name string) bool {
	if len(call.Args) != 1 || call.Args[0] != argument {
		return false
	}
	function, ok := x.calledObject(unwrappedCallee(call.Fun)).(*types.Func)
	return ok && resolvedName(function) == name
}
