package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
)

func (b *irBuilder) scalarReadOwner(id *ast.Ident) pdg.Ref {
	return b.addNode(id, pdg.Read, pdg.Attributes{})
}

func (b *irBuilder) implicitScalarRead(owner, source ast.Node, read pdg.Access) {
	span := b.graph.Nodes[b.nodes[owner]-1].Span
	if source != nil {
		span = b.scalarSourceSpan(source)
	}
	ref := b.scalarEntity(span, pdg.Read, pdg.Attributes{Symbols: []pdg.Ref{read.Symbol}, Reads: []pdg.Access{read}})
	b.implicitReads[owner] = append(b.implicitReads[owner], ref)
}
