package goquality

import (
	"go/ast"
	"go/types"
)

// A single reaching append to a typed nil slice replaces even a parameter's
// storage. An empty expansion stays nil; a nonempty expansion allocates. Only
// the outer storage is owned. Captured reads do not publish the header's address;
// any other header write or address exposure rejects this bounded proof.
func (x *extractor) nilCloneStorage(id *ast.Ident) bool {
	object := x.pkg.TypesInfo.ObjectOf(id)
	writes := x.sliceWrites[object]
	if len(writes) != 1 || x.bindings[object].unknown || packageVariable(object) {
		return false
	}
	value := writes[0].value
	if !x.nilClone(value) {
		return false
	}
	current, block := x.unlabeledFieldBlock(id)
	return block != nil && x.reachingSliceValue(object, current, block) == value
}

func (x *extractor) nilClone(value ast.Expr) bool {
	call, ok := ast.Unparen(value).(*ast.CallExpr)
	if !ok {
		return false
	}
	builtin, ok := x.calledObject(unwrappedCallee(call.Fun)).(*types.Builtin)
	return ok && builtin.Name() == "append" && x.typedNilSlice(call.Args[0])
}

func (x *extractor) typedNilSlice(value ast.Expr) bool {
	call, ok := ast.Unparen(value).(*ast.CallExpr)
	if !ok || !x.pkg.TypesInfo.Types[call.Fun].IsType() {
		return false
	}
	_, slice := underlying(x.pkg.TypesInfo.TypeOf(call)).(*types.Slice)
	return slice && x.pkg.TypesInfo.Types[call.Args[0]].IsNil()
}
