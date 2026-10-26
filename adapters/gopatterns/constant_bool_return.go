package gopatterns

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

func findConstantBoolReturns(fn *ast.FuncDecl, fset *token.FileSet, info *types.Info, graph *pdg.Graph) []patterns.ConstantBoolReturnHit {
	if !unnamedBooleanResult(fn, info) {
		return nil
	}
	branch, left, right := terminalBoolBranch(fn.Body.List, info)
	value, known := matchingBooleanReturns(left, right, info)
	if !known || !reachableBoolReturns(graph, fset, left, right) {
		return nil
	}
	return []patterns.ConstantBoolReturnHit{{
		Line:    fset.PositionFor(branch.Pos(), false).Line,
		EndLine: fset.PositionFor(right.End(), false).Line,
		Value:   value,
	}}
}

func unnamedBooleanResult(fn *ast.FuncDecl, info *types.Info) bool {
	if fn.Body == nil || fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	result := fn.Type.Results.List[0]
	if len(result.Names) != 0 {
		return false
	}
	basic, ok := info.TypeOf(result.Type).Underlying().(*types.Basic)
	return ok && basic.Kind() == types.Bool
}

func terminalBoolBranch(statements []ast.Stmt, info *types.Info) (*ast.IfStmt, *ast.ReturnStmt, *ast.ReturnStmt) {
	for index, statement := range statements {
		if branch, ok := statement.(*ast.IfStmt); ok {
			return branch, singleExplicitReturn(branch.Body.List), terminalBoolAlternative(branch, statements[index+1:], info)
		}
		if !constantBoolSetup(statement) {
			return nil, nil, nil
		}
	}
	return nil, nil, nil
}

func terminalBoolAlternative(branch *ast.IfStmt, tail []ast.Stmt, info *types.Info) *ast.ReturnStmt {
	if !constantBoolSetup(branch.Init) || !constantBoolCondition(branch.Cond, info) {
		return nil
	}
	if branch.Else == nil {
		return singleExplicitReturn(tail)
	}
	other, ok := branch.Else.(*ast.BlockStmt)
	if !ok || len(tail) != 0 {
		return nil
	}
	return singleExplicitReturn(other.List)
}

func singleExplicitReturn(statements []ast.Stmt) *ast.ReturnStmt {
	if len(statements) != 1 {
		return nil
	}
	returned, _ := statements[0].(*ast.ReturnStmt)
	return returned
}

func constantBoolCondition(expression ast.Expr, info *types.Info) bool {
	if info.Types[expression].Value != nil {
		return false
	}
	supported := true
	ast.Inspect(expression, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		_, logical := logicalExpression(node)
		supported = supported && !logical
		return supported
	})
	return supported
}

func constantBoolSetup(statement ast.Stmt) bool {
	switch statement.(type) {
	case nil, *ast.AssignStmt, *ast.DeclStmt, *ast.ExprStmt, *ast.IncDecStmt, *ast.EmptyStmt:
		return true
	default:
		return false
	}
}

func matchingBooleanReturns(first, second *ast.ReturnStmt, info *types.Info) (bool, bool) {
	left, leftKnown := returnedBoolean(first, info)
	right, rightKnown := returnedBoolean(second, info)
	return left, leftKnown && rightKnown && left == right
}

func returnedBoolean(statement *ast.ReturnStmt, info *types.Info) (bool, bool) {
	if statement == nil || len(statement.Results) != 1 {
		return false, false
	}
	value := info.Types[statement.Results[0]].Value
	if value == nil || value.Kind() != constant.Bool {
		return false, false
	}
	return constant.BoolVal(value), true
}

func reachableBoolReturns(graph *pdg.Graph, fset *token.FileSet, left, right *ast.ReturnStmt) bool {
	kept := pdg.ReachableProjection(graph)
	return kept != nil && reachableBoolReturn(graph, kept, fset, left) && reachableBoolReturn(graph, kept, fset, right)
}

func reachableBoolReturn(graph *pdg.Graph, kept []bool, fset *token.FileSet, returned *ast.ReturnStmt) bool {
	rangeOfReturn := nodeByteRange(fset, returned)
	for index, node := range graph.Nodes {
		if node.Category != pdg.Return || !kept[index] {
			continue
		}
		span := graph.Spans[node.Span-1]
		if span.Start == rangeOfReturn.Start && span.End == rangeOfReturn.End {
			return true
		}
	}
	return false
}
