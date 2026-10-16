package gopatterns

import (
	"go/ast"
	"go/types"
)

// A discarded error alone does not establish that a failed write becomes
// success. Require a reachable write immediately followed by a nil error return.
func (w *errorCheckWalker) successReturnOutput(call *ast.CallExpr) bool {
	if !w.reachableOutput(call) || !simpleOutputOperands(call.Args, w.info) {
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
	return directOutputStatement(statement, call) && successfulOutputReturn(result, index, info)
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
	value, ok := w.reachable[span]
	return ok && !nilOutputWriter(call, value.Nil, w.info)
}

// Only operands with no unmodeled evaluation establish completion here.
func simpleOutputOperands(operands []ast.Expr, info *types.Info) bool {
	for _, operand := range operands {
		if !simpleOutputOperand(ast.Unparen(operand), info) {
			return false
		}
	}
	return true
}

func simpleOutputOperand(expr ast.Expr, info *types.Info) bool {
	if info.Types[expr].Value != nil {
		return true
	}
	switch expr := expr.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		ident, ok := expr.X.(*ast.Ident)
		if !ok {
			return false
		}
		_, ok = info.ObjectOf(ident).(*types.PkgName)
		return ok
	default:
		return false
	}
}

func successfulOutputReturn(result *ast.ReturnStmt, index int, info *types.Info) bool {
	return typedNil(info, ast.Unparen(result.Results[index])) && simpleOutputOperands(result.Results, info)
}

func nilOutputWriter(call *ast.CallExpr, knownNil bool, info *types.Info) bool {
	if len(call.Args) == 0 {
		return false
	}
	writer := call.Args[0]
	if info.Types[writer].IsNil() {
		return true
	}
	typ := info.TypeOf(writer)
	return knownNil && !unknownInterfaceValue(typ) && interfaceType(typ)
}
