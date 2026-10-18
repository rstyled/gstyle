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
		value := nilErrValue{Empty: emptyNilErrResult(ast.Unparen(element), info), Nil: info.Types[ast.Unparen(element)].IsNil()}
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
