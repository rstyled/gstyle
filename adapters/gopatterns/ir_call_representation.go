package gopatterns

import (
	"go/ast"
	"go/types"
)

// Assignment representations describe each expanded receiving value slot.
// The call shape separately describes tuple expansion and variadic packing.
func (b *irBuilder) callRepresentations(call *ast.CallExpr) ([]types.Type, []ast.Expr) {
	signature := b.callSignature(call)
	if signature == nil {
		return nil, nil
	}
	targets := make([]types.Type, b.callArgumentCount(call))
	for i := range targets {
		targets[i] = callParameterType(signature, i, call.Ellipsis.IsValid())
	}
	return targets, call.Args
}
