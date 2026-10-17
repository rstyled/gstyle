package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"strconv"
)

// Postorder records Go's lexical invocation order, including calls in a callee
// expression before argument calls. It does not order independent non-call work.
func (b *irBuilder) invocationOrderFacts(body *ast.BlockStmt) {
	var stack []ast.Node
	next := 0
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil {
			source := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			b.invocationOrderNode(source, &next)
			return true
		}
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		stack = append(stack, node)
		return true
	})
}
func (b *irBuilder) invocationOrderNode(source ast.Node, next *int) {
	if _, call := source.(*ast.CallExpr); !call {
		return
	}
	ref := b.nodes[source]
	if ref == 0 || b.graph.Nodes[ref-1].Category != pdg.Call {
		return
	}
	*next = *next + 1
	attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle.go:invocation-order", strconv.Itoa(*next))...)
}
