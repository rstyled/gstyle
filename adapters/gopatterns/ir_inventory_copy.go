package gopatterns

import (
	"maps"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

// summaryInventory borrows compiler/source facts and the intern pool. Dependency
// passes own their mutable inventory and never overwrite the final builder.
func (b *irBuilder) summaryInventory() *irBuilder {
	copy := *b
	copy.nodes = maps.Clone(b.nodes)
	copy.symbols = maps.Clone(b.symbols)
	copy.logicalPredicates = maps.Clone(b.logicalPredicates)
	copy.rangeBounds = maps.Clone(b.rangeBounds)
	copy.graph.Nodes = slices.Clone(b.graph.Nodes)
	copy.graph.Spans = slices.Clone(b.graph.Spans)
	copy.graph.Edges = slices.Clone(b.graph.Edges)
	copy.graph.Definitions = slices.Clone(b.graph.Definitions)
	copy.graph.Symbols = slices.Clone(b.graph.Symbols)
	copy.graph.Locations = slices.Clone(b.graph.Locations)
	copy.graph.Extensions = slices.Clone(b.graph.Extensions)
	copy.attributes = make([]pdg.Attributes, len(b.attributes))
	for i, attrs := range b.attributes {
		copy.attributes[i] = copyInventoryAttributes(attrs)
	}
	return &copy
}

func copyInventoryAttributes(attrs pdg.Attributes) pdg.Attributes {
	attrs.Symbols = slices.Clone(attrs.Symbols)
	attrs.Defines = slices.Clone(attrs.Defines)
	attrs.Reads = slices.Clone(attrs.Reads)
	attrs.Writes = slices.Clone(attrs.Writes)
	attrs.Outcomes = slices.Clone(attrs.Outcomes)
	attrs.Extensions = slices.Clone(attrs.Extensions)
	return attrs
}
