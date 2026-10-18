package taintflow

import (
	"cmp"
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
)

func (q *query) analyze(solver *summarySolver) summary {
	result := emptySummary()
	inputs := map[pdg.Ref]state{q.flow.Entry: q.initial()}
	e := evaluation{query: q, solver: solver, catalog: solver.queries, summaries: solver.summaries, result: &result, exits: map[pdg.Ref]state{}}
	e.solveBody(inputs)
	summarizeWrites(&result, e.exits)
	return result
}

type bodyWorklist struct {
	queue   []pdg.Ref
	pending map[pdg.Ref]bool
	inputs  map[pdg.Ref]state
}

func (e *evaluation) solveBody(inputs map[pdg.Ref]state) {
	work := bodyWorklist{queue: []pdg.Ref{e.query.flow.Entry}, pending: map[pdg.Ref]bool{e.query.flow.Entry: true}, inputs: inputs}
	for cursor := 0; cursor < len(work.queue); cursor++ {
		ref := work.queue[cursor]
		delete(work.pending, ref)
		e.state = cloneState(inputs[ref])
		block := e.query.flow.Blocks[ref-1]
		if !block.Reachable {
			continue
		}
		for _, operation := range block.Operations {
			e.operation(operation)
		}
		e.retainLiveNodes()
		for _, next := range block.Successors {
			work.enqueue(next, e.state)
		}
	}
}
func (work *bodyWorklist) enqueue(target pdg.Ref, next state) {
	if !mergeInput(work.inputs, target, next) || work.pending[target] {
		return
	}
	work.pending[target] = true
	work.queue = append(work.queue, target)
}
func mergeInput(inputs map[pdg.Ref]state, target pdg.Ref, next state) bool {
	old, reached := inputs[target]
	if !reached {
		inputs[target] = next
		return true
	}
	joined := joinState(old, next)
	if stateEqual(old, joined) {
		return false
	}
	inputs[target] = joined
	return true
}
func (q *query) indexLiveNodes() {
	blocks := map[pdg.Ref]int{}
	for index, block := range q.flow.Blocks {
		for _, ref := range block.Operations {
			blocks[ref] = index + 1
		}
	}
	q.live = map[pdg.Ref]bool{}
	for _, edge := range q.graph.Edges {
		if crossBlockValue(edge, blocks) {
			q.live[edge.Source] = true
		}
	}
}
func (e *evaluation) retainLiveNodes() {
	kept := map[slot]value{}
	for key, item := range e.state.Nodes {
		if e.query.live[key.Node] {
			kept[key] = item
		}
	}
	e.state.Nodes = kept
}
func joinState(left, right state) state {
	result := cloneState(left)
	maps.Copy(result.Repeated, right.Repeated)
	for key, item := range right.Bindings {
		result.Bindings[key] = joinValue(result.Bindings[key], item)
	}
	for key, item := range right.Nodes {
		result.Nodes[key] = joinValue(result.Nodes[key], item)
	}
	for key, item := range left.Memory {
		result.Memory[key] = joinValue(item, readCell(right, key))
	}
	for key, item := range right.Memory {
		result.Memory[key] = joinValue(item, readCell(left, key))
	}
	return result
}
func readCell(store state, key cell) value {
	if item, known := store.Memory[key]; known {
		return wildcardWrites(store, key, item)
	}
	item := emptyValue()
	nested := key.Object
	nested.Fields = appendFields(nested.Fields, key.Fields)
	item.Objects[nested] = true
	if key.Object.Input != 0 {
		item.Data[atom{Input: key.Object.Input, Fields: appendFields(key.Object.Fields, key.Fields)}] = true
	}
	return wildcardWrites(store, key, item)
}
func (e *evaluation) operation(ref pdg.Ref) {
	node := e.query.graph.Nodes[ref-1]
	switch node.Category {
	case pdg.Call:
		e.call(ref)
	case pdg.Assignment, pdg.Declaration:
		e.assignment(ref)
	case pdg.Return:
		e.returned(ref)
	default:
		e.store(slot{Node: ref}, e.expression(ref))
	}
}
func (e *evaluation) returned(ref pdg.Ref) {
	for index := range e.query.graph.Function.Outputs {
		position := pdg.Position(index + 1)
		item := e.input(ref, position)
		e.result.Results[position] = joinValue(e.result.Results[position], item)
	}
	e.exits[ref] = state{Memory: maps.Clone(e.state.Memory), Repeated: maps.Clone(e.state.Repeated)}
}
func summarizeWrites(result *summary, exits map[pdg.Ref]state) {
	var joined state
	first := true
	for _, exit := range exits {
		if first {
			joined = exit
			first = false
			continue
		}
		joined = joinState(joined, exit)
	}
	maps.Copy(result.Repeated, joined.Repeated)
	for key, item := range joined.Memory {
		result.Writes[key] = item
	}
}
func compareFinding(left, right Finding) int {
	if order := cmp.Compare(left.Sink.Function, right.Sink.Function); order != 0 {
		return order
	}
	if order := cmp.Compare(left.Sink.Node, right.Sink.Node); order != 0 {
		return order
	}
	if order := cmp.Compare(left.Source.Function, right.Source.Function); order != 0 {
		return order
	}
	if order := cmp.Compare(left.Source.Node, right.Source.Node); order != 0 {
		return order
	}
	return cmp.Compare(left.Kind, right.Kind)
}

func crossBlockValue(edge pdg.Edge, blocks map[pdg.Ref]int) bool {
	return edge.Kind == pdg.Data && blocks[edge.Source] != blocks[edge.Target]
}
