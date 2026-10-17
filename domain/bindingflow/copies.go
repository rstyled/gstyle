// Package bindingflow selects binding transfers from canonical PDG operands.
// It does not infer paths, aliases or call results. Value shapes need compiler facts.
package bindingflow

import (
	"cmp"
	"github.com/rstyled/gstyle/domain/pdg"
	"slices"
)

type operand struct {
	ref      pdg.Ref
	conflict bool
}

type Query struct {
	callShapes      map[pdg.Ref]CallShape
	tupleArities    map[pdg.Ref]uint32
	representations map[pdg.Ref]map[pdg.Position]Representation
	reaching        [][]pdg.Edge
	graph           *pdg.Graph
	operands        []map[pdg.Position]operand
	weak            map[pdg.Ref]bool
	fields          map[pdg.Ref][]uint32
	required        map[pdg.Ref][]pdg.Position
	reached         []bool
	allocations     map[pdg.Ref]pdg.Ref
}

func New(graph *pdg.Graph) (*Query, error) {
	if graph == nil {
		return nil, nil
	}
	if err := pdg.Validate(graph); err != nil {
		return nil, err
	}
	q := &Query{callShapes: map[pdg.Ref]CallShape{}, tupleArities: map[pdg.Ref]uint32{}, representations: map[pdg.Ref]map[pdg.Position]Representation{}, reaching: reachingDefinitions(graph), graph: graph, operands: make([]map[pdg.Position]operand, len(graph.Nodes)), weak: map[pdg.Ref]bool{}, fields: map[pdg.Ref][]uint32{}, required: map[pdg.Ref][]pdg.Position{}, reached: pdg.ReachableProjection(graph), allocations: map[pdg.Ref]pdg.Ref{}}
	if err := q.indexMetadata(); err != nil {
		return nil, err
	}
	for _, edge := range graph.Edges {
		q.index(edge)
	}
	return q, nil
}

// Writes returns fresh records in slot order; unordered writes follow them.
// Missing/conflicting operands, compound writes,
// iteration values and opaque effects carry no known source.
func (q *Query) Writes(ref pdg.Ref) []Write {
	if q == nil || ref == 0 || int(ref) > len(q.graph.Nodes) {
		return nil
	}
	node := q.graph.Nodes[ref-1]
	attrs := q.graph.NodeAttributes(node)
	var writes []Write
	for _, access := range attrs.Writes {
		writes = append(writes, q.writeTransfer(ref, attrs, access))
	}
	slices.SortFunc(writes, writeOrder)
	return writes
}

// Operand selects one canonical value producer. Duplicate identical edges are
// harmless; conflicting producers and missing slots remain unknown.
func (q *Query) Operand(owner pdg.Ref, position pdg.Position) pdg.Ref {
	if q == nil || owner == 0 || int(owner) > len(q.operands) || position == 0 {
		return 0
	}
	source := q.operands[owner-1][position]
	if source.conflict {
		return 0
	}
	return source.ref
}

// ReadSymbol selects an explicit direct read at a result/operand slot.
func (q *Query) ReadSymbol(owner pdg.Ref, position pdg.Position) pdg.Ref {
	if q == nil || owner == 0 || int(owner) > len(q.graph.Nodes) || position == 0 {
		return 0
	}
	attrs := q.graph.NodeAttributes(q.graph.Nodes[owner-1])
	return directRead(attrs.Reads, position)
}

// RequiredOperands returns the compiler's value-slot inventory. A missing
// inventory remains distinct from a known empty one. The returned slice is owned.
func (q *Query) RequiredOperands(ref pdg.Ref) ([]pdg.Position, bool) {
	if q == nil {
		return nil, false
	}
	positions, found := q.required[ref]
	return append([]pdg.Position{}, positions...), found
}

// InputValues returns the complete evaluated value inputs in compiler slot order.
// Missing, extra or conflicting operands cannot certify an evaluation inventory.
func (q *Query) InputValues(owner pdg.Ref) ([]pdg.Ref, bool) {
	positions, known := q.RequiredOperands(owner)
	if !known || !q.validRef(owner) {
		return nil, false
	}
	if len(positions) != len(q.operands[owner-1]) {
		return nil, false
	}
	values := make([]pdg.Ref, len(positions))
	for i, position := range positions {
		ref := q.Operand(owner, position)
		if ref == 0 {
			return nil, false
		}
		values[i] = ref
	}
	return values, true
}

func (q *Query) index(edge pdg.Edge) {
	if !copyOperand(q.graph, edge) {
		return
	}
	if q.operands[edge.Target-1] == nil {
		q.operands[edge.Target-1] = map[pdg.Position]operand{}
	}
	previous := q.operands[edge.Target-1][edge.Position]
	if previous.ref != 0 && previous.ref != edge.Source {
		previous.conflict = true
	}
	previous.ref = edge.Source
	q.operands[edge.Target-1][edge.Position] = previous
}

func (q *Query) copySource(ref pdg.Ref, attrs pdg.Attributes, write *Write) {
	source := q.operands[ref-1][write.Position]
	if q.zeroInitialized(attrs) {
		write.Zero = source.ref == 0 && !source.conflict
		return
	}
	if source.conflict {
		return
	}

	write.Source = source.ref
}

func (q *Query) writeTransfer(ref pdg.Ref, attrs pdg.Attributes, access pdg.Access) Write {
	write := Write{Symbol: access.Symbol, Location: access.Location, Position: access.Position, Unresolved: access.UnresolvedReason != 0}
	if !copyOperation(q.graph, q.graph.Nodes[ref-1], attrs) || !directBinding(access) || !q.strongWrite(attrs, access.Symbol) {
		return write
	}
	q.copySource(ref, attrs, &write)
	return write
}

func (q *Query) zeroInitialized(attrs pdg.Attributes) bool {
	for _, extension := range attrs.Extensions {
		if q.graph.Tables.Text(extension.Name) == "gstyle.runtime:zero-initialized" && q.graph.Tables.Text(extension.Value) == "true" {
			return true
		}
	}
	return false
}

type Write struct {
	Symbol, Location, Source pdg.Ref
	Position                 pdg.Position
	Zero                     bool
	Unresolved               bool
}

// UnknownTarget cannot identify one binding or storage location.
func (write Write) UnknownTarget() bool {
	return write.Symbol == 0 && (write.Location == 0 || write.Unresolved)
}

func valueKind(kind string) bool {
	switch kind {
	case "source-value", "binding-value", "scalar-value":
		return true
	default:
		return false
	}
}

func copyOperation(graph *pdg.Graph, node pdg.Node, attrs pdg.Attributes) bool {
	if node.Category != pdg.Assignment && node.Category != pdg.Declaration {
		return false
	}
	if graph.Tables.Text(attrs.Construct) != "" {
		return false
	}
	switch graph.Tables.Text(attrs.WriteForm) {
	case "", "=", ":=":
		return true
	default:
		return false
	}
}

func copyOperand(graph *pdg.Graph, edge pdg.Edge) bool {
	return edge.Kind == pdg.Data && edge.Position != 0 && valueKind(graph.Tables.Text(edge.Subkind))
}

func directRead(reads []pdg.Access, position pdg.Position) pdg.Ref {
	var symbol pdg.Ref
	for _, read := range reads {
		if read.Position != position {
			continue
		}
		if !directBinding(read) {
			return 0
		}
		if symbol != 0 && symbol != read.Symbol {
			return 0
		}
		symbol = read.Symbol
	}
	return symbol
}

func directBinding(access pdg.Access) bool {
	return access.Symbol != 0 && access.Location == 0 && access.UnresolvedReason == 0
}

// reachingDefinitions builds the definition relation separately from value
// operands. A reaching edge can carry a binding slot without being an operand.
func reachingDefinitions(graph *pdg.Graph) [][]pdg.Edge {
	reaching := make([][]pdg.Edge, len(graph.Nodes))
	for _, edge := range graph.Edges {
		if edge.Kind != pdg.Data || graph.Tables.Text(edge.Subkind) != "reaching-definition" {
			continue
		}
		reaching[edge.Target-1] = append(reaching[edge.Target-1], edge)
	}
	return reaching
}

func writeOrder(a, b Write) int {
	if a.Position == 0 || b.Position == 0 {
		return cmp.Compare(b.Position, a.Position)
	}
	return cmp.Compare(a.Position, b.Position)
}
