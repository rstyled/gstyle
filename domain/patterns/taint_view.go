package patterns

import "github.com/rstyled/gstyle/domain/pdg"

// Taint follows argument values. Opaque memory effects and saved-call identity
// are dependencies for other analyses, not evidence that an argument contains
// the result of an earlier call. Keep the shared semantic view unchanged.
func taintView(f *MiningFacts) *MiningGraph {
	if f.Canonical == nil {
		return f.Pdg
	}
	view := &MiningGraph{Nodes: f.Pdg.Nodes}
	for _, edge := range f.Canonical.Edges {
		if !taintValueEdge(f.Canonical, edge) {
			continue
		}
		from, to := int(edge.Source)-1, int(edge.Target)-1
		if f.Pdg.Nodes[from].Kind == "unreachable" || f.Pdg.Nodes[to].Kind == "unreachable" {
			continue
		}
		view.Edges = append(view.Edges, semanticEdge(edge))
	}
	return view
}

func taintValueEdge(graph *pdg.Graph, edge pdg.Edge) bool {
	if edge.Kind != pdg.Data {
		return false
	}
	target := graph.Nodes[edge.Target-1]
	if target.Category == pdg.Call && edge.Position == 0 {
		return false
	}
	// Calls can weakly redefine escaped bindings. Their return value still flows
	// through its explicit result operand, not through the memory definition.
	if opaqueCallDefinition(graph, edge) {
		return false
	}
	switch graph.Tables.Text(edge.Subkind) {
	case "address", "write-address", "capture-address", "call-target", "defer-registration":
		return false
	default:
		return true
	}
}

func opaqueCallDefinition(graph *pdg.Graph, edge pdg.Edge) bool {
	source := graph.Nodes[edge.Source-1]
	return source.Category == pdg.Call && edge.Definition != 0 && len(graph.NodeAttributes(source).Writes) != 0
}
