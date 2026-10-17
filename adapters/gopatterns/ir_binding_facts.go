package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
)

// Binding-only lowering needs explicit nil operands and declaration zero facts
// too. These compiler facts do not enable the broader runtime profile.
func (b *irBuilder) bindingSourceFacts(fn *ast.FuncDecl) {
	if !b.bindingOperations {
		return
	}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, literal := node.(*ast.FuncLit); literal {
			return false
		}
		b.runtimeZeroDeclaration(node)
		expr, ok := node.(ast.Expr)
		if !ok || !b.info.Types[expr].IsNil() {
			return true
		}
		ref := b.nodes[node]
		if ref == 0 {
			ref = b.addNode(node, pdg.Computation, pdg.Attributes{Operator: b.pool.Known("literal"), Literal: b.pool.Known("nil")})
		}
		attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
		b.runtimeConstant(expr, attrs)
		b.runtimeCompletionFacts(expr, attrs)
		return true
	})
}
