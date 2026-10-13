package patterns

import "github.com/rstyled/gstyle/domain/pdg"

// semanticFact borrows the canonical graph and builds scratch for one function.
// It does not retain a second full-corpus graph or copy symbols and evidence.
func semanticFact(f *MiningFacts) *MiningFacts {
	if f == nil || f.Canonical == nil {
		return f
	}
	out := *f
	out.Pdg = semanticView(f.Canonical)
	return &out
}

// Node indexes correspond exactly to canonical records. Unreachable operation
// slots remain as nonsemantic placeholders; incident edges are omitted. Callers
// can resolve binding, definition, memory, and evidence facts through Canonical.
func semanticView(graph *pdg.Graph) *MiningGraph {
	kept := pdg.NormalProjection(graph)
	view := &MiningGraph{Nodes: make([]PdgNode, len(graph.Nodes)), Edges: make([]PdgEdge, 0, len(graph.Edges))}
	for i, node := range graph.Nodes {
		if normalNodeRemoved(kept, i) {
			view.Nodes[i] = PdgNode{Kind: "unreachable", Line: int(node.Line)}
			continue
		}
		view.Nodes[i] = semanticNode(graph, node)
	}
	for _, edge := range graph.Edges {
		if normalEdgeRemoved(kept, edge) {
			continue
		}
		view.Edges = append(view.Edges, semanticEdge(edge))
	}
	return view
}

func semanticNode(graph *pdg.Graph, node pdg.Node) PdgNode {
	result := PdgNode{Kind: semanticKinds[node.Category], Line: int(node.Line)}
	if node.MatchLabel != 0 {
		result = miningNode(graph, node)
	}
	result.Kind = canonicalSemanticKind(node.Category, result.Kind)
	if node.Category == pdg.Call {
		return semanticCallNode(graph, node, result)
	}
	return result
}

func semanticCallee(graph *pdg.Graph, attrs pdg.Attributes) string {
	if attrs.Callee.Status != pdg.Known {
		return ""
	}
	return graph.Tables.Text(attrs.Callee.Value)
}

var semanticKinds = map[pdg.Category]NodeKind{
	pdg.Entry: "entry", pdg.Exit: "exit", pdg.FormalInput: Param,
	pdg.FormalOutput: "formal_output", pdg.Declaration: Let, pdg.Assignment: Op,
	pdg.Computation: Op, pdg.Read: Field, pdg.Call: Call, pdg.Control: Branch,
	pdg.Return: Return, pdg.Other: Op,
}

func semanticEdge(edge pdg.Edge) PdgEdge {
	kind := Data
	switch edge.Kind {
	case pdg.ControlEdge:
		kind = Ctrl
	case pdg.Execution:
		kind = Exec
	}
	return PdgEdge{From: int(edge.Source) - 1, To: int(edge.Target) - 1, Kind: kind, ArgPos: int(edge.Position) - 1}
}

func canonicalNodeID(graph *pdg.Graph, index int) string {
	if graph == nil || index < 0 || index >= len(graph.Nodes) {
		return ""
	}
	return graph.Tables.Text(graph.Nodes[index].ID)
}

func semanticCallNode(graph *pdg.Graph, node pdg.Node, result PdgNode) PdgNode {
	result.Kind = Call
	if node.Attributes != 0 {
		result.CalleeID = semanticCallee(graph, graph.NodeAttributes(node))
	}
	return result
}

func canonicalSemanticKind(category pdg.Category, fallback NodeKind) NodeKind {
	switch category {
	case pdg.FormalInput, pdg.FormalOutput, pdg.Return, pdg.Call:
		return semanticKinds[category]
	default:
		return fallback
	}
}
