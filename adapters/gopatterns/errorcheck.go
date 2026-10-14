package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/patterns"
)

type errorCheckWalker struct {
	fset  *token.FileSet
	info  *types.Info
	sites *[]patterns.ErrorCheckSite
	stack []ast.Node // Parent stack
	body  ast.Node
}

func (w *errorCheckWalker) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		// Pop the stack on exit.
		w.stack = w.stack[:len(w.stack)-1]
		return nil
	}
	// Push current node.
	w.stack = append(w.stack, n)
	// Check if this is a call expression.
	if call, ok := n.(*ast.CallExpr); ok {
		w.visitCall(call)
	}
	return w
}

func (w *errorCheckWalker) visitCall(call *ast.CallExpr) {
	callee := calleeName(call, w.info)
	if callee == "" {
		return
	}
	if !returnsError(call, w.info) {
		return
	}
	checked := w.isChecked(call)
	line := w.fset.Position(call.Pos()).Line
	*w.sites = append(*w.sites, patterns.ErrorCheckSite{
		Line:    line,
		Callee:  callee,
		Checked: checked,
		// FuncID is set by the caller (needs the full ID).
	})
}

// isChecked determines if the error from a call was checked.
// Returns false if assigned to `_` or result ignored, true otherwise.
func (w *errorCheckWalker) isChecked(call *ast.CallExpr) bool {
	switch parent := w.callParent().(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BinaryExpr:
		return parent.Op == token.EQL || parent.Op == token.NEQ
	case *ast.AssignStmt:
		return w.assignedErrorChecked(call, parent)
	default:
		return false
	}
}

// Parentheses change source grouping, not how the error result is consumed.
func (w *errorCheckWalker) callParent() ast.Node {
	for i := len(w.stack) - 2; i >= 0; i-- {
		if _, grouped := w.stack[i].(*ast.ParenExpr); !grouped {
			return w.stack[i]
		}
	}
	return nil
}

func (w *errorCheckWalker) assignedErrorChecked(call *ast.CallExpr, assign *ast.AssignStmt) bool {
	positions := errorCallPositions(w.info.TypeOf(call))
	offset := errorAssignmentOffset(assign, call)
	for _, pos := range positions {
		pos += offset
		if pos >= len(assign.Lhs) {
			return false
		}
		if !w.errorBindingUsed(assign.Lhs[pos], call.End()) {
			return false
		}
	}
	return len(positions) > 0
}

func (w *errorCheckWalker) errorBindingUsed(expr ast.Expr, after token.Pos) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name != "_" && meaningfulErrorUse(w.body, w.info.ObjectOf(id), w.info, after)
}

func errorCallPositions(t types.Type) []int {
	if tuple, ok := t.(*types.Tuple); ok {
		var positions []int
		for i := 0; i < tuple.Len(); i++ {
			if isErrorTypeCheck(tuple.At(i).Type()) {
				positions = append(positions, i)
			}
		}
		return positions
	}
	if isErrorTypeCheck(t) {
		return []int{0}
	}
	return nil
}

func meaningfulErrorUse(body ast.Node, object types.Object, info *types.Info, after token.Pos) bool {
	if object == nil {
		return false
	}
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		for _, expr := range errorUseOperands(n) {
			if errorUseMatches(expr, object, info, after) {
				found = true
			}
		}
		return true
	})
	return found
}

func errorUseOperands(n ast.Node) []ast.Expr {
	switch n := n.(type) {
	case *ast.BinaryExpr:
		return []ast.Expr{n.X, n.Y}
	case *ast.CallExpr:
		return n.Args
	case *ast.ReturnStmt:
		return n.Results
	}
	return nil
}

// findErrorCheckSites finds call sites and tracks whether the error was checked.
// Used for deviant_behavior mining (Engler et al., SOSP 2001).
func findErrorCheckSites(fn *ast.FuncDecl, fset *token.FileSet, info *types.Info) []patterns.ErrorCheckSite {
	if fn.Body == nil {
		return nil
	}
	var sites []patterns.ErrorCheckSite
	// Use a custom walker that tracks parents.
	w := &errorCheckWalker{
		fset:  fset,
		info:  info,
		sites: &sites,
		body:  fn.Body,
	}
	ast.Walk(w, fn.Body)
	return sites
}

// calleeName extracts the called function name from a call expression.
func calleeName(call *ast.CallExpr, info *types.Info) string {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name
	case *ast.SelectorExpr:
		if ident, ok := fun.X.(*ast.Ident); ok {
			return ident.Name + "." + fun.Sel.Name
		}
		return fun.Sel.Name
	}
	return ""
}

// returnsError checks if a call returns an error (via type info).
func returnsError(call *ast.CallExpr, info *types.Info) bool {
	if info == nil {
		return false
	}
	tv, ok := info.Types[call]
	if !ok {
		return false
	}
	if tup, ok := tv.Type.(*types.Tuple); ok {
		for i := 0; i < tup.Len(); i++ {
			if isErrorTypeCheck(tup.At(i).Type()) {
				return true
			}
		}
	}
	return isErrorTypeCheck(tv.Type)
}

// isErrorTypeCheck checks if a type is the error interface.
func isErrorTypeCheck(t types.Type) bool {
	if named, ok := types.Unalias(t).(*types.Named); ok {
		obj := named.Obj()
		return obj.Name() == "error" && obj.Pkg() == nil
	}
	return false
}

func errorAssignmentOffset(assign *ast.AssignStmt, call *ast.CallExpr) int {
	if len(assign.Rhs) == 1 {
		return 0
	}
	for i, expr := range assign.Rhs {
		if specUnwrapParens(expr) == call {
			return i
		}
	}
	return len(assign.Lhs)
}

func errorUseMatches(expr ast.Expr, object types.Object, info *types.Info, after token.Pos) bool {
	id, ok := specUnwrapParens(expr).(*ast.Ident)
	return ok && id.Pos() > after && info.ObjectOf(id) == object
}
