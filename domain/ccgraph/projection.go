package ccgraph

import (
	"fmt"
	"math/big"
)

// ProjectionInput describes validated operation reachability and boundary roles.
// It cannot establish those facts from PDG producer claims alone.
type Operation struct {
	ID, Category   string
	Administrative bool
	Reachable      *bool
	ReadSymbols    []string
}
type ProjectionInput struct {
	Nodes []Operation
	Edges []Edge
}
type Projection struct {
	Graph             Graph
	Retained, Removed []string
	DroppedEdges      []Edge
	Features          [9]uint64
	Nodes             []NodeDecision
	Edges             []EdgeDecision
}

type NodeDecision struct {
	ID, Reason string
	Retained   bool
}
type EdgeDecision struct {
	Index    int
	Retained bool
	Reason   string
}

var categoryLabels = map[string]Label{
	"declaration": Decl, "assignment": Assign, "control": Control, "call": State,
	"read": Other, "computation": Other, "formal_input": Other, "return": Other,
	"formal_output": Other, "merge": Other, "other": Other,
}
var labelCoordinates = map[Label]int{Decl: 3, Assign: 4, Control: 5, State: 6, Other: 7}
var familyCoordinates = map[Family]int{Ctrl: 0, Execution: 1, Data: 2}

func projectNode(n Operation) (Label, string, error) {
	if n.Category == "entry" || n.Category == "exit" {
		if !n.Administrative {
			return "", "", fmt.Errorf("boundary %q has no administrative validation", n.ID)
		}
		return "", "administrative", nil
	}
	label, ok := categoryLabels[n.Category]
	if !ok {
		return "", "", fmt.Errorf("unsupported category %q", n.Category)
	}
	if n.Category == "formal_input" {
		return label, "", nil
	}
	return reachableLabel(n, label)
}

func reachableLabel(n Operation, label Label) (Label, string, error) {
	if n.Reachable == nil {
		return "", "", fmt.Errorf("operation %q has unknown reachability", n.ID)
	}
	if !*n.Reachable {
		return "", "unreachable", nil
	}
	return label, "", nil
}

type projectionBuilder struct {
	result  Projection
	all     map[string]int
	kept    map[string]bool
	reads   map[string]bool
	reasons map[string]string
}

func Project(input ProjectionInput) (Projection, error) {
	b := projectionBuilder{all: map[string]int{}, kept: map[string]bool{}, reads: map[string]bool{}}
	for _, n := range input.Nodes {
		if err := b.operation(n); err != nil {
			return Projection{}, err
		}
	}
	b.reasons = projectionReasons(b.result.Nodes)
	for i, e := range input.Edges {
		if err := validateEdge(e, b.all); err != nil {
			return Projection{}, err
		}
		if err := b.edge(e, i); err != nil {
			return Projection{}, err
		}
	}
	b.result.Features[8] = uint64(len(b.reads))
	return b.result, nil
}

func (b *projectionBuilder) operation(n Operation) error {
	if n.ID == "" {
		return fmt.Errorf("empty operation ID")
	}
	if _, ok := b.all[n.ID]; ok {
		return fmt.Errorf("duplicate operation %q", n.ID)
	}
	b.all[n.ID] = len(b.all)
	label, reason, err := projectNode(n)
	if err != nil {
		return err
	}
	if reason != "" {
		b.result.Removed = append(b.result.Removed, n.ID)
		b.result.Nodes = append(b.result.Nodes, NodeDecision{n.ID, reason, false})
		return nil
	}
	return b.retain(n, label)
}

func (b *projectionBuilder) retain(n Operation, label Label) error {
	if n.Category == "read" {
		if err := recordReads(n.ReadSymbols, b.reads); err != nil {
			return err
		}
	}
	b.kept[n.ID] = true
	b.result.Nodes = append(b.result.Nodes, NodeDecision{n.ID, "validated reachable value or formal input", true})
	b.result.Retained = append(b.result.Retained, n.ID)
	b.result.Graph.Nodes = append(b.result.Graph.Nodes, Node{n.ID, label})
	b.result.Features[labelCoordinates[label]]++
	return nil
}

func recordReads(symbols []string, reads map[string]bool) error {
	if len(symbols) == 0 {
		return fmt.Errorf("read operation has unknown bindings")
	}
	for _, symbol := range symbols {
		if symbol == "" {
			return fmt.Errorf("read access has unknown binding")
		}
		reads[symbol] = true
	}
	return nil
}

func (b *projectionBuilder) edge(e Edge, index int) error {
	if b.kept[e.To] && b.reasons[e.From] == "unreachable" {
		return fmt.Errorf("dependency from unreachable %q to retained %q", e.From, e.To)
	}
	if !b.kept[e.From] || !b.kept[e.To] {
		b.result.DroppedEdges = append(b.result.DroppedEdges, e)
		b.result.Edges = append(b.result.Edges, EdgeDecision{index, false, "incident on removed node; no splicing"})
		return nil
	}
	b.result.Edges = append(b.result.Edges, EdgeDecision{index, true, "both endpoints retained"})
	b.result.Graph.Edges = append(b.result.Graph.Edges, e)
	b.result.Features[familyCoordinates[e.Family]]++
	return nil
}

func projectionReasons(nodes []NodeDecision) map[string]string {
	out := make(map[string]string, len(nodes))
	for _, node := range nodes {
		out[node.ID] = node.Reason
	}
	return out
}

func Numerical(a, b [9]uint64) Cosine {
	dot, aa, bb := new(big.Int), new(big.Int), new(big.Int)
	for i, x := range a {
		left, right := new(big.Int).SetUint64(x), new(big.Int).SetUint64(b[i])
		dot.Add(dot, new(big.Int).Mul(left, right))
		aa.Add(aa, new(big.Int).Mul(left, left))
		bb.Add(bb, new(big.Int).Mul(right, right))
	}
	return Cosine{dot, new(big.Int).Mul(aa, bb)}
}
