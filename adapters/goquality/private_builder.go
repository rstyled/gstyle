package goquality

import (
	"go/ast"
	"go/token"
	"go/types"
)

func (x *extractor) privateBuilderFormat(call *ast.CallExpr, callee string) bool {
	if !x.memoryFormat(call, callee) {
		return false
	}
	object := x.builderObject(call.Args[0])
	if !x.zeroBuilder(object) {
		return false
	}
	for _, use := range x.fields.uses[object] {
		if !x.builderSynchronous(use) || !x.privateBuilderUse(use) {
			return false
		}
	}
	return true
}

func (x *extractor) builderObject(value ast.Expr) types.Object {
	address, ok := ast.Unparen(value).(*ast.UnaryExpr)
	if !ok || address.Op != token.AND {
		return nil
	}
	id, ok := ast.Unparen(address.X).(*ast.Ident)
	if !ok {
		return nil
	}
	return x.pkg.TypesInfo.ObjectOf(id)
}

func (x *extractor) zeroBuilder(object types.Object) bool {
	if !x.localContainerRoot(object) {
		return false
	}
	if !metadataType(object.Type(), "strings", "Builder") {
		return false
	}
	binding := x.bindings[object]
	return len(binding.values) == 1 && binding.values[0] == nil
}

func (x *extractor) builderSynchronous(node ast.Node) bool {
	for parent := x.fields.parents[node]; parent != nil; parent = x.fields.parents[parent] {
		switch parent.(type) {
		case *ast.FuncLit, *ast.GoStmt, *ast.DeferStmt:
			return false
		}
	}
	return true
}

func (x *extractor) privateBuilderUse(id *ast.Ident) bool {
	_, parent := (receiverProof{parents: x.fields.parents}).parent(id)
	switch parent := parent.(type) {
	case *ast.SelectorExpr:
		call, ok := x.fields.parents[parent].(*ast.CallExpr)
		return ok && call.Fun == parent && builderMethod(x.readerCallee(call))
	case *ast.UnaryExpr:
		return x.privateBuilderAddress(parent)
	default:
		return false
	}
}

func (x *extractor) privateBuilderAddress(address *ast.UnaryExpr) bool {
	if address.Op != token.AND {
		return false
	}
	node, outer := (receiverProof{parents: x.fields.parents}).parent(address)
	call, ok := outer.(*ast.CallExpr)
	return ok && len(call.Args) > 0 && call.Args[0] == node && x.memoryFormat(call, x.readerCallee(call))
}

func builderMethod(callee string) bool {
	switch callee {
	case "strings.Builder.Write", "strings.Builder.WriteString", "strings.Builder.WriteByte", "strings.Builder.WriteRune",
		"strings.Builder.Grow", "strings.Builder.Reset", "strings.Builder.Len", "strings.Builder.Cap", "strings.Builder.String":
		return true
	default:
		return false
	}
}
