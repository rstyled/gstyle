package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
)

func fileAcquisition(info *types.Info, assign *ast.AssignStmt) bool {
	if assign.Tok != token.DEFINE || len(assign.Lhs) != 2 || len(assign.Rhs) != 1 {
		return false
	}
	if !newResourceBindings(info, assign.Lhs) {
		return false
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	return osFileCall(info, call)
}

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
