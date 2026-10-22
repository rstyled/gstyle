package goquality

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/quality"
)

func (x *extractor) indexedOrigin(value *ast.IndexExpr, seen map[types.Object]bool) quality.Provenance {
	if object := x.mapEntryRoot(value); x.privateAppendEntries(object) {
		return quality.Local
	}
	return referencedOrigin(x.origin(value.X, seen))
}

// A fresh private map or slice starts with nil slice entries. Updating entries
// only by append from that same container preserves owned row storage.
// Element references retain their usual borrowed provenance.
func (x *extractor) privateAppendEntries(object types.Object) bool {
	if !x.emptyLocalEntries(object) || x.fields.deferred {
		return false
	}
	proof := receiverProof{parents: x.fields.parents}
	for _, use := range x.fields.uses[object] {
		if proof.function(use) != nil || !x.privateMapUse(use, object) {
			return false
		}
	}
	return true
}

func (x *extractor) emptyLocalEntries(object types.Object) bool {
	if !x.localContainerRoot(object) {
		return false
	}
	if !sliceEntryContainer(object.Type()) {
		return false
	}
	binding := x.bindings[object]
	return !binding.unknown && len(binding.values) == 1 && x.emptyEntries(binding.values[0])
}

func sliceEntryContainer(typ types.Type) bool {
	switch shape := underlying(typ).(type) {
	case *types.Map:
		typ = shape.Elem()
	case *types.Slice:
		typ = shape.Elem()
	default:
		return false
	}
	_, slice := underlying(typ).(*types.Slice)
	return slice
}

func (x *extractor) emptyEntries(value ast.Expr) bool {
	switch value := ast.Unparen(value).(type) {
	case *ast.CompositeLit:
		return len(value.Elts) == 0
	case *ast.CallExpr:
		builtin, ok := x.calledObject(unwrappedCallee(value.Fun)).(*types.Builtin)
		return ok && builtin.Name() == "make"
	default:
		return false
	}
}

func (x *extractor) privateMapUse(id *ast.Ident, object types.Object) bool {
	proof := receiverProof{parents: x.fields.parents}
	node, parent := proof.parent(id)
	switch parent := parent.(type) {
	case *ast.ReturnStmt:
		return true // Publishing ends construction; deferred work is excluded.
	case *ast.IndexExpr:
		return parent.X == node && x.privateMapEntryUse(parent, object)
	default:
		return false
	}
}

func (x *extractor) privateMapEntryUse(entry *ast.IndexExpr, object types.Object) bool {
	_, parent := (receiverProof{parents: x.fields.parents}).parent(entry)
	switch parent := parent.(type) {
	case *ast.CallExpr:
		return x.mapEntryAppend(parent, object) && ast.Unparen(parent.Args[0]) == entry && x.mapAppendWrite(parent, object)
	case *ast.AssignStmt:
		return x.mapEntryAssignment(parent, object)
	case *ast.BinaryExpr:
		return x.nilComparison(parent)
	default:
		return false
	}
}

// Comparing with the predeclared nil observes a header without exposing storage.
func (x *extractor) nilComparison(value *ast.BinaryExpr) bool {
	return (value.Op == token.EQL || value.Op == token.NEQ) &&
		(x.pkg.TypesInfo.Types[value.X].IsNil() || x.pkg.TypesInfo.Types[value.Y].IsNil())
}

func (x *extractor) mapAppendWrite(call *ast.CallExpr, object types.Object) bool {
	_, parent := (receiverProof{parents: x.fields.parents}).parent(call)
	statement, ok := parent.(*ast.AssignStmt)
	return ok && x.mapEntryAssignment(statement, object)
}

func (x *extractor) mapEntryAssignment(statement *ast.AssignStmt, object types.Object) bool {
	return len(statement.Lhs) == 1 && len(statement.Rhs) == 1 &&
		x.mapEntryRoot(statement.Lhs[0]) == object && x.mapEntryAppend(statement.Rhs[0], object)
}

func (x *extractor) mapEntryAppend(value ast.Expr, object types.Object) bool {
	call, ok := ast.Unparen(value).(*ast.CallExpr)
	if !ok || len(call.Args) == 0 {
		return false
	}
	builtin, ok := x.calledObject(unwrappedCallee(call.Fun)).(*types.Builtin)
	return ok && builtin.Name() == "append" && x.mapEntryRoot(call.Args[0]) == object
}

func (x *extractor) mapEntryRoot(value ast.Expr) types.Object {
	entry, ok := ast.Unparen(value).(*ast.IndexExpr)
	if !ok {
		return nil
	}
	id, ok := ast.Unparen(entry.X).(*ast.Ident)
	if !ok {
		return nil
	}
	return x.pkg.TypesInfo.ObjectOf(id)
}
