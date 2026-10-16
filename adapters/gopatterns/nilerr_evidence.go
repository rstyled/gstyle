package gopatterns

import (
	"go/ast"
	"go/constant"
	"go/types"
)

func emptyNilErrResult(expr ast.Expr, info *types.Info) bool {
	if info.Types[expr].IsNil() {
		return true
	}
	if literal, ok := expr.(*ast.CompositeLit); ok {
		return emptyNilErrLiteral(literal, info)
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
		return !nilErrLoggingCall(node, e.info) && !e.info.Types[node.Fun].IsType()
	case *ast.AssignStmt:
		return e.unknownAssignment(node)
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

func nilErrFmtPrintName(name string) bool {
	return name == "Print" || name == "Printf" || name == "Println" || name == "Fprint" || name == "Fprintf" || name == "Fprintln"
}

func nilErrNamedTargets(targets []ast.Expr) bool {
	for _, target := range targets {
		if _, ok := ast.Unparen(target).(*ast.Ident); !ok {
			return false
		}
	}
	return true
}

func emptyNilErrLiteral(literal *ast.CompositeLit, info *types.Info) bool {
	if len(literal.Elts) == 0 {
		return true
	}
	switch info.TypeOf(literal).Underlying().(type) {
	case *types.Array, *types.Struct:
		return emptyNilErrElements(literal, info)
	default:
		return false
	}
}

func emptyNilErrElements(literal *ast.CompositeLit, info *types.Info) bool {
	for i, element := range literal.Elts {
		typ := nilErrElementType(literal, element, i, info)
		if pair, ok := element.(*ast.KeyValueExpr); ok {
			element = pair.Value
		}
		value := nilErrValue{empty: emptyNilErrResult(ast.Unparen(element), info), nilValue: info.Types[ast.Unparen(element)].IsNil()}
		if !nilErrAssignableZero(value, typ) {
			return false
		}
	}
	return true
}

func nilErrElementType(literal *ast.CompositeLit, element ast.Expr, index int, info *types.Info) types.Type {
	switch typ := info.TypeOf(literal).Underlying().(type) {
	case *types.Array:
		return typ.Elem()
	case *types.Struct:
		if pair, ok := element.(*ast.KeyValueExpr); ok {
			key := pair.Key.(*ast.Ident)
			return info.ObjectOf(key).Type()
		}
		return typ.Field(index).Type()
	default:
		return info.TypeOf(element)
	}
}

func (e nilErrEvidence) unknownAssignment(statement *ast.AssignStmt) bool {
	return !nilErrNamedTargets(statement.Lhs) || e.replacesCheckedValue(statement.Lhs) || e.copiesCheckedValue(statement)
}
