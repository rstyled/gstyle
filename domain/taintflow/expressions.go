package taintflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"strings"
)

// Field paths are bounded independently of recursion depth. A widened suffix
// keeps a may-dependency; it never establishes an exact alias or safe value.
func appendFields(left, right string) string {
	parts := strings.Split(strings.Trim(left+"/"+right, "/"), "/")
	if len(parts) > 8 {
		return strings.Join(parts[:8], "/") + "/*"
	}
	return strings.Join(parts, "/")
}
func (e *evaluation) input(owner pdg.Ref, position pdg.Position) value {
	operand := e.query.paths.Operand(owner, position)
	if item, found := e.state.Nodes[slot{Node: operand}]; found {
		return item
	}
	origins := e.query.paths.OriginsAt(owner, position, bindingflow.Facts{})
	result := emptyValue()
	for _, source := range origins.Sources {
		result = joinValue(result, e.origin(source))
	}
	if origins.Known {
		return result
	}
	return joinValue(result, e.node(e.query.paths.Operand(owner, position), 0))
}
func (e *evaluation) origin(source bindingflow.ValueSource) value {
	if source.Zero {
		item := emptyValue()
		item.Nil = e.query.paths.ZeroValue(source.Symbol).Nil
		return item
	}
	if source.Symbol != 0 {
		return e.symbolValue(source.Symbol)
	}
	return e.node(source.Node, source.Result)
}
func (e *evaluation) node(ref pdg.Ref, result pdg.Position) value {
	if ref == 0 {
		return emptyValue()
	}
	item, found := e.state.Nodes[slot{Node: ref, Result: result}]
	if found {
		return item
	}
	if result != 0 {
		return e.state.Nodes[slot{Node: ref}]
	}
	return emptyValue()
}
func (e *evaluation) expression(ref pdg.Ref) value {
	attrs := e.query.graph.NodeAttributes(e.query.graph.Nodes[ref-1])
	if item, known := e.constructedValue(ref); known {
		return item
	}
	if attrs.Literal.Status == pdg.Known {
		item := emptyValue()
		item.Nil = e.query.facts.Nodes[ref].Value.Nil
		return item
	}
	if len(attrs.Reads) == 1 && attrs.Reads[0].DirectSymbol() {
		return e.symbolValue(attrs.Reads[0].Symbol)
	}
	if item, found := e.memoryValue(ref, attrs); found {
		return item
	}
	return e.expressionEncoding(ref, e.expressionInputs(ref))
}
func (e *evaluation) expressionInputs(ref pdg.Ref) value {
	result := emptyValue()
	positions, _ := e.query.paths.RequiredOperands(ref)
	for _, position := range positions {
		result = joinValue(result, e.input(ref, position))
	}
	return result
}
func (e *evaluation) fieldValue(ref pdg.Ref) value {
	base := e.node(e.query.paths.Operand(ref, 1), 0)
	fields := fieldKey(e.query.paths.FieldIndexes(ref))
	return pathValue(e.state, base, fields)
}
func (e *evaluation) assignment(ref pdg.Ref) {
	for _, write := range e.query.paths.Writes(ref) {
		e.write(ref, write)
	}
	e.store(slot{Node: ref}, e.expressionInputs(ref))
}
func (e *evaluation) write(ref pdg.Ref, write bindingflow.Write) {
	if write.Position == 0 {
		return
	}
	item := e.input(ref, write.Position)
	if write.Zero {
		item = emptyValue()
		item.Nil = e.query.paths.ZeroValue(write.Symbol).Nil
	}
	if write.UnknownTarget() {
		return
	}
	if write.Location != 0 {
		e.writeField(ref, write.Position, item)
		return
	}
	e.state.Bindings[write.Symbol] = e.bindingObject(write.Symbol, item)
	e.writeCapture(write.Symbol, item)
}
func (e *evaluation) bindingObject(symbol pdg.Ref, item value) value {
	if e.query.paths.ImmutableSymbol(symbol) {
		item.Objects = map[object]bool{}
		return item
	}
	if len(item.Objects) != 0 {
		return item
	}
	result := item
	result.Objects = map[object]bool{{Node: symbol, Allocation: allocationIdentity(e.query.name+":binding", symbol)}: true}
	return result
}
func (e *evaluation) writeField(ref pdg.Ref, position pdg.Position, item value) {
	target := e.query.addresses[ref][position]
	if target == 0 {
		return
	}
	base := e.node(e.query.paths.Operand(target, 1), 0)
	fields := fieldKey(e.query.paths.FieldIndexes(target))
	targets := pathTargets(e.state, base, fields)
	for _, target := range targets {
		next := item
		if len(targets) != 1 || strings.Contains(target.Fields, "*") {
			next = joinValue(readCell(e.state, target), next)
		}
		e.state.Memory[target] = next
	}
}

func (e *evaluation) allocation(ref pdg.Ref) value {
	result := emptyValue()
	attrs := e.query.graph.NodeAttributes(e.query.graph.Nodes[ref-1])
	result.Opaque = e.query.graph.Tables.Text(attrs.Construct) == "closure"
	result.Objects[object{Node: ref, Allocation: allocationIdentity(e.query.name, ref)}] = true
	return result
}

func (e *evaluation) store(key slot, item value) {
	if e.query.paths.ImmutableValue(key.Node) {
		item.Objects = map[object]bool{}
	}
	e.state.Nodes[key] = item
}

func (e *evaluation) constructedValue(ref pdg.Ref) (value, bool) {
	if item, known := e.functionValue(ref); known {
		return item, true
	}
	if e.query.paths.IsAllocation(ref) {
		return e.allocation(ref), true
	}
	return value{}, false
}
