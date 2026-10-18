package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"strconv"
)

// A region bounds operand evaluation within one source statement. It does not
// assert a total operand order. IDs are local traversal ordinals, not positions.
func (b *irBuilder) evaluationRegionFacts(body *ast.BlockStmt) {
	stack := []int{0}
	next := 0
	seen := map[pdg.Ref]bool{}
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		group := stack[len(stack)-1]
		if _, statement := node.(ast.Stmt); statement {
			next++
			group = next
		}
		b.evaluationRegionNode(node, group, seen)
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		stack = append(stack, group)
		return true
	})
	b.invocationOrderFacts(body)
}
func (b *irBuilder) evaluationRegionNode(source ast.Node, group int, seen map[pdg.Ref]bool) {
	refs := append([]pdg.Ref{b.nodes[source]}, b.implicitReads[source]...)
	for _, ref := range refs {
		b.evaluationRegionRef(ref, group, seen)
	}
}
func (b *irBuilder) evaluationRegionRef(ref pdg.Ref, group int, seen map[pdg.Ref]bool) {
	if ref == 0 || seen[ref] {
		return
	}
	seen[ref] = true
	attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle.go:evaluation-region", strconv.Itoa(group))...)
}
