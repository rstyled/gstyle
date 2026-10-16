package gopatterns

import (
	"go/ast"
	"go/constant"
	"go/types"
)

// live means a path still has the checked failure with no unknown handling.
// Unknown paths are omitted; they are not claimed to handle the error safely.
type nilErrFlow struct{ lost, live bool }
type nilErrEvidence struct {
	checked   ast.Expr
	positions []int
	info      *types.Info
}

func nilErrLostPath(branch ast.Stmt, checked ast.Expr, positions []int, info *types.Info) bool {
	// Only stable name/field identities are tracked by this source-local pass.
	if !sameNilErrValue(checked, checked, info) {
		return false
	}
	evidence := nilErrEvidence{checked: checked, positions: positions, info: info}
	return evidence.statement(branch).lost
}

func (e nilErrEvidence) block(statements []ast.Stmt) nilErrFlow {
	out := nilErrFlow{live: true}
	for _, statement := range statements {
		if !out.live {
			break
		}
		step := e.statement(statement)
		out.lost = out.lost || step.lost
		out.live = step.live
	}
	return out
}

func (e nilErrEvidence) statement(statement ast.Stmt) nilErrFlow {
	switch statement := statement.(type) {
	case *ast.BlockStmt:
		return e.block(statement.List)
	case *ast.ReturnStmt:
		return nilErrFlow{lost: nilErrorResults(statement, e.positions, e.info) && emptyNilErrResults(statement, e.info)}
	case *ast.IfStmt:
		return e.conditional(statement)
	case *ast.ExprStmt, *ast.AssignStmt, *ast.DeclStmt, *ast.EmptyStmt:
		return nilErrFlow{live: !e.unknownHandling(statement)}
	default:
		return nilErrFlow{}
	}
}

func (e nilErrEvidence) conditional(statement *ast.IfStmt) nilErrFlow {
	if e.checkedOperation(statement.Init) || e.unknownHandling(statement.Init) || e.mentions(statement.Cond) {
		return nilErrFlow{}
	}
	then := e.statement(statement.Body)
	other := nilErrFlow{live: true}
	if statement.Else != nil {
		other = e.statement(statement.Else)
	}
	return then.merge(other)
}

func emptyNilErrResults(statement *ast.ReturnStmt, info *types.Info) bool {
	for _, result := range statement.Results {
		if !emptyNilErrResult(ast.Unparen(result), info) {
			return false
		}
	}
	return true
}

func emptyNilErrResult(expr ast.Expr, info *types.Info) bool {
	if info.Types[expr].IsNil() {
		return true
	}
	if literal, ok := expr.(*ast.CompositeLit); ok {
		return len(literal.Elts) == 0
	}
	value := info.Types[expr].Value
	if value == nil {
		return false
	}
	switch value.Kind() {
	case constant.Bool:
		return !constant.BoolVal(value)
	case constant.String:
		return constant.StringVal(value) == ""
	default:
		return constant.Sign(value) == 0
	}
}

func (e nilErrEvidence) mentions(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		if expr, ok := node.(ast.Expr); ok && sameNilErrValue(expr, e.checked, e.info) {
			found = true
		}
		return !found
	})
	return found
}

func sameNilErrValue(left, right ast.Expr, info *types.Info) bool {
	left, right = ast.Unparen(left), ast.Unparen(right)
	switch left := left.(type) {
	case *ast.Ident:
		other, ok := right.(*ast.Ident)
		return ok && sameNilErrObject(info.ObjectOf(left), info.ObjectOf(other))
	case *ast.SelectorExpr:
		other, ok := right.(*ast.SelectorExpr)
		return ok && sameNilErrSelector(left, other, info)
	default:
		return false
	}
}

func (e nilErrEvidence) unknownHandling(node ast.Node) bool {
	if node == nil {
		return false
	}
	unknown := false
	ast.Inspect(node, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		unknown = unknown || e.unknownOperation(node)
		return !unknown
	})
	return unknown
}

func (e nilErrEvidence) unknownOperation(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.CallExpr:
		return !nilErrLoggingCall(node, e.info) && e.mentions(node)
	case *ast.AssignStmt:
		return e.replacesCheckedValue(node.Lhs) || e.copiesCheckedValue(node)
	case *ast.ValueSpec:
		return e.mentions(node)
	default:
		return false
	}
}

func (e nilErrEvidence) replacesCheckedValue(targets []ast.Expr) bool {
	for _, target := range targets {
		if replacesNilErrValue(target, e.checked, e.info) {
			return true
		}
	}
	return false
}

// Logging records a failure; it does not recover from it.
func nilErrLoggingCall(call *ast.CallExpr, info *types.Info) bool {
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		builtin, ok := info.ObjectOf(fun).(*types.Builtin)
		return ok && (builtin.Name() == "print" || builtin.Name() == "println")
	case *ast.SelectorExpr:
		return nilErrLoggingFunction(info.ObjectOf(fun.Sel))
	default:
		return false
	}
}

func nilErrLoggingFunction(object types.Object) bool {
	function, ok := object.(*types.Func)
	if !ok || function.Pkg() == nil {
		return false
	}
	switch function.Pkg().Path() {
	case "log/slog":
		return true
	case "log":
		return nilErrPrintName(function.Name())
	case "fmt":
		return nilErrFmtPrintName(function.Name())
	default:
		return false
	}
}

func nilErrPrintName(name string) bool {
	return name == "Print" || name == "Printf" || name == "Println" || name == "Output"
}

func (e nilErrEvidence) copiesCheckedValue(assign *ast.AssignStmt) bool {
	for i, value := range assign.Rhs {
		if i >= len(assign.Lhs) || !e.mentions(value) {
			continue
		}
		target, ok := ast.Unparen(assign.Lhs[i]).(*ast.Ident)
		if !ok || target.Name != "_" {
			return true
		}
	}
	return false
}

func sameNilErrObject(left, right types.Object) bool {
	return left != nil && left == right
}

func sameNilErrSelector(left, right *ast.SelectorExpr, info *types.Info) bool {
	return sameNilErrObject(info.ObjectOf(left.Sel), info.ObjectOf(right.Sel)) && sameNilErrValue(left.X, right.X, info)
}

func nilErrFmtPrintName(name string) bool {
	return name == "Print" || name == "Printf" || name == "Println" || name == "Fprint" || name == "Fprintf" || name == "Fprintln"
}

func replacesNilErrValue(target, checked ast.Expr, info *types.Info) bool {
	if sameNilErrValue(target, checked, info) {
		return true
	}
	selector, ok := ast.Unparen(checked).(*ast.SelectorExpr)
	return ok && replacesNilErrValue(target, selector.X, info)
}

// A checked alternative operation may recover without consuming the original
// error. Its recovery contract is unknown, so do not carry failure evidence on.
func (e nilErrEvidence) checkedOperation(node ast.Node) bool {
	if node == nil {
		return false
	}
	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		if call, ok := node.(*ast.CallExpr); ok {
			found = returnsError(call, e.info)
		}
		return !found
	})
	return found
}

func (flow nilErrFlow) merge(other nilErrFlow) nilErrFlow {
	return nilErrFlow{lost: flow.lost || other.lost, live: flow.live || other.live}
}
