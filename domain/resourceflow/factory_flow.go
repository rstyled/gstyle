package resourceflow

import "github.com/rstyled/gstyle/domain/pdg"

func newFactoryState() factoryState {
	return factoryState{issued: map[pdg.Ref]bool{}, bad: map[pdg.Ref]bool{}}
}

func (state factoryState) clone() factoryState {
	copy := newFactoryState()
	copy.join(state)
	return copy
}

func (state *factoryState) join(other factoryState) bool {
	changed := joinFactorySet(state.issued, other.issued)
	changed = joinFactorySet(state.bad, other.bad) || changed
	if other.unknownExit && !state.unknownExit {
		state.unknownExit = true
		changed = true
	}
	return changed
}

func (a *factoryAnalysis) solve() {
	entry := int(a.query.flow.Entry - 1)
	a.inputs[entry] = newFactoryState()
	queue := []int{entry}
	pending := make([]bool, len(a.inputs))
	pending[entry] = true
	for len(queue) != 0 {
		block := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		pending[block] = false
		state := a.inputs[block].clone()
		for _, ref := range a.query.flow.Blocks[block].Operations {
			a.effect(ref, &state)
		}
		queue = a.propagate(block, state, queue, pending)
	}
}

func (a *factoryAnalysis) propagate(block int, state factoryState, queue []int, pending []bool) []int {
	for _, next := range a.query.flow.Blocks[block].Successors {
		index := int(next - 1)
		changed := a.joinInput(index, state)
		if !changed || pending[index] {
			continue
		}
		pending[index] = true
		queue = append(queue, index)
	}
	return queue
}

func (a *factoryAnalysis) joinInput(index int, state factoryState) bool {
	if a.inputs[index].issued == nil {
		a.inputs[index] = state.clone()
		return true
	}
	return a.inputs[index].join(state)
}

func (a *factoryAnalysis) describe() factoryDescriptor {
	descriptor := factoryDescriptor{valid: true}
	returned := false
	for i := range a.query.flow.Blocks {
		if a.inputs[i].issued == nil {
			continue
		}
		returned = a.describeBlock(i, &descriptor) || returned
	}
	descriptor.valid = descriptor.valid && returned
	return descriptor
}

func (a *factoryAnalysis) describeBlock(index int, descriptor *factoryDescriptor) bool {
	returned := false
	state := a.inputs[index].clone()
	for _, ref := range a.query.flow.Blocks[index].Operations {
		if a.query.graph.Nodes[ref-1].Category == pdg.Return {
			returned = true
			if !a.describeReturn(ref, state, descriptor) {
				descriptor.valid = false
			}
		}
		a.effect(ref, &state)
	}
	return returned
}

func (a *factoryAnalysis) describeReturn(ref pdg.Ref, state factoryState, descriptor *factoryDescriptor) bool {
	origins := a.origins(ref, 1)
	if !origins.Known || state.unknownExit {
		return false
	}
	for _, source := range origins.Sources {
		if a.query.emptySource(source) {
			continue
		}
		requirement, known := a.query.sourceRequirement(source, state)
		if !known {
			return false
		}
		errorNil, paired := a.errorRelation(ref, source.Node)
		descriptor.requirements = append(descriptor.requirements, factoryRequirement{target: requirement.target, primitive: requirement.primitive, isolated: requirement.isolated, safe: requirement.safe, errorNil: errorNil, paired: paired})
	}
	return len(origins.Sources) != 0
}

func joinFactorySet(target, source map[pdg.Ref]bool) bool {
	changed := false
	for ref := range source {
		if !target[ref] {
			target[ref] = true
			changed = true
		}
	}
	return changed
}
