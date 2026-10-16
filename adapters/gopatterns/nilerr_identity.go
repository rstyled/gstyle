package gopatterns

import (
	"go/ast"
	"go/types"
	"slices"
)

type nilErrLocation struct {
	root   types.Object
	fields []int
}

// Selection.Index includes promoted fields, so p.Err and p.Embedded.Err share
// one location. Pointer aliases are not equated by their spelling.
func nilErrLocationOf(expr ast.Expr, info *types.Info) nilErrLocation {
	switch expr := ast.Unparen(expr).(type) {
	case *ast.Ident:
		return nilErrLocation{root: info.ObjectOf(expr)}
	case *ast.SelectorExpr:
		return nilErrFieldLocation(expr, info)
	case *ast.CallExpr:
		return nilErrConversionLocation(expr, info)
	default:
		return nilErrLocation{}
	}
}

func nilErrFieldLocation(expr *ast.SelectorExpr, info *types.Info) nilErrLocation {
	selection := info.Selections[expr]
	if selection == nil {
		return nilErrLocation{root: info.ObjectOf(expr.Sel)}
	}
	if selection.Kind() != types.FieldVal {
		return nilErrLocation{}
	}
	parent := nilErrLocationOf(expr.X, info)
	parent.fields = append(parent.fields, selection.Index()...)
	return parent
}

func nilErrConversionLocation(call *ast.CallExpr, info *types.Info) nilErrLocation {
	if !info.Types[call.Fun].IsType() || len(call.Args) != 1 {
		return nilErrLocation{}
	}
	if !types.Identical(info.TypeOf(call), info.TypeOf(call.Args[0])) {
		return nilErrLocation{}
	}
	return nilErrLocationOf(call.Args[0], info)
}

func sameNilErrValue(left, right ast.Expr, info *types.Info) bool {
	a, b := nilErrLocationOf(left, info), nilErrLocationOf(right, info)
	return a.root != nil && a.root == b.root && slices.Equal(a.fields, b.fields)
}

func replacesNilErrValue(target, checked ast.Expr, info *types.Info) bool {
	if pointer, ok := ast.Unparen(target).(*ast.StarExpr); ok {
		target = pointer.X
	}
	a, b := nilErrLocationOf(target, info), nilErrLocationOf(checked, info)
	if a.root == nil || a.root != b.root || len(a.fields) > len(b.fields) {
		return false
	}
	return slices.Equal(a.fields, b.fields[:len(a.fields)])
}
