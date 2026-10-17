package gopatterns

import (
	"github.com/rstyled/gstyle/domain/errorflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"go/token"
)

type errorUseFacts struct {
	graph *pdg.Graph
	query *errorflow.Query
	nodes map[outputSpan]pdg.Ref
	fset  *token.FileSet
}

func newErrorUseFacts(graph *pdg.Graph, fset *token.FileSet) (*errorUseFacts, error) {
	query, err := errorflow.New(graph)
	if err != nil {
		return nil, err
	}
	facts := &errorUseFacts{graph: graph, query: query, nodes: map[outputSpan]pdg.Ref{}, fset: fset}
	for index, node := range graph.Nodes {
		if node.OriginalKind.Status != pdg.Known || node.Span == 0 {
			continue
		}
		span := graph.Spans[node.Span-1]
		facts.nodes[outputSpan{span.Start, span.End}] = pdg.Ref(index + 1)
	}
	return facts, nil
}

// callee uses declaration identity from the canonical call, never source aliases.
func (f *errorUseFacts) callee(call *ast.CallExpr) string {
	ref := f.node(call)
	if ref == 0 {
		return ""
	}
	node := f.graph.Nodes[ref-1]
	attrs := f.graph.NodeAttributes(node)
	if node.Category != pdg.Call || attrs.CallForm != pdg.DirectCall || attrs.Callee.Status != pdg.Known {
		return ""
	}
	return f.graph.Tables.Text(attrs.Callee.Value)
}
func (f *errorUseFacts) node(node ast.Node) pdg.Ref {
	if f == nil {
		return 0
	}
	span := outputSpan{uint32(f.fset.PositionFor(node.Pos(), false).Offset), uint32(f.fset.PositionFor(node.End(), false).Offset)}
	return f.nodes[span]
}
func (f *errorUseFacts) known(call *ast.CallExpr) bool {
	return f != nil && f.query.Reachable(f.node(call))
}
func (f *errorUseFacts) valueUsed(call *ast.CallExpr) errorflow.Result {
	if f == nil {
		return errorflow.Result{}
	}
	return f.query.Value(f.node(call))
}
func (f *errorUseFacts) assignmentUsed(owner ast.Node, position pdg.Position) errorflow.Result {
	if f == nil {
		return errorflow.Result{}
	}
	return f.query.Assignment(f.node(owner), position)
}
