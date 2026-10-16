package gopatterns

import (
	"go/ast"
	"go/types"
)

// A discarded error alone does not establish that a failed write becomes
// success. Require a reachable write immediately followed by a nil error return.
func (w *errorCheckWalker) successReturnOutput(call *ast.CallExpr) bool {
	if !w.reachableOutput(call) || nestedOutputCall(call) {
		return false
	}
	index := outputErrorResult(w.function, w.info)
	if index < 0 {
		return false
	}
	for _, node := range w.stack {
		if successfulOutputBlock(node, call, index, w.info) {
			return true
		}
	}
	return false
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
	if !ok || len(block.List) < 2 {
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

func (w *errorCheckWalker) reachableOutput(call *ast.CallExpr) bool {
	if w.reachable == nil {
		return true
	}
	span := outputSpan{uint32(w.fset.PositionFor(call.Pos(), false).Offset), uint32(w.fset.PositionFor(call.End(), false).Offset)}
	return w.reachable[span]
}

// An opaque argument call can panic or exit before the write. Do not claim
// that the output error reaches a success return without its completion facts.
func nestedOutputCall(call *ast.CallExpr) bool {
	found := false
	for _, arg := range call.Args {
		ast.Inspect(arg, func(node ast.Node) bool {
			if _, ok := node.(*ast.CallExpr); ok {
				found = true
			}
			return !found
		})
	}
	return found
}
