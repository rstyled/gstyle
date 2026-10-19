package taintflow

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
	"strconv"
)

func allocationIdentity(name string, ref pdg.Ref) string {
	return name + ":" + strconv.FormatUint(uint64(ref), 10)
}
func (q *query) indexCells() {
	q.cells = map[pdg.Ref]cell{}
	for i, symbol := range q.graph.Symbols {
		ref := pdg.Ref(i + 1)
		if !q.facts.CapturedSymbols[ref] {
			continue
		}
		root := object{Captured: true, Node: ref, Allocation: allocationIdentity(q.name+":cell", ref)}
		if symbol.Role == pdg.CaptureRole {
			q.captures = append(q.captures, ref)
			position := len(q.graph.Function.Inputs) + len(q.captures)
			q.inputs[ref] = position
			root = object{Captured: true, Input: position}
		}
		q.cells[ref] = cell{Object: root, Fields: "value"}
	}
}
func (q *query) initialInput(symbol pdg.Ref, position int) value {
	item := emptyValue()
	if q.graph.Symbols[symbol-1].Role == pdg.CaptureRole {
		return readCell(emptyState(), q.cells[symbol])
	}
	item.Data[atom{Input: position}] = true
	if !q.paths.ImmutableSymbol(symbol) {
		item.Objects[object{Input: position}] = true
	}
	return item
}
func linkEnvironments(queries map[string]*query) error {
	for _, owner := range queries {
		owner.environments = map[string]map[pdg.Ref]pdg.Ref{}
		for i, node := range owner.graph.Nodes {
			attrs := owner.graph.NodeAttributes(node)
			if owner.graph.Tables.Text(attrs.Construct) != "closure" {
				continue
			}
			if err := owner.linkEnvironment(queries, pdg.Ref(i+1)); err != nil {
				return err
			}
		}
	}
	return nil
}
func (q *query) linkEnvironment(queries map[string]*query, ref pdg.Ref) error {
	name, err := bindingflow.CallableIdentity(q.graph, ref)
	if err != nil {
		return err
	}
	body := queries[name]
	if body == nil {
		return nil
	}
	return q.attachCaptureBindings(name, body)
}

func (q *query) attachCaptureBindings(name string, body *query) error {
	links, err := bindingflow.CaptureBindings(body.graph, q.graph)
	if err != nil {
		return fmt.Errorf("taint closure %s: %w", name, err)
	}
	if links == nil && len(body.captures) != 0 {
		return fmt.Errorf("taint closure %s lacks capture bindings", name)
	}
	for _, symbol := range links {
		q.ensureCaptureCell(symbol)
	}
	q.environments[name] = links
	return nil
}
func (e *evaluation) symbolValue(symbol pdg.Ref) value {
	if key, found := e.query.cells[symbol]; found {
		return readCell(e.state, key)
	}
	return e.state.Bindings[symbol]
}
func (e *evaluation) writeCapture(symbol pdg.Ref, item value) {
	key, found := e.query.cells[symbol]
	if !found {
		return
	}
	if !e.query.facts.SingleCellSymbols[symbol] {
		e.state.Repeated[key.Object] = true
		item = joinValue(readCell(e.state, key), item)
	}
	e.state.Memory[key] = item
}
func (e *evaluation) closureValue(ref pdg.Ref, name string) value {
	attrs := e.query.graph.NodeAttributes(e.query.graph.Nodes[ref-1])
	if e.query.graph.Tables.Text(attrs.Construct) != "closure" {
		return emptyValue()
	}
	item := e.allocation(ref)
	links, known := e.query.environments[name]
	if !known {
		return e.opaqueEnvironment(ref, item)
	}
	item.Opaque = false
	for inner, outer := range links {
		key, found := e.query.cells[outer]
		if !found {
			item.Opaque = true
			continue
		}
		reference := emptyValue()
		reference.Objects[key.Object] = true
		for root := range item.Objects {
			e.state.Memory[cellKey(root, captureField(inner))] = reference
		}
	}
	return item
}
func captureField(symbol pdg.Ref) string { return "capture:" + strconv.FormatUint(uint64(symbol), 10) }
func (e *evaluation) targetValue(ref pdg.Ref) value {
	source := e.query.calls.TargetOperand(ref)
	if item, found := e.state.Nodes[slot{Node: source}]; found {
		return item
	}
	result := emptyValue()
	origins := e.query.paths.OriginsOf(source, bindingflow.Facts{})
	for _, origin := range origins.Sources {
		result = joinValue(result, e.origin(origin))
	}
	return result
}
func (e *evaluation) captureArguments(ref pdg.Ref, callee *query, arguments map[pdg.Position]value) {
	target := e.targetValue(ref)
	for index, symbol := range callee.captures {
		position := pdg.Position(len(callee.graph.Function.Inputs) + index + 1)
		arguments[position] = pathValue(e.state, target, captureField(symbol))
	}
}

func (e *evaluation) functionArguments(callee *query, arguments map[pdg.Position]value) map[pdg.Position]value {
	result := make(map[pdg.Position]value, len(arguments))
	for position, item := range arguments {
		result[position] = item
	}
	for i := range callee.captures {
		position := pdg.Position(len(callee.graph.Function.Inputs) + i + 1)
		result[position] = pathValue(e.state, arguments[position], "value")
	}
	return result
}

func (e *evaluation) opaqueEnvironment(ref pdg.Ref, item value) value {
	item.Opaque = true
	attrs := e.query.graph.NodeAttributes(e.query.graph.Nodes[ref-1])
	for _, symbol := range attrs.Symbols {
		maps.Copy(item.Data, e.symbolValue(symbol).Data)
		key, found := e.query.cells[symbol]
		if !found {
			continue
		}
		reference := emptyValue()
		reference.Objects[key.Object] = true
		for root := range item.Objects {
			e.state.Memory[cellKey(root, captureField(symbol))] = reference
		}
	}
	return item
}

func (q *query) ensureCaptureCell(symbol pdg.Ref) {
	if _, found := q.cells[symbol]; found {
		return
	}
	q.cells[symbol] = cell{Object: object{Captured: true, Node: symbol, Allocation: allocationIdentity(q.name+":cell", symbol)}, Fields: "value"}
}
