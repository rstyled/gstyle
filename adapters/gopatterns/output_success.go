package gopatterns

import (
	"go/ast"
	"go/types"
)

func outputErrorResult(fn *ast.FuncDecl, info *types.Info) int {
	results := info.Defs[fn.Name].Type().(*types.Signature).Results()
	if results.Len() == 0 {
		return -1
	}
	index := results.Len() - 1
	if !isErrorTypeCheck(results.At(index).Type()) {
		return -1
	}

	return index
}
