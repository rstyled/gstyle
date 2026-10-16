package gopatterns

import (
	"go/ast"
	"go/types"
)

func newResourceBindings(info *types.Info, expressions []ast.Expr) bool {
	for _, expr := range expressions {
		ident, ok := expr.(*ast.Ident)
		if !ok || ident.Name == "_" || info.Defs[ident] == nil {
			return false
		}
	}
	return true
}

func osFileCall(info *types.Info, call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	fn, ok := info.Uses[selector.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "os" {
		return false
	}
	switch fn.Name() {
	case "Open", "OpenFile", "Create", "CreateTemp":
		return true
	}
	return false
}
