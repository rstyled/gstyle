package gopatterns

import (
	"go/ast"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/cfg"
)

// x/tools materializes a normal fallthrough return at the closing brace. It
// belongs to the same canonical flow as explicit returns, without invented
// source text or a lexical end-of-block inference in downstream consumers.
func (b *irBuilder) implicitReturnFacts(flow *cfg.CFG) {
	if len(b.graph.Function.Outputs) != 0 {
		return
	}
	for _, block := range flow.Blocks {
		returned := block.Return()
		if returned == nil || b.nodes[returned] != 0 {
			continue
		}
		b.implicitReturn(returned)
	}
}
func (b *irBuilder) implicitReturn(returned *ast.ReturnStmt) {
	offset := uint32(b.fset.PositionFor(returned.Pos(), false).Offset)
	b.graph.Spans = append(b.graph.Spans, pdg.Span{Source: 1, Start: offset, End: offset})
	ref := b.scalarEntity(pdg.Ref(len(b.graph.Spans)), pdg.Return, pdg.Attributes{Construct: b.pool.Text("implicit-return"), ExitKind: pdg.NormalExit, Description: b.pool.Text("Compiler CFG normal fallthrough return at the closing brace of a void function.")})
	b.nodes[returned] = ref
}
