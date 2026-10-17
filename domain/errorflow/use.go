// Package errorflow queries value uses from canonical definitions and operands.
// A positive result is a reachable may-use, not a proof of handling on all paths.
package errorflow

import "github.com/rstyled/gstyle/domain/pdg"

type Query struct {
	graph       *pdg.Graph
	kept        []bool
	definitions [][]pdg.Ref
	values      [][]pdg.Edge
}

func New(graph *pdg.Graph) (*Query, error) {
	if graph == nil {
		return nil, nil
	}
	if err := pdg.Validate(graph); err != nil {
		return nil, err
	}
	if !bindingEvidence(graph) || pdg.ReachableProjection(graph) == nil {
		return nil, nil
	}
	query := &Query{graph: graph, kept: pdg.ReachableProjection(graph), definitions: make([][]pdg.Ref, len(graph.Definitions)), values: make([][]pdg.Edge, len(graph.Nodes))}
	query.index()
	return query, nil
}

func (q *Query) Value(ref pdg.Ref) Result {
	if q == nil || !q.retained(ref) {
		return Result{}
	}
	return q.search(q.valueUses(ref, nil))
}

func (q *Query) Assignment(owner pdg.Ref, position pdg.Position) Result {
	if q == nil || !q.assignmentSite(owner, position) {
		return Result{}
	}
	definitions := q.copyDefinitions(owner, position)
	if len(definitions) == 0 {
		return Result{}
	}
	return q.search(definitions)
}

func (q *Query) Reachable(ref pdg.Ref) bool { return q != nil && q.retained(ref) }

func (q *Query) retained(ref pdg.Ref) bool {
	return ref != 0 && int(ref) <= len(q.graph.Nodes) && (q.kept == nil || q.kept[ref-1])
}

func (q *Query) index() {
	for _, edge := range q.graph.Edges {
		if !q.retainedData(edge) {
			continue
		}
		if q.graph.Tables.Text(edge.Subkind) == "reaching-definition" {
			q.indexDefinition(edge)
			continue
		}
		if valueOperand(q.graph, edge) {
			q.values[edge.Source-1] = append(q.values[edge.Source-1], edge)
		}
	}
}

func (q *Query) indexDefinition(edge pdg.Edge) {
	if edge.Definition == 0 {
		return
	}
	definition := q.graph.Definitions[edge.Definition-1]
	if definition.Node != edge.Source || definition.Symbol != edge.Symbol {
		return
	}
	q.definitions[edge.Definition-1] = append(q.definitions[edge.Definition-1], edge.Target)
}

func (q *Query) copyDefinitions(owner pdg.Ref, position pdg.Position) []item {
	attrs := q.graph.NodeAttributes(q.graph.Nodes[owner-1])
	var result []item
	for index, write := range attrs.Writes {
		slot := write.Position
		if slot == 0 {
			slot = pdg.Position(index + 1)
		}
		if slot != position {
			continue
		}
		definitions := q.symbolDefinitions(attrs, write.Symbol)
		if write.Symbol == 0 || len(definitions) == 0 {
			definitions = []item{{uncertain: true}}
		}
		result = append(result, definitions...)
	}
	return result
}

func (q *Query) symbolDefinitions(attrs pdg.Attributes, symbol pdg.Ref) []item {
	var result []item
	if q.unknownStorage(symbol) {
		result = append(result, item{uncertain: true})
	}
	for _, ref := range attrs.Defines {
		definition := q.graph.Definitions[ref-1]
		if definition.Symbol == symbol && definition.Location == 0 {
			result = append(result, item{ref: ref, definition: true})
		}
	}
	return result
}

func (q *Query) search(queue []item) Result {
	unknown := false
	seen := make([]bool, len(q.values)+len(q.definitions))
	for len(queue) > 0 {
		current := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if current.uncertain {
			unknown = true
			continue
		}
		index := current.index(len(q.values))
		if seen[index] {
			continue
		}
		seen[index] = true
		if current.definition {
			queue = q.definitionUses(current.ref, queue)
			continue
		}
		if q.meaningful(current.ref) {
			return Result{Used: true, Known: true}
		}
		queue = q.valueUses(current.ref, queue)
	}
	return Result{Known: !unknown}
}

func (q *Query) definitionUses(ref pdg.Ref, queue []item) []item {
	for _, target := range q.definitions[ref-1] {
		next := []item{{ref: target}}
		if q.graph.Nodes[target-1].Category == pdg.Merge {
			attrs := q.graph.NodeAttributes(q.graph.Nodes[target-1])
			next = q.symbolDefinitions(attrs, q.graph.Definitions[ref-1].Symbol)
		}
		queue = append(queue, next...)
	}
	return queue
}

func (q *Query) valueUses(ref pdg.Ref, queue []item) []item {
	if len(q.values[ref-1]) == 0 && q.graph.Nodes[ref-1].Category == pdg.Read {
		return append(queue, item{uncertain: true})
	}
	for _, edge := range q.values[ref-1] {
		target := q.graph.Nodes[edge.Target-1]
		var next []item
		switch target.Category {
		case pdg.Assignment, pdg.Declaration:
			next = q.copyDefinitions(edge.Target, edge.Position)
		default:
			next = []item{{ref: edge.Target, uncertain: !q.transparent(target)}}
		}
		queue = append(queue, next...)
	}
	return queue
}

func (q *Query) transparent(node pdg.Node) bool {
	if node.Category != pdg.Computation {
		return true
	}
	attrs := q.graph.NodeAttributes(node)
	if q.graph.Tables.Text(attrs.Construct) == "type-switch-guard" {
		return true
	}
	switch q.graph.Tables.Text(attrs.Operator.Value) {
	case "parentheses", "conversion", "==", "!=", "type-assert":
		return true
	default:
		return false
	}
}

func (q *Query) meaningful(ref pdg.Ref) bool {
	switch q.graph.Nodes[ref-1].Category {
	case pdg.Call, pdg.Return, pdg.Control:
		return true
	default:
		return false
	}
}

func (q *Query) retainedData(edge pdg.Edge) bool {
	return edge.Kind == pdg.Data && q.retained(edge.Source) && q.retained(edge.Target)
}

func (q *Query) assignmentSite(owner pdg.Ref, position pdg.Position) bool {
	if position == 0 || !q.retained(owner) {
		return false
	}
	category := q.graph.Nodes[owner-1].Category
	return category == pdg.Assignment || category == pdg.Declaration
}

// Indirect readers and external users need memory/call summaries. Direct uses
// can still establish may-use, but their absence is not evidence of discard.
func (q *Query) unknownStorage(symbol pdg.Ref) bool {
	if symbol == 0 {
		return true
	}
	binding := q.graph.Symbols[symbol-1]
	if binding.Role == pdg.GlobalRole {
		return true
	}
	for _, extension := range binding.Extensions {
		if unknownStorageFact(q.graph.Tables.Text(extension.Name), q.graph.Tables.Text(extension.Value)) {
			return true
		}
	}
	return false
}

type Result struct{ Used, Known bool }

type item struct {
	ref        pdg.Ref
	definition bool
	uncertain  bool
}

func (vertex item) index(values int) int {
	if vertex.definition {
		return values + int(vertex.ref) - 1
	}
	return int(vertex.ref) - 1
}

func bindingEvidence(graph *pdg.Graph) bool {
	for _, capability := range graph.Capabilities {
		name := graph.Tables.Text(capability.Name)
		if name != "scalar_reaching_definitions" && name != "interface_binding_flow" {
			continue
		}
		status := graph.Tables.Evidence[capability.Evidence-1].Status
		if status == pdg.Supported || status == pdg.Approximated {
			return true
		}
	}
	return false
}

func valueOperand(graph *pdg.Graph, edge pdg.Edge) bool {
	if edge.Position == 0 {
		return false
	}
	switch graph.Tables.Text(edge.Subkind) {
	case "source-value", "binding-value", "scalar-value", "control-value", "type-switch-value":
		return true
	default:
		return false
	}
}

func unknownStorageFact(name, value string) bool {
	return (name == "gstyle.runtime:unstable" || name == "gstyle.memory:captured") && value == "true"
}
