package govalidation

import (
	"fmt"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

type definitions map[pdg.Ref][]pdg.Ref

type reachingCheck struct {
	g        *pdg.Graph
	incoming map[pdg.Ref][]pdg.Ref
	used     map[pdg.Ref]bool
}

func (a *reachingCheck) mergeSites() error {
	for i, definition := range a.g.Definitions {
		if a.g.Nodes[definition.Node-1].Category != pdg.Merge {
			continue
		}
		if !a.used[pdg.Ref(i+1)] {
			return fmt.Errorf("binding merge has no verified reachable value use")
		}
	}
	return definitionTargets(a.g, a.incoming)
}

func (a *reachingCheck) flow() error {
	next := blockTransitions(a.g.NormalFlow)
	predecessors, _ := predecessorCounts(next)
	remaining := consumerCounts(next)
	order, err := topologicalBlocks(next, a.g.NormalFlow.Entry)
	if err != nil {
		return err
	}
	states := map[pdg.Ref]definitions{}
	for _, block := range order {
		state := mergeDefinitionStates(predecessors[block], states, remaining)
		if block == a.g.NormalFlow.Entry {
			a.seed(state)
		}
		if err := a.block(block, state); err != nil {
			return err
		}
		if len(next[block]) != 0 {
			states[block] = state
		}
	}
	return nil
}

func (a *reachingCheck) block(block pdg.Ref, state definitions) error {
	for _, ref := range a.g.NormalFlow.Blocks[block-1].Operations {
		if err := a.operation(ref, state); err != nil {
			return err
		}
	}
	return nil
}

func (a *reachingCheck) seed(state definitions) {
	for i, definition := range a.g.Definitions {
		category := a.g.Nodes[definition.Node-1].Category
		if category == pdg.FormalInput || category == pdg.FormalOutput {
			state[definition.Symbol] = []pdg.Ref{pdg.Ref(i + 1)}
		}
	}
}

func (a *reachingCheck) operation(ref pdg.Ref, state definitions) error {
	node := a.g.Nodes[ref-1]
	attrs := a.g.NodeAttributes(node)
	for _, read := range attrs.Reads {
		if err := a.read(ref, read.Symbol, state[read.Symbol]); err != nil {
			return err
		}
	}
	for _, definition := range attrs.Defines {
		symbol := a.g.Definitions[definition-1].Symbol
		state[symbol] = []pdg.Ref{definition}
	}
	return nil
}

func (a *reachingCheck) read(ref, symbol pdg.Ref, expected []pdg.Ref) error {
	if len(expected) == 0 {
		return fmt.Errorf("read %d has no reaching source definition", ref)
	}
	if len(a.incoming[ref]) != 1 {
		return fmt.Errorf("read %d requires one reaching value definition", ref)
	}
	definition := a.incoming[ref][0]
	if a.g.Definitions[definition-1].Symbol != symbol {
		return fmt.Errorf("reaching edge carries a different binding")
	}
	actual, err := a.leaves(definition)
	if err != nil {
		return err
	}
	if !slices.Equal(sortedRefs(actual), sortedRefs(expected)) {
		return fmt.Errorf("reaching definitions disagree at read %d", ref)
	}
	return nil
}

// Each read uses linear scratch. Do not cache expanded definition histories:
// nested joins would otherwise retain a quadratic number of leaf references.
func (a *reachingCheck) leaves(ref pdg.Ref) ([]pdg.Ref, error) {
	walk := mergeWalk{check: a, state: map[pdg.Ref]uint8{}, stack: []mergeFrame{{ref: ref}}}
	return walk.run()
}

func (a *reachingCheck) mergeInputs(definition pdg.Definition, inputs []pdg.Ref) error {
	if len(inputs) < 2 {
		return fmt.Errorf("binding merge has fewer than two input definitions")
	}
	if len(sortedRefs(inputs)) != len(inputs) {
		return fmt.Errorf("binding merge repeats an input definition")
	}
	for _, input := range inputs {
		if a.g.Definitions[input-1].Symbol != definition.Symbol {
			return fmt.Errorf("merge combines different bindings")
		}
	}
	return nil
}

type mergeFrame struct {
	ref   pdg.Ref
	leave bool
}

type mergeWalk struct {
	check  *reachingCheck
	state  map[pdg.Ref]uint8
	stack  []mergeFrame
	leaves []pdg.Ref
}

func (w *mergeWalk) run() ([]pdg.Ref, error) {
	for len(w.stack) > 0 {
		frame := w.stack[len(w.stack)-1]
		w.stack = w.stack[:len(w.stack)-1]
		if err := w.visit(frame); err != nil {
			return nil, err
		}
	}
	return sortedRefs(w.leaves), nil
}

func (w *mergeWalk) visit(frame mergeFrame) error {
	if frame.leave {
		w.state[frame.ref] = 2
		return nil
	}
	if w.state[frame.ref] == 1 {
		return fmt.Errorf("cyclic scalar value merge")
	}
	if w.state[frame.ref] == 2 {
		return nil
	}
	definition := w.check.g.Definitions[frame.ref-1]
	if w.check.g.Nodes[definition.Node-1].Category != pdg.Merge {
		w.state[frame.ref] = 2
		w.leaves = append(w.leaves, frame.ref)
		return nil
	}
	return w.expand(frame.ref, definition)
}

func (w *mergeWalk) expand(ref pdg.Ref, definition pdg.Definition) error {
	inputs := w.check.incoming[definition.Node]
	if err := w.check.mergeInputs(definition, inputs); err != nil {
		return err
	}
	w.state[ref] = 1
	w.check.used[ref] = true
	w.stack = append(w.stack, mergeFrame{ref: ref, leave: true})
	for _, input := range inputs {
		w.stack = append(w.stack, mergeFrame{ref: input})
	}
	return nil
}

// Reaching compares actual definition paths, expanding producer merge entities,
// with an independent kill/union analysis on the validated acyclic CFG.
func Reaching(g *pdg.Graph) error {
	if err := pdg.Validate(g); err != nil {
		return err
	}
	if g.NormalFlow == nil {
		return fmt.Errorf("normal-flow context is missing")
	}
	a := reachingCheck{g: g, incoming: definitionInputs(g), used: map[pdg.Ref]bool{}}
	if err := a.flow(); err != nil {
		return err
	}
	return a.mergeSites()
}

func definitionTargets(g *pdg.Graph, incoming map[pdg.Ref][]pdg.Ref) error {
	for target := range incoming {
		category := g.Nodes[target-1].Category
		if category != pdg.Read && category != pdg.Merge {
			return fmt.Errorf("definition edge targets a non-binding value site")
		}
	}
	return nil
}

func definitionInputs(g *pdg.Graph) map[pdg.Ref][]pdg.Ref {
	out := map[pdg.Ref][]pdg.Ref{}
	for _, edge := range g.Edges {
		if edge.Kind == pdg.Data && edge.Definition != 0 {
			out[edge.Target] = append(out[edge.Target], edge.Definition)
		}
	}
	return out
}

func consumerCounts(next map[pdg.Ref][]pdg.Ref) map[pdg.Ref]int {
	out := map[pdg.Ref]int{}
	for ref, successors := range next {
		out[ref] = len(sortedRefs(successors))
	}
	return out
}

func blockTransitions(flow *pdg.NormalFlow) map[pdg.Ref][]pdg.Ref {
	out := map[pdg.Ref][]pdg.Ref{}
	for i, block := range flow.Blocks {
		if block.Reachable {
			out[pdg.Ref(i+1)] = block.Successors
		}
	}
	return out
}

func topologicalBlocks(next map[pdg.Ref][]pdg.Ref, entry pdg.Ref) ([]pdg.Ref, error) {
	_, remaining := predecessorCounts(next)
	queue := []pdg.Ref{entry}
	for i := 0; i < len(queue); i++ {
		for _, to := range sortedRefs(next[queue[i]]) {
			remaining[to]--
			if remaining[to] == 0 {
				queue = append(queue, to)
			}
		}
	}
	if len(queue) != len(next) {
		return nil, fmt.Errorf("normal CFG is not a reachable acyclic graph")
	}
	return queue, nil
}

func mergeDefinitionStates(predecessors []pdg.Ref, states map[pdg.Ref]definitions, remaining map[pdg.Ref]int) definitions {
	state := definitions{}
	for _, previous := range predecessors {
		for symbol, refs := range states[previous] {
			state[symbol] = sortedRefs(append(state[symbol], refs...))
		}
		remaining[previous]--
		if remaining[previous] == 0 {
			delete(states, previous)
		}
	}
	return state
}
