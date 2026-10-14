package runtimeguard

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"slices"
	"strconv"
)

type analysis struct {
	view   *graphView
	inputs []State
	queued []bool
	queue  []pdg.Ref
}

func (a *analysis) enqueue(ref pdg.Ref) {
	if a.queued[ref-1] {
		return
	}
	a.queued[ref-1] = true
	a.queue = append(a.queue, ref)
}

func (a *analysis) successors(ref pdg.Ref, state State) {
	block := a.view.graph.ValueFlow.Blocks[ref-1]
	for outcome, next := range block.Successors {
		branch := copyState(state)
		if block.Condition != 0 {
			branch = a.view.assume(block.Condition, outcome == 0, branch)
		}
		if branch == nil {
			continue
		}
		joined := a.view.join(a.inputs[next-1], branch)
		if equalState(a.inputs[next-1], joined) && a.inputs[next-1] != nil {
			continue
		}
		a.inputs[next-1] = joined
		a.enqueue(next)
	}
}

func (a *analysis) block(block pdg.FlowBlock, state State, report *Report) {
	for _, ref := range block.Operations {
		if report != nil {
			a.access(ref, state, report)
		}
		a.transfer(ref, state)
	}
}

func (a *analysis) transfer(ref pdg.Ref, state State) {
	attrs := a.view.attrs(ref)
	if len(attrs.Writes) == 0 {
		return
	}
	value := Value{}
	form := a.view.graph.Tables.Text(attrs.WriteForm)
	if singleValueWrite(a.view.graph.Nodes[ref-1].Category, form, len(attrs.Writes)) {
		value = a.view.value(a.view.operand(ref, 0), state)
	}
	value.Guarded = false
	for _, write := range attrs.Writes {
		if !a.view.trackedBinding(write.Symbol) {
			continue
		}
		state[write.Symbol] = a.view.assignmentValue(write.Symbol, value)
		a.rangeWrite(ref, write.Symbol, state)
	}
}

func (a *analysis) report() Report {
	report := Report{}
	for i, block := range a.view.graph.ValueFlow.Blocks {
		if a.inputs[i] == nil {
			continue
		}
		a.block(block, copyState(a.inputs[i]), &report)
	}
	slices.SortStableFunc(report.Accesses, func(a, b Access) int { return int(a.Node) - int(b.Node) })
	return report
}

func (a *analysis) access(ref pdg.Ref, state State, report *Report) {
	pointer, index := a.view.fact(ref, "dereference") == "true", a.view.fact(ref, "index") == "true"
	if !pointer && !index {
		return
	}
	position := 0
	if index {
		position = 1
	}
	operand := a.view.operand(ref, position)
	site := Access{Node: ref, Symbol: a.view.symbol(operand), Value: a.view.value(operand, state), Index: index}
	if index {
		base := a.view.value(a.view.operand(ref, 0), state)
		site.Value.Extent, site.Value.HasExtent = base.Extent, base.HasExtent
		site.Literal = a.view.fact(ref, "literal-index") == "true"
		site.Constant = a.view.fact(operand, "integer") != ""
	}
	if a.view.fact(ref, "embedded-dereference") == "true" {
		site.Symbol, site.Value = 0, Value{}
	}
	site.Guards = pointerGuardContext(site, state)
	report.Accesses = append(report.Accesses, site)
}

func (v *graphView) join(previous, next State) State {
	if previous == nil {
		return copyState(next)
	}
	if next == nil {
		return copyState(previous)
	}
	out := make(State)
	for symbol, value := range previous {
		out[symbol] = value.merge(v.binding(symbol, next))
	}
	for symbol, value := range next {
		if _, present := previous[symbol]; !present {
			out[symbol] = value.merge(v.binding(symbol, previous))
		}
	}
	return out
}

func (v *graphView) assignmentValue(symbol pdg.Ref, value Value) Value {
	base := v.base(symbol)
	if base.HasLower && !value.HasLower {
		value.Lower, value.HasLower = base.Lower, true
	}
	return value
}

func Analyze(g *pdg.Graph) (Report, error) {
	if g == nil || g.ValueFlow == nil {
		return Report{}, nil
	}
	view, err := newGraphView(g)
	if err != nil {
		return Report{}, err
	}
	a := analysis{view: view, inputs: make([]State, len(g.ValueFlow.Blocks)), queued: make([]bool, len(g.ValueFlow.Blocks))}
	a.inputs[g.ValueFlow.Entry-1] = State{}
	a.enqueue(g.ValueFlow.Entry)
	for cursor := 0; cursor < len(a.queue); cursor++ {
		ref := a.queue[cursor]
		a.queued[ref-1] = false
		state := copyState(a.inputs[ref-1])
		a.block(g.ValueFlow.Blocks[ref-1], state, nil)
		a.successors(ref, state)
	}
	return a.report(), nil
}

func singleValueWrite(category pdg.Category, form string, writes int) bool {
	if writes != 1 {
		return false
	}
	return form == "=" || form == ":=" || (form == "" && category == pdg.Declaration)
}

func activePointerGuards(state State, target pdg.Ref) []pdg.Ref {
	var guards []pdg.Ref
	for symbol, value := range state {
		if value.Guarded && symbol != target {
			guards = append(guards, symbol)
		}
	}
	slices.Sort(guards)
	return guards
}

func pointerGuardContext(site Access, state State) []pdg.Ref {
	if site.Index || site.Value.NonNil {
		return nil
	}
	return activePointerGuards(state, site.Symbol)
}

func (a *analysis) rangeWrite(ref, symbol pdg.Ref, state State) {
	index, err := strconv.ParseUint(a.view.fact(ref, "range-index"), 10, 32)
	if err == nil && pdg.Ref(index) == symbol {
		state[symbol] = Value{HasLower: true, Lower: 0}
	}
}
