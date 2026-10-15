// Package interfacenil proves non-nil interface comparisons from the shared
// PDG. It never consumes ASTs, mining-value edges, or clone matching labels.
package interfacenil

import "github.com/rstyled/gstyle/domain/pdg"

type site struct {
	graph int
	node  pdg.Ref
}

type proofNode struct {
	dependencies []int
	users        []int
	pending      int
	seed         bool
}

// The worklist is private scratch storage. Mutating counters and adjacency
// lists keeps proof linear in graph size; no per-node ancestry sets are copied.
type solver struct {
	nodes   []proofNode
	sites   []site
	offsets []int
	outputs map[string]int
}

func newSolver(graphs []pdg.Graph) *solver {
	s := &solver{outputs: map[string]int{}}
	for i, graph := range graphs {
		if !supported(&graph) {
			s.offsets = append(s.offsets, -1)
			continue
		}
		s.offsets = append(s.offsets, len(s.nodes))
		for ref, node := range graph.Nodes {
			s.nodes = append(s.nodes, proofNode{})
			s.sites = append(s.sites, site{i, pdg.Ref(ref + 1)})
			if node.Category == pdg.FormalOutput && len(graph.Function.Outputs) == 1 {
				s.outputs[graph.Tables.Text(graph.Function.ID)] = s.offsets[i] + ref
			}
		}
	}
	return s
}

func (s *solver) function(graph pdg.Graph, index int) {
	if !supported(&graph) {
		return
	}
	offset := s.offsets[index]
	eligible := make([]bool, len(graph.Nodes))
	for ref, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		target := offset + ref
		eligible[ref] = s.operation(&graph, node, attrs, target)
	}

	for _, edge := range graph.Edges {
		if !eligible[edge.Target-1] || !proofEdge(&graph, edge) {
			continue
		}
		target := offset + int(edge.Target) - 1
		s.nodes[target].dependencies = append(s.nodes[target].dependencies, offset+int(edge.Source)-1)
	}
}

func (s *solver) operation(graph *pdg.Graph, node pdg.Node, attrs pdg.Attributes, target int) bool {
	kind := extension(graph, attrs, "gstyle:interface-value")
	s.nodes[target].seed = kind == `"concrete"` || kind == `"boxed"`
	if node.Category == pdg.Call {
		s.call(graph, attrs, target)
		return false
	}
	return transfer(node.Category, attrs, graph)
}

func (s *solver) call(graph *pdg.Graph, attrs pdg.Attributes, target int) {
	if attrs.CallForm != pdg.DirectCall || attrs.Callee.Status != pdg.Known {
		return
	}
	output, ok := s.outputs[graph.Tables.Text(attrs.Callee.Value)]
	if ok {
		s.nodes[target].dependencies = append(s.nodes[target].dependencies, output)
	}
}

func (s *solver) solve() []bool {
	known := make([]bool, len(s.nodes))
	queue := s.seedQueue(known)

	for cursor := 0; cursor < len(queue); cursor++ {
		for _, user := range s.nodes[queue[cursor]].users {
			s.nodes[user].pending--
			if known[user] || s.nodes[user].pending != 0 {
				continue
			}
			known[user] = true
			queue = append(queue, user)
		}
	}
	return known
}

func (s *solver) seedQueue(known []bool) []int {
	var queue []int
	for index := range s.nodes {
		node := &s.nodes[index]
		node.pending = len(node.dependencies)
		for _, dependency := range node.dependencies {
			s.nodes[dependency].users = append(s.nodes[dependency].users, index)
		}
		if node.seed {
			known[index] = true
			queue = append(queue, index)
		}
	}
	return queue
}

func (s *solver) operandKnown(graphs []pdg.Graph, index int, position pdg.Position, known []bool) bool {
	where := s.sites[index]
	graph := &graphs[where.graph]
	offset := s.offsets[where.graph]
	found := false
	for _, edge := range graph.Edges {
		if !comparisonOperand(graph, edge, where.node, position) {
			continue
		}
		if !known[offset+int(edge.Source)-1] {
			return false
		}
		found = true
	}
	return found
}

// Find returns comparison nodes with a proven non-nil interface operand.
// Unknown inputs and unresolved/recursive call summaries do not become proof.
func Find(graphs []pdg.Graph) [][]pdg.Ref {
	s := newSolver(graphs)
	for i := range graphs {
		s.function(graphs[i], i)
	}
	known := s.solve()
	results := make([][]pdg.Ref, len(graphs))
	for index, where := range s.sites {
		graph := &graphs[where.graph]
		attrs := graph.NodeAttributes(graph.Nodes[where.node-1])
		position := nilOperand(graph, attrs)
		if position == 0 {
			continue
		}
		if s.operandKnown(graphs, index, position, known) {
			results[where.graph] = append(results[where.graph], where.node)
		}
	}
	return results
}

func supported(graph *pdg.Graph) bool {
	for _, capability := range graph.Capabilities {
		if graph.Tables.Text(capability.Name) == "interface_binding_flow" {
			return graph.Tables.Evidence[capability.Evidence-1].Status == pdg.Supported
		}
	}
	return false
}

func transfer(category pdg.Category, attrs pdg.Attributes, graph *pdg.Graph) bool {
	switch category {
	case pdg.Read:
		return localRead(attrs)
	case pdg.Assignment, pdg.Declaration, pdg.Return, pdg.FormalOutput, pdg.Merge:
		return true
	case pdg.Computation:
		if attrs.Operator.Status != pdg.Known {
			return false
		}
		operator := graph.Tables.Text(attrs.Operator.Value)
		return operator == "conversion" || operator == "parentheses"
	default:
		return false
	}
}

func localRead(attrs pdg.Attributes) bool {
	if len(attrs.Reads) != 1 {
		return false
	}
	read := attrs.Reads[0]
	return read.Symbol != 0 && read.Location == 0 && read.UnresolvedReason == 0
}

func proofEdge(graph *pdg.Graph, edge pdg.Edge) bool {
	if edge.Kind != pdg.Data {
		return false
	}
	switch graph.Tables.Text(edge.Subkind) {
	case "reaching-definition", "binding-value", "return-value":
		return graph.Tables.Evidence[edge.Evidence-1].Status == pdg.Supported
	default:
		return false
	}
}

func extension(graph *pdg.Graph, attrs pdg.Attributes, name string) string {
	for _, value := range attrs.Extensions {
		if graph.Tables.Text(value.Name) == name {
			return graph.Tables.Text(value.Value)
		}
	}
	return ""
}

func nilOperand(graph *pdg.Graph, attrs pdg.Attributes) pdg.Position {
	switch extension(graph, attrs, "gstyle:interface-nil-operand") {
	case "1":
		return 1
	case "2":
		return 2
	default:
		return 0
	}
}

func comparisonOperand(graph *pdg.Graph, edge pdg.Edge, target pdg.Ref, position pdg.Position) bool {
	if edge.Target != target || edge.Position != position {
		return false
	}
	return graph.Tables.Text(edge.Subkind) == "binding-value" && proofEdge(graph, edge)
}
