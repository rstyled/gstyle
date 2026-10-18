package gopatterns

import (
	"go/ast"
	"go/token"

	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
)

// Source spans connect typed acquisition/report boundaries to canonical nodes.
// All resource dependency and path queries live in resourceflow.
type releaseGraph struct {
	graph *pdg.Graph
	query *resourceflow.Query
	nodes map[outputSpan]pdg.Ref
	fset  *token.FileSet
}

func newReleaseGraph(graph *pdg.Graph, fset *token.FileSet) (*releaseGraph, error) {
	query, err := resourceflow.New(graph)
	if err != nil {
		return nil, err
	}
	if query == nil {
		if _, err := deferflow.Sites(graph); err != nil {
			return nil, err
		}
	}
	return &releaseGraph{graph: graph, query: query, nodes: releaseSourceNodes(graph), fset: fset}, nil
}

func (g *releaseGraph) ref(node ast.Node) pdg.Ref {
	if node == nil {
		return 0
	}
	start, end := g.fset.PositionFor(node.Pos(), false).Offset, g.fset.PositionFor(node.End(), false).Offset
	return g.nodes[outputSpan{uint32(start), uint32(end)}]
}

func (g *releaseGraph) writtenSymbol(ref pdg.Ref, position pdg.Position) pdg.Ref {
	if ref == 0 {
		return 0
	}
	for _, write := range g.graph.NodeAttributes(g.graph.Nodes[ref-1]).Writes {
		if write.Position == position && write.Symbol != 0 {
			return write.Symbol
		}
	}
	return 0
}

func releaseSourceNodes(graph *pdg.Graph) map[outputSpan]pdg.Ref {
	nodes := map[outputSpan]pdg.Ref{}
	for i, node := range graph.Nodes {
		if node.Span == 0 || node.OriginalKind.Status != pdg.Known {
			continue
		}
		span := graph.Spans[node.Span-1]
		nodes[outputSpan{span.Start, span.End}] = pdg.Ref(i + 1)
	}
	return nodes
}

func releaseGraphForContext(graph *pdg.Graph, fset *token.FileSet, ctx extractCtx) (*releaseGraph, error) {
	query := ctx.resourceQueries[graph]
	if query == nil {
		return newReleaseGraph(graph, fset)
	}
	return &releaseGraph{graph: graph, query: query, nodes: releaseSourceNodes(graph), fset: fset}, nil
}
