package runtimeguard

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
	"strconv"
)

type graphView struct {
	graph        *pdg.Graph
	operands     [][2]pdg.Ref
	unstable     []bool
	kinds        []string
	tracked      []bool
	fields       []pdg.Ref
	paths        map[string]pdg.Ref
	fieldRoots   map[pdg.Ref]pdg.Ref
	dereferences map[pdg.Ref][]pdg.Ref
}

func newGraphView(g *pdg.Graph) (*graphView, error) {
	if err := validateFlow(g); err != nil {
		return nil, err
	}
	view := &graphView{graph: g, operands: make([][2]pdg.Ref, len(g.Nodes)), unstable: make([]bool, len(g.Symbols)), kinds: make([]string, len(g.Symbols)), tracked: make([]bool, len(g.Symbols))}
	for i, symbol := range g.Symbols {
		view.kinds[i] = extension(g, symbol.Extensions, "type")
		view.unstable[i] = symbol.Role == pdg.GlobalRole || extension(g, symbol.Extensions, "unstable") == "true"
	}
	for _, edge := range g.Edges {
		view.operandEdge(edge)
	}
	view.fieldBindings()
	view.trackBindings()
	return view, nil
}

func (v *graphView) operandEdge(edge pdg.Edge) {
	if !sourceOperandEdge(v.graph, edge) || !operandReferences(edge, len(v.graph.Nodes)) {
		return
	}
	v.operands[edge.Target-1][edge.Position-1] = edge.Source
}

func (v *graphView) operand(ref pdg.Ref, position int) pdg.Ref {
	if ref == 0 || position >= len(v.operands[ref-1]) {
		return 0
	}
	return v.operands[ref-1][position]
}

func (v *graphView) attrs(ref pdg.Ref) pdg.Attributes {
	if ref == 0 {
		return pdg.Attributes{}
	}
	return v.graph.NodeAttributes(v.graph.Nodes[ref-1])
}

func (v *graphView) fact(ref pdg.Ref, name string) string {
	return extension(v.graph, v.attrs(ref).Extensions, name)
}

func (v *graphView) symbol(ref pdg.Ref) pdg.Ref {
	if ref == 0 {
		return 0
	}
	if v.fields[ref-1] != 0 {
		return v.fields[ref-1]
	}
	attrs := v.attrs(ref)
	if directBindingRead(v.graph.Nodes[ref-1], attrs) {
		symbol := attrs.Reads[0].Symbol
		if int(symbol) > len(v.kinds) {
			return 0
		}
		return symbol
	}
	operator := v.graph.Tables.Text(attrs.Operator.Value)
	if operator == "parentheses" {
		return v.symbol(v.operand(ref, 0))
	}
	return 0
}

func (v *graphView) base(symbol pdg.Ref) Value {
	if symbol == 0 || int(symbol) > len(v.kinds) {
		return Value{}
	}
	if v.kinds[symbol-1] == `"unsigned"` {
		return Value{Lower: 0, HasLower: true}
	}
	return Value{}
}

func (v *graphView) binding(symbol pdg.Ref, state State) Value {
	if symbol == 0 || int(symbol) > len(v.kinds) {
		return Value{}
	}
	if v.unstable[symbol-1] {
		return v.base(symbol)
	}
	if value, ok := state[symbol]; ok {
		return value
	}
	value := v.base(symbol)
	if v.kinds[symbol-1] == `"pointer"` {
		value.Identity = uint64(symbol) * 2
	}
	return value
}

func (v *graphView) value(ref pdg.Ref, state State) Value {
	if ref == 0 {
		return Value{}
	}
	attrs := v.attrs(ref)
	value := v.constantValue(ref)
	if symbol := v.symbol(ref); symbol != 0 {
		return v.binding(symbol, state)
	}
	operator := v.graph.Tables.Text(attrs.Operator.Value)
	if operator == "parentheses" {
		return v.value(v.operand(ref, 0), state)
	}
	if operator == "conversion" {
		return v.conversionValue(ref, state, value)
	}
	value.Length = attrs.Callee.Status == pdg.Known && v.graph.Tables.Text(attrs.Callee.Value) == "len"
	return value
}

func (v *graphView) constantValue(ref pdg.Ref) Value {
	value := Value{NonNil: v.fact(ref, "non-nil") == "true", Nil: v.fact(ref, "nil") == "true"}
	if integer, err := strconv.ParseInt(v.fact(ref, "integer"), 10, 64); err == nil {
		value.Lower, value.Upper = integer, integer
		value.HasLower, value.HasUpper, value.Integer = true, true, true
	}
	if extent, err := strconv.ParseInt(v.fact(ref, "literal-extent"), 10, 64); err == nil {
		value.Extent, value.HasExtent = extent, true
	}
	return value
}

func (v *graphView) conversionValue(ref pdg.Ref, state State, constant Value) Value {
	if constant.Integer {
		return constant
	}
	source := v.value(v.operand(ref, 0), state)
	switch v.fact(ref, "type") {
	case `"pointer"`:
		return Value{NonNil: source.NonNil, Nil: source.Nil}
	case `"unsigned"`:
		return Value{HasLower: true, Lower: 0, Length: source.Length}
	default:
		return Value{Length: source.Length}
	}
}

// The guard projection consumes binary operands, a unary operand, or the first
// RHS of a single-binding assignment. All other operands remain in the PDG.
func sourceOperandEdge(graph *pdg.Graph, edge pdg.Edge) bool {
	if edge.Kind != pdg.Data || edge.Position == 0 || edge.Position > 2 {
		return false
	}
	kind := graph.Tables.Text(edge.Subkind)
	return kind == "source-value" || kind == "scalar-value"
}

func operandReferences(edge pdg.Edge, nodes int) bool {
	return edge.Source != 0 && int(edge.Source) <= nodes && edge.Target != 0 && int(edge.Target) <= nodes
}

func extension(g *pdg.Graph, values []pdg.Extension, name string) string {
	for _, value := range values {
		if g.Tables.Text(value.Name) == "gstyle.runtime:"+name {
			return g.Tables.Text(value.Value)
		}
	}
	return ""
}

func validateFlow(g *pdg.Graph) error {
	if g.Tables == nil || g.ValueFlow.Entry == 0 || int(g.ValueFlow.Entry) > len(g.ValueFlow.Blocks) {
		return fmt.Errorf("invalid PDG value-flow entry")
	}
	for _, block := range g.ValueFlow.Blocks {
		if err := validateBlock(g, block); err != nil {
			return err
		}
	}
	return nil
}

func validateBlock(g *pdg.Graph, block pdg.FlowBlock) error {
	if int(block.Condition) > len(g.Nodes) {
		return fmt.Errorf("invalid PDG value-flow condition")
	}
	if err := validateRefs(block.Operations, len(g.Nodes)); err != nil {
		return err
	}
	return validateRefs(block.Successors, len(g.ValueFlow.Blocks))
}

func validateRefs(refs []pdg.Ref, count int) error {
	for _, ref := range refs {
		if ref == 0 || int(ref) > count {
			return fmt.Errorf("invalid PDG value-flow reference")
		}
	}
	return nil
}

func directBindingRead(node pdg.Node, attrs pdg.Attributes) bool {
	return node.Category == pdg.Read && len(attrs.Reads) == 1 && attrs.Reads[0].Location == 0
}
