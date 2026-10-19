package goquality

import (
	"go/ast"
	"go/types"
)

// errors.New returns a private record containing only an immutable string.
// Package use checks still reject rebinding and exposing the global slot.
func stableErrorInitializer(value ast.Expr, info *types.Info) bool {
	call, ok := ast.Unparen(value).(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	selector, ok := unwrappedCallee(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return false
	}
	function, ok := info.ObjectOf(selector.Sel).(*types.Func)
	if !ok || resolvedName(function) != "errors.New" {
		return false
	}
	// The typed errors.New argument is a string; require a constant value.
	return info.Types[call.Args[0]].Value != nil
}

func errorValueType(typ types.Type) bool {
	return types.Identical(typ, types.Universe.Lookup("error").Type())
}
