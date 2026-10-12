package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"strconv"
)

// Publish a complete derived entity in one step. Reuse the value-site span;
// no unused function-wide span or partially initialized node is allocated.
func (b *irBuilder) scalarEntity(span pdg.Ref, category pdg.Category, attrs pdg.Attributes) pdg.Ref {
	ref := pdg.Ref(len(b.graph.Nodes) + 1)
	b.attributes = append(b.attributes, attrs)
	b.graph.Nodes = append(b.graph.Nodes, pdg.Node{
		ID: b.pool.Text("n" + strconv.Itoa(int(ref))), Category: category, Span: span, Attributes: pdg.Ref(len(b.attributes)), Synthetic: true,
		OriginalKind: b.pool.Missing(pdg.NotApplicable, "Derived scalar value/read/predicate entity; its span identifies the source value site."),
		Evidence:     b.pool.Evidence(pdg.Supported, scalarDefinitionDescription), Provenance: b.pool.Origin(pdg.Analysis, scalarDefinitionDescription),
	})
	return ref
}

func (b *irBuilder) scalarSourceSpan(source ast.Node) pdg.Ref {
	if ref := b.nodes[source]; ref != 0 {
		return b.graph.Nodes[ref-1].Span
	}
	b.graph.Spans = append(b.graph.Spans, b.span(source))
	return pdg.Ref(len(b.graph.Spans))
}
