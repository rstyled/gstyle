package changerisk

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
)

type operation struct {
	Category                  pdg.Category
	Synthetic                 bool
	Operator, Literal, Callee string
	Construct, WriteForm      string
	CallForm                  pdg.CallForm
	ExitKind                  pdg.ExitKind
	Position                  pdg.Position
	Symbols, Reads, Writes    []string
}

func Project(g *pdg.Graph) (Graph, error) {
	if err := pdg.Validate(g); err != nil {
		return Graph{}, err
	}
	operations, err := projectNodes(g)
	if err != nil {
		return Graph{}, err
	}
	nodes := numberedNodes(operations)
	edges, shape := projectEdges(g, nodeKeys(nodes), nodeShapes(operations))
	slices.Sort(shape)
	encoded, encodeErr := json.Marshal(shape)
	if encodeErr != nil {
		return Graph{}, encodeErr
	}
	return Graph{Nodes: nodes, Edges: edges, Shape: digest(encoded), Ambiguous: duplicateNodes(operations)}, nil
}

func projectNodes(g *pdg.Graph) ([]Node, error) {
	var nodes []Node
	for _, source := range g.Nodes {
		node, err := projectNode(g, source)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func numberedNodes(nodes []Node) []Node {
	counts := map[string]int{}
	var result []Node
	for _, node := range nodes {
		counts[node.Key]++
		node.Key += ":" + strconv.Itoa(counts[node.Key])
		result = append(result, node)
	}
	return result
}

func nodeKeys(nodes []Node) []string {
	var keys []string
	for _, node := range nodes {
		keys = append(keys, node.Key)
	}
	return keys
}

func nodeShapes(nodes []Node) []string {
	var shapes []string
	for _, node := range nodes {
		shapes = append(shapes, "node:"+node.Key)
	}
	return shapes
}

func duplicateNodes(nodes []Node) bool {
	seen := map[string]bool{}
	for _, node := range nodes {
		if seen[node.Key] {
			return true
		}
		seen[node.Key] = true
	}
	return false
}

func digest(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func projectNode(g *pdg.Graph, node pdg.Node) (Node, error) {
	a := g.NodeAttributes(node)
	op := operation{Category: node.Category, Synthetic: node.Synthetic,
		Operator: fact(g, a.Operator), Literal: fact(g, a.Literal), Callee: identityFact(g, a.Callee),
		Construct: g.Tables.Text(a.Construct), WriteForm: g.Tables.Text(a.WriteForm), CallForm: a.CallForm, ExitKind: a.ExitKind, Position: a.Position,
		Symbols: symbols(g, a.Symbols), Reads: accesses(g, a.Reads), Writes: accesses(g, a.Writes)}
	data, err := json.Marshal(op)
	if err != nil {
		return Node{}, err
	}
	item := Node{Key: digest(data), Label: nodeLabel(g, node, a), Line: int(node.Line)}
	if node.Category == pdg.Call && a.CallForm == pdg.DirectCall && a.Callee.Status == pdg.Known {
		item.Callee = g.Tables.Text(a.Callee.Value)
	}
	item.UnresolvedCall = node.Category == pdg.Call && item.Callee == ""
	return item, nil
}

func fact(g *pdg.Graph, f pdg.Fact) string {
	return strconv.Itoa(int(f.Status)) + ":" + g.Tables.Text(f.Value)
}

func identityFact(g *pdg.Graph, f pdg.Fact) string {
	parts := strings.Split(g.Tables.Text(f.Value), ".closure@")
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.TrimLeft(parts[i], "0123456789")
	}
	return strconv.Itoa(int(f.Status)) + ":" + strings.Join(parts, ".closure")
}

func symbols(g *pdg.Graph, refs []pdg.Ref) []string {
	var result []string
	for _, ref := range refs {
		result = append(result, symbol(g, ref))
	}
	return result
}

func symbol(g *pdg.Graph, ref pdg.Ref) string {
	if ref == 0 {
		return ""
	}
	s := g.Symbols[ref-1]
	return fmt.Sprintf("%s|%s|%d", identityFact(g, s.Name), fact(g, s.Type), s.Role)
}

func accesses(g *pdg.Graph, values []pdg.Access) []string {
	var result []string
	for _, access := range values {
		result = append(result, fmt.Sprintf("%s|%s|%d", symbol(g, access.Symbol), g.Tables.Text(access.UnresolvedReason), access.Position))
	}
	return result
}

func nodeLabel(g *pdg.Graph, node pdg.Node, a pdg.Attributes) string {
	if a.Callee.Status == pdg.Known {
		return g.Tables.Text(a.Callee.Value)
	}
	if label := sourceLabel(g, node); label != "" {
		return label
	}
	if a.Operator.Status == pdg.Known {
		return g.Tables.Text(a.Operator.Value)
	}
	if a.Literal.Status == pdg.Known {
		return g.Tables.Text(a.Literal.Value)
	}
	if a.Construct != 0 {
		return g.Tables.Text(a.Construct)
	}
	return []string{"declaration", "read", "assignment", "computation", "control", "call", "entry", "input", "return", "output", "exit", "merge", "memory", "other"}[node.Category]
}

func sourceLabel(g *pdg.Graph, node pdg.Node) string {
	if g.Tables.Sources == nil || node.Span == 0 {
		return ""
	}
	span := g.Spans[node.Span-1]
	if span.Source == 0 {
		return ""
	}
	_, content, ok := g.Tables.Sources.Source(g.Sources[span.Source-1].Snapshot)
	if !ok || uint64(span.End) > uint64(len(content)) {
		return ""
	}
	return strings.Join(strings.Fields(content[span.Start:span.End]), " ")
}

func projectEdges(g *pdg.Graph, keys []string, shape []string) ([]Edge, []string) {
	var result []Edge
	for _, edge := range g.Edges {
		label := edgeLabel(edge.Kind) + ":" + g.Tables.Text(edge.Subkind) + ":" + g.Tables.Text(edge.Outcome)
		label += fmt.Sprintf(":%d:%d", edge.Position, edge.LoopCarried)
		from, to := keys[edge.Source-1], keys[edge.Target-1]
		key := from + "|" + to + "|" + label
		result = append(result, Edge{Key: key, From: from, To: to, Label: edgeLabel(edge.Kind)})
		shape = append(shape, "edge:"+from[:64]+"|"+to[:64]+"|"+label)
	}
	return result, shape
}

func edgeLabel(kind pdg.EdgeKind) string {
	return []string{"data", "control", "execution"}[kind]
}
