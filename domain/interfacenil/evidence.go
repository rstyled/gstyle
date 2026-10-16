package interfacenil

import "github.com/rstyled/gstyle/domain/pdg"

// ProofSite identifies a node in the original input graph array.
type ProofSite struct {
	Graph int     `json:"graph"`
	Node  pdg.Ref `json:"node"`
}

// ProofStep records the dependencies consumed by the non-nil solver. A seed
// establishes non-nil directly, so its syntax operands are not proof inputs.
type ProofStep struct {
	Site         ProofSite   `json:"site"`
	Seed         bool        `json:"seed"`
	Dependencies []ProofSite `json:"dependencies"`
}

// Proof explains one reported comparison, without claiming complete program
// behavior. It uses the same eligibility, dependencies, and solution as Find.
type Proof struct {
	Comparison ProofSite    `json:"comparison"`
	Operand    pdg.Position `json:"operand"`
	Roots      []ProofSite  `json:"roots"`
	Steps      []ProofStep  `json:"steps"`
}

// Explain returns the successful solver dependencies behind each finding.
// The caller must supply valid native PDGs, as for Find.
func Explain(graphs []pdg.Graph) []Proof {
	s := newSolver(graphs)
	for i := range graphs {
		s.function(graphs[i], i)
	}
	known := s.solve()
	var out []Proof
	for index := range s.sites {
		if proof, ok := s.explanation(graphs, index, known); ok {
			out = append(out, proof)
		}
	}
	return out
}

func (s *solver) explanation(graphs []pdg.Graph, index int, known []bool) (Proof, bool) {
	where := s.sites[index]
	graph := &graphs[where.graph]
	attrs := graph.NodeAttributes(graph.Nodes[where.node-1])
	position := nilOperand(graph, attrs)
	if position == 0 {
		return Proof{}, false
	}
	if !s.operandKnown(graphs, index, position, known) {
		return Proof{}, false
	}
	roots := comparisonRoots(graph, where.graph, where.node, position)
	return Proof{Comparison: publicSite(where), Operand: position, Roots: roots, Steps: s.proofSteps(roots)}, true
}

func comparisonRoots(graph *pdg.Graph, index int, target pdg.Ref, position pdg.Position) []ProofSite {
	var roots []ProofSite
	for _, edge := range graph.Edges {
		if comparisonOperand(graph, edge, target, position) {
			roots = append(roots, ProofSite{Graph: index, Node: edge.Source})
		}
	}
	return roots
}

func publicSite(where site) ProofSite { return ProofSite{Graph: where.graph, Node: where.node} }

func (s *solver) proofSteps(roots []ProofSite) []ProofStep {
	queue := append([]ProofSite(nil), roots...)
	seen := make(map[ProofSite]bool)
	var out []ProofStep
	for cursor := 0; cursor < len(queue); cursor++ {
		where := queue[cursor]
		if seen[where] {
			continue
		}
		seen[where] = true
		step := s.proofStep(where)
		out = append(out, step)
		queue = append(queue, step.Dependencies...)
	}
	return out
}

func (s *solver) proofStep(where ProofSite) ProofStep {
	node := s.nodes[s.offsets[where.Graph]+int(where.Node)-1]
	step := ProofStep{Site: where, Seed: node.seed, Dependencies: []ProofSite{}}
	if node.seed {
		return step
	}
	for _, dependency := range node.dependencies {
		step.Dependencies = append(step.Dependencies, publicSite(s.sites[dependency]))
	}
	return step
}
