package gopatterns

import (
	"go/ast"
	"go/types"
)

// These facts describe a successfully produced value. Operand evaluation and
// allocation/call faults remain separate operation and invocation effects.
func (b *irBuilder) canonicalNonNilValue(expr ast.Expr) bool {
	if runtimeNonNil(expr, b.info) {
		return true
	}
	if b.functionDeclaration(expr) != nil {
		return true
	}
	switch expr := expr.(type) {
	case *ast.CompositeLit:
		kind := memoryTypeKind(b.info.TypeOf(expr))
		return kind == "slice" || kind == "map"
	case *ast.CallExpr:
		return nonNilMake(b.callee(expr.Fun))
	default:
		return false
	}
}

func nonNilMake(object types.Object) bool {
	builtin, ok := object.(*types.Builtin)
	return ok && builtin.Name() == "make"
}
