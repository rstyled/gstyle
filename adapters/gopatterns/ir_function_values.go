package gopatterns

import (
	"encoding/json"
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

// Function values name declarations; they do not resolve invocation effects.
// Direct call targets already have Callee identity and need no extra value node.
func (b *irBuilder) functionValueFacts(body *ast.BlockStmt) {
	callees := map[ast.Node]bool{}
	ast.Inspect(body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		if call, ok := node.(*ast.CallExpr); ok {
			b.skipStaticCallee(call, callees)
		}
		if selector, ok := node.(*ast.SelectorExpr); ok {
			callees[selector.Sel] = true // A selector name is not an evaluated expression.
		}
		expression, ok := node.(ast.Expr)
		if !ok || callees[node] {
			return true
		}
		b.functionValue(expression)
		return true
	})
}
func (b *irBuilder) skipStaticCallee(call *ast.CallExpr, callees map[ast.Node]bool) {
	if _, static := b.callee(call.Fun).(*types.Func); !static {
		return
	}
	if !b.declarationValue(call.Fun) {
		return // A bound receiver can evaluate nested arguments and calls.
	}
	ast.Inspect(call.Fun, func(node ast.Node) bool { callees[node] = true; return true })
}
func (b *irBuilder) functionValue(expression ast.Expr) {
	function := b.functionDeclaration(expression)
	if function == nil {
		return
	}
	data, _ := json.Marshal(patterns.FuncID(function))
	b.addNode(expression, pdg.Computation, pdg.Attributes{Operator: b.pool.Known("function-value"), Extensions: b.pool.SingletonExtension("gstyle.go:function-value", string(data))})
}
func (b *irBuilder) functionDeclaration(expression ast.Expr) *types.Func {
	if b.info.TypeOf(expression) == nil || b.info.Types[expression].IsType() {
		return nil
	}
	function, ok := b.callee(expression).(*types.Func)
	if !ok || !b.declarationValue(expression) {
		return nil
	}
	// Parentheses retain their operand instead of becoming a second terminal.
	if _, parentheses := expression.(*ast.ParenExpr); parentheses {
		return nil
	}
	return function
}
func (b *irBuilder) declarationValue(expression ast.Expr) bool {
	if selector, ok := unparen(expression).(*ast.SelectorExpr); ok {
		selection := b.info.Selections[selector]
		return selection == nil || selection.Kind() == types.MethodExpr
	}
	switch expression := expression.(type) {
	case *ast.IndexExpr:
		return b.declarationValue(expression.X)
	case *ast.IndexListExpr:
		return b.declarationValue(expression.X)
	default:
		return true
	}
}
