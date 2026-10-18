// Package pdgmemory computes conservative storage aliases from canonical PDG
// operations. It never reads Go syntax, compiler objects or variable spellings.
package pdgmemory

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

// The zero Place is unknown externally reachable storage. Other places identify
// a variable's storage or an allocation site. Allocation sites may hold many
// concrete objects; they are not singleton heap cells.
type Place struct{ Symbol, Allocation pdg.Ref }

type Attributes func(pdg.Node) pdg.Attributes

type Model struct {
	graph            *pdg.Graph
	attrs            []pdg.Attributes
	operands         []map[pdg.Position]pdg.Ref
	addresses        [][]pdg.Ref
	values, bindings [][]Place
	addressed        []bool
	addressOnly      []bool
	places           []Place
}

func (m *Model) extension(values []pdg.Extension, name string) string {
	for _, value := range values {
		if m.graph.Tables.Text(value.Name) == name {
			return m.graph.Tables.Text(value.Value)
		}
	}
	return ""
}

func (m *Model) reference(ref pdg.Ref) bool {
	return ref != 0 && m.extension(m.attrs[ref-1].Extensions, "gstyle.memory:reference") == "true"
}

func (m *Model) symbolReference(ref pdg.Ref) bool {
	return m.extension(m.graph.Symbols[ref-1].Extensions, "gstyle.memory:reference") == "true"
}

func (m *Model) operator(ref pdg.Ref) string {
	if ref == 0 || m.attrs[ref-1].Operator.Status != pdg.Known {
		return ""
	}
	return m.graph.Tables.Text(m.attrs[ref-1].Operator.Value)
}

func (m *Model) operand(ref pdg.Ref, position pdg.Position) pdg.Ref {
	if ref == 0 {
		return 0
	}
	return m.operands[ref-1][position]
}

func (m *Model) nodeValues(ref pdg.Ref) []Place {
	if ref == 0 {
		return nil
	}
	return m.values[ref-1]
}

func (m *Model) index() error {
	for _, edge := range m.graph.Edges {
		if err := m.indexEdge(edge); err != nil {
			return err
		}
	}

	return m.validateBindings()
}

func (m *Model) validateBindings() error {
	for _, attrs := range m.attrs {
		if err := m.validateCaptureSymbols(attrs); err != nil {
			return err
		}
		for _, access := range slices.Concat(attrs.Reads, attrs.Writes) {
			if int(access.Symbol) > len(m.bindings) {
				return fmt.Errorf("invalid memory binding reference")
			}
		}
	}
	return nil
}

func (m *Model) indexEdge(edge pdg.Edge) error {
	if !memoryEdge(m.graph, edge) {
		return nil
	}
	if !m.validNode(edge.Source) || !m.validNode(edge.Target) {
		return fmt.Errorf("invalid memory operand edge")
	}
	if m.graph.Tables.Text(edge.Subkind) == "write-address" {
		m.addresses[edge.Target-1] = append(m.addresses[edge.Target-1], edge.Source)
		return nil
	}
	if m.operands[edge.Target-1] == nil {
		m.operands[edge.Target-1] = map[pdg.Position]pdg.Ref{}
	}
	m.operands[edge.Target-1][edge.Position] = edge.Source
	return nil
}

func (m *Model) validNode(ref pdg.Ref) bool { return ref != 0 && int(ref) <= len(m.graph.Nodes) }

func (m *Model) validateCaptureSymbols(attrs pdg.Attributes) error {
	for _, symbol := range attrs.Symbols {
		if symbol == 0 || int(symbol) > len(m.bindings) {
			return fmt.Errorf("invalid capture symbol reference")
		}
	}
	return nil
}

func Analyze(graph *pdg.Graph, attributes Attributes) (*Model, error) {
	if !validInput(graph, attributes) {
		return nil, fmt.Errorf("memory analysis requires PDG operations and attributes")
	}
	m := &Model{graph: graph, attrs: make([]pdg.Attributes, len(graph.Nodes)), operands: make([]map[pdg.Position]pdg.Ref, len(graph.Nodes)), addresses: make([][]pdg.Ref, len(graph.Nodes)), values: make([][]Place, len(graph.Nodes)), bindings: make([][]Place, len(graph.Symbols)), addressed: make([]bool, len(graph.Symbols)), addressOnly: make([]bool, len(graph.Nodes))}
	for i, node := range graph.Nodes {
		m.attrs[i] = attributes(node)
	}
	if err := m.index(); err != nil {
		return nil, err
	}
	m.seed()
	for m.update() {
	}
	m.collectPlaces()
	return m, nil
}

func union(left, right []Place) ([]Place, bool) {
	joined := append(slices.Clone(left), right...)
	slices.SortFunc(joined, func(a, b Place) int {
		if order := cmp.Compare(a.Symbol, b.Symbol); order != 0 {
			return order
		}
		return cmp.Compare(a.Allocation, b.Allocation)
	})
	joined = slices.Compact(joined)
	return joined, !slices.Equal(left, joined)
}

func memoryEdge(graph *pdg.Graph, edge pdg.Edge) bool {
	if edge.Kind != pdg.Data || edge.Position == 0 {
		return false
	}
	switch graph.Tables.Text(edge.Subkind) {
	case "source-value", "scalar-value", "binding-value", "control-value", "write-address":
		return true
	default:
		return false
	}
}

func validInput(graph *pdg.Graph, attributes Attributes) bool {
	return graph != nil && graph.Tables != nil && attributes != nil
}
