package gopatterns

import (
	"go/ast"
	"go/types"
)

// A discarded error alone does not establish that a failed write becomes
// success. Match a two-statement body: the write, then a nil error return.
func (w *errorCheckWalker) successReturnOutput(call *ast.CallExpr) bool {
	index := outputErrorResult(w.function, w.info)
	if index < 0 {
		return false
	}
	return successfulOutputBlock(w.function.Body, call, index, w.info)
}

func outputErrorResult(fn *ast.FuncDecl, info *types.Info) int {
	results := info.Defs[fn.Name].Type().(*types.Signature).Results()
	if results.Len() == 0 {
		return -1
	}
	index := results.Len() - 1
	if !isErrorTypeCheck(results.At(index).Type()) {
		return -1
	}
	// A deferred call can replace a named nil result after return evaluation.
	if results.At(index).Name() != "" && containsOutputDefer(fn.Body) {
		return -1
	}
	return index
}

func containsOutputDefer(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if _, ok := node.(*ast.DeferStmt); ok {
			found = true
		}
		return !found
	})
	return found
}

func successfulOutputBlock(node ast.Node, call *ast.CallExpr, index int, info *types.Info) bool {
	block, ok := node.(*ast.BlockStmt)
	if !ok || len(block.List) != 2 {
		return false
	}
	result, ok := block.List[len(block.List)-1].(*ast.ReturnStmt)
	if !ok || index >= len(result.Results) {
		return false
	}
	statement := block.List[len(block.List)-2]
	return directOutputStatement(statement, call) && typedNil(info, ast.Unparen(result.Results[index]))
}

func directOutputStatement(statement ast.Stmt, call *ast.CallExpr) bool {
	switch statement.(type) {
	case *ast.ExprStmt, *ast.AssignStmt, *ast.DeclStmt:
		return statement.Pos() <= call.Pos() && call.End() <= statement.End()
	default:
		return false
	}
}
