package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/errorflow"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

type errorCheckWalker struct {
	successOutput      func(*ast.CallExpr) bool
	checks             *errorUseFacts
	successReturnSites []int
	fset               *token.FileSet
	info               *types.Info
	sites              *[]patterns.ErrorCheckSite
	stack              []ast.Node // Parent stack
}

func (w *errorCheckWalker) Visit(n ast.Node) ast.Visitor {
	if _, closure := n.(*ast.FuncLit); closure {
		return nil
	}
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
	if !returnsError(call, w.info) {
		return
	}
	checked := w.isChecked(call)
	callee := w.checks.callee(call)
	if callee == "" {
		callee = calleeName(call, w.info)
		checked.Known = false
	}
	line := w.fset.PositionFor(call.Pos(), false).Line
	discarded := w.discardsOutput(call)
	*w.sites = append(*w.sites, patterns.ErrorCheckSite{
		Line:            line,
		Callee:          callee,
		Checked:         checked.Used,
		CheckUnknown:    !checked.Known,
		DiscardedOutput: discarded,
		// FuncID is set by the caller (needs the full ID).
	})
	if discarded && w.successOutput != nil && w.successOutput(call) {
		w.successReturnSites = append(w.successReturnSites, len(*w.sites)-1)
	}
}

// isChecked determines if the error from a call was checked.
// Returns false if assigned to `_` or result ignored, true otherwise.
func (w *errorCheckWalker) isChecked(call *ast.CallExpr) errorflow.Result {
	if !w.checks.known(call) {
		return errorflow.Result{}
	}
	switch parent := w.callParent().(type) {
	case *ast.AssignStmt:
		return w.errorTargetsChecked(parent, parent.Lhs, errorCallPositions(w.info.TypeOf(call)), errorAssignmentOffset(parent, call))
	case *ast.ValueSpec:
		return w.errorTargetsChecked(parent, memoryDeclaredTargets(parent.Names), errorCallPositions(w.info.TypeOf(call)), errorValueOffset(parent.Values, call))
	default:
		return w.checks.valueUsed(call)
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

func (w *errorCheckWalker) errorTargetsChecked(owner ast.Node, targets []ast.Expr, positions []int, offset int) errorflow.Result {
	result := errorflow.Result{Used: len(positions) > 0, Known: true}
	for _, position := range positions {
		result = result.All(w.errorTargetUse(owner, targets, position+offset))
		if result.Unused() {
			return result
		}
	}
	return result
}

func (w *errorCheckWalker) errorTargetUse(owner ast.Node, targets []ast.Expr, position int) errorflow.Result {
	if position >= len(targets) {
		return errorflow.Result{}
	}
	identifier, ok := unparen(targets[position]).(*ast.Ident)
	if !ok {
		return errorflow.Result{}
	}
	if identifier.Name == "_" {
		return errorflow.Result{Known: true}
	}
	return w.checks.assignmentUsed(owner, pdg.Position(position+1))
}

func (w errorCheckWalker) scan(fn *ast.FuncDecl) ([]patterns.ErrorCheckSite, []int) {
	if fn.Body == nil {
		return nil, nil
	}
	var sites []patterns.ErrorCheckSite
	w.sites = &sites
	w.stack = nil
	w.successReturnSites = nil
	ast.Walk(&w, fn.Body)
	return sites, w.successReturnSites
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

// calleeName extracts the called function name from a call expression.
func calleeName(call *ast.CallExpr, info *types.Info) string {
	switch fun := specUnwrapParens(call.Fun).(type) {
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
	if info.Types[call.Fun].IsType() {
		return false
	}
	value, known := info.Types[call]
	return known && len(errorCallPositions(value.Type)) > 0
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
	return errorValueOffset(assign.Rhs, call)
}

func errorValueOffset(values []ast.Expr, call *ast.CallExpr) int {
	if len(values) == 1 {
		return 0
	}
	for index, expression := range values {
		if specUnwrapParens(expression) == call {
			return index
		}
	}
	return len(values)
}
