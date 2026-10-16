package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
)

// Exclude only diagnostics whose enclosing error guard returns the same
// non-nil local binding on every normal path. This does not classify callbacks.
func (w *errorCheckWalker) failureOnlyOutput(call *ast.CallExpr) bool {
	index := failureResultIndex(w.function, w.info)
	if index < 0 {
		return false
	}
	for _, node := range w.stack {
		if w.failureOutputGuard(call, node, index) {
			return true
		}
	}
	return false
}

func (w *errorCheckWalker) failureOutputGuard(call *ast.CallExpr, node ast.Node, index int) bool {
	guard, ok := node.(*ast.IfStmt)
	if !ok || !insideOutputGuard(call, guard) {
		return false
	}
	object := guardedOutputError(guard.Cond, w.info)
	if object == nil || object.Pos() < w.function.Pos() {
		return false
	}
	proof := failureOutputProof{info: w.info, errorObject: object, resultIndex: index, valid: true}
	ast.Inspect(w.function.Body, proof.checkEscape)
	return proof.valid && failureReturnBlock(guard.Body, object, index, w.info)
}

func insideOutputGuard(call *ast.CallExpr, guard *ast.IfStmt) bool {
	return call.Pos() > guard.Body.Lbrace && call.End() < guard.Body.Rbrace
}

func failureResultIndex(fn *ast.FuncDecl, info *types.Info) int {
	signature := info.Defs[fn.Name].Type().(*types.Signature)
	results := signature.Results()
	if results.Len() == 0 {
		return -1
	}
	index := results.Len() - 1
	if !isErrorTypeCheck(results.At(index).Type()) {
		return -1
	}
	// Deferred writes can replace named result values after return evaluation.
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

func guardedOutputError(expr ast.Expr, info *types.Info) types.Object {
	comparison, ok := ast.Unparen(expr).(*ast.BinaryExpr)
	if !ok || comparison.Op != token.NEQ {
		return nil
	}
	if !typedNil(info, ast.Unparen(comparison.Y)) {
		return nil
	}
	return outputErrorObject(comparison.X, info)
}

func outputErrorObject(expr ast.Expr, info *types.Info) types.Object {
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	if !ok {
		return nil
	}
	object := info.ObjectOf(ident)
	if object == nil || !isErrorTypeCheck(object.Type()) {
		return nil
	}
	return object
}

type failureOutputProof struct {
	info        *types.Info
	errorObject types.Object
	resultIndex int
	valid       bool
}

func failureReturnBlock(body *ast.BlockStmt, object types.Object, index int, info *types.Info) bool {
	if len(body.List) == 0 {
		return false
	}
	last, ok := body.List[len(body.List)-1].(*ast.ReturnStmt)
	if !ok {
		return false
	}
	proof := failureOutputProof{info: info, errorObject: object, resultIndex: index, valid: true}
	if !proof.returnsError(last) {
		return false
	}
	ast.Inspect(body, proof.visit)
	return proof.valid
}

func (p *failureOutputProof) returnsError(stmt *ast.ReturnStmt) bool {
	if p.resultIndex >= len(stmt.Results) {
		return false
	}
	return p.matches(stmt.Results[p.resultIndex])
}

func (p *failureOutputProof) matches(expr ast.Expr) bool {
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	return ok && p.info.ObjectOf(ident) == p.errorObject
}

func (p *failureOutputProof) visit(node ast.Node) bool {
	if !p.valid {
		return false
	}
	switch node := node.(type) {
	case *ast.ReturnStmt:
		p.valid = p.returnsError(node)
	case *ast.AssignStmt:
		p.valid = !p.reassigns(node.Lhs)
	case *ast.RangeStmt:
		p.valid = !p.reassigns([]ast.Expr{node.Key, node.Value})
	case *ast.FuncLit, *ast.BranchStmt:
		p.valid = false
	}
	return p.valid
}

func (p *failureOutputProof) reassigns(targets []ast.Expr) bool {
	for _, expr := range targets {
		if expr != nil && p.matches(expr) {
			return true
		}
	}
	return false
}

// A captured or address-taken binding can change through an opaque call.
func (p *failureOutputProof) checkEscape(node ast.Node) bool {
	if !p.valid {
		return false
	}
	switch node := node.(type) {
	case *ast.FuncLit:
		p.valid = false
	case *ast.UnaryExpr:
		p.valid = node.Op != token.AND || !p.matches(node.X)
	}
	return p.valid
}
