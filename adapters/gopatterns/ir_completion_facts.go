package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"go/token"
	"go/types"
)

func (b *irBuilder) runtimeCompletionFacts(expression ast.Expr, attrs *pdg.Attributes) {
	if normalExpressionOperation(expression, b.info) || b.pool.Tables.Text(attrs.Operator.Value) == "function-value" {
		b.runtimeExtension(attrs, "normal-completion", "true")
	}
	if runtimeNonNil(expression, b.info) {
		b.runtimeExtension(attrs, "non-nil", "true")
	}
}

// This fact concerns only the operation itself. Operand evaluation, calls,
// memory accesses, deferred work and control paths remain separate PDG facts.
func normalExpressionOperation(expression ast.Expr, info *types.Info) bool {
	value := info.Types[expression]
	if value.Value != nil || value.IsNil() {
		return true
	}
	return normalTypedOperation(expression, info)
}

func normalTypedOperation(expression ast.Expr, info *types.Info) bool {
	switch expression := expression.(type) {
	case *ast.CallExpr:
		return normalConversionOperation(expression, info)
	case *ast.Ident, *ast.ParenExpr:
		return true
	case *ast.UnaryExpr:
		return normalUnaryOperation(expression.Op)
	case *ast.BinaryExpr:
		return normalBinaryOperation(expression, info)
	case *ast.TypeAssertExpr:
		return expression.Type == nil
	default:
		return false
	}
}

func normalUnaryOperation(operator token.Token) bool {
	switch operator {
	case token.ADD, token.SUB, token.NOT, token.XOR, token.AND:
		return true
	default:
		return false
	}
}

func normalBinaryOperation(expression *ast.BinaryExpr, info *types.Info) bool {
	switch expression.Op {
	case token.QUO, token.REM, token.SHL, token.SHR:
		return false
	case token.EQL, token.NEQ:
		return normalEquality(expression.X, expression.Y, info)
	default:
		return true
	}
}

func normalEquality(left, right ast.Expr, info *types.Info) bool {
	if info.Types[left].IsNil() || info.Types[right].IsNil() {
		return true
	}
	return strictComparable(info.TypeOf(left)) && strictComparable(info.TypeOf(right))
}

func strictComparable(typ types.Type) bool {
	if typ == nil {
		return false
	}
	switch typ := typ.Underlying().(type) {
	case *types.Basic, *types.Pointer, *types.Chan:
		return true
	case *types.Array:
		return strictComparable(typ.Elem())
	case *types.Struct:
		return strictComparableFields(typ)
	default:
		return false
	}
}

func strictComparableFields(typ *types.Struct) bool {
	for index := 0; index < typ.NumFields(); index++ {
		if !strictComparable(typ.Field(index).Type()) {
			return false
		}
	}
	return true
}

// These conversions preserve nilness or box a concrete value. Slice-to-array
// conversions can panic and do not enter this set.
func normalConversionOperation(call *ast.CallExpr, info *types.Info) bool {
	if len(call.Args) != 1 || !info.Types[call.Fun].IsType() {
		return false
	}
	if info.Types[call.Args[0]].IsNil() {
		return true
	}
	if types.Identical(info.TypeOf(call), info.TypeOf(call.Args[0])) {
		return true
	}
	return conversionNilness(memoryTypeKind(info.TypeOf(call.Args[0])), memoryTypeKind(info.TypeOf(call))) != "unknown"
}
