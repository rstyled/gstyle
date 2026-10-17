package gopatterns

import (
	"maps"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/cfg"
)

const cyclicDefinitionDescription = "Least-fixed-point reaching definitions for local scalar normal flow, including loops, branch transfers and switches. Writes kill previous definitions; predecessor joins retain every reaching source definition. Calls, memory, closures, deferred execution and concurrency are excluded."

// Solve with original source definitions only. Creating merge definitions while
// iterating would grow the domain and make a loop invent unbounded new values.
// The finite definition domain and monotone kill/gen transfer ensure convergence.
func (a *reachingAnalysis) fixedPoint(flow *cfg.CFG, dom *executionDominators) {
	queue := []int{0}
	pending := make([]bool, len(flow.Blocks))
	pending[0] = true
	for len(queue) != 0 {
		index := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		pending[index] = false
		if !a.update(flow.Blocks[index]) {
			continue
		}
		for _, next := range flow.Blocks[index].Succs {
			if pending[next.Index] {
				continue
			}
			pending[next.Index] = true
			queue = append(queue, int(next.Index))
		}
	}
	// Publish each source read once, after every incoming definition set is stable.
	for _, index := range dom.order {
		a.block(flow.Blocks[index], a.fixedInput(index))
	}
}

func (a *reachingAnalysis) fixedInput(index int) reachingState {
	state := reachingState{}
	if index == 0 {
		a.seed(state)
	}
	for _, predecessor := range a.predecessors[index] {
		state.merge(a.outputs[predecessor])
	}
	return state
}

func (a *reachingAnalysis) transfer(block *cfg.Block, state reachingState) {
	for _, source := range block.Nodes {
		node := a.builder.nodes[source]
		if node == 0 {
			continue
		}
		attributes := a.builder.attributes[a.builder.graph.Nodes[node-1].Attributes-1]
		for _, ref := range attributes.Defines {
			a.writeDefinition(ref, state)
		}
	}
}

func (a *reachingAnalysis) update(block *cfg.Block) bool {
	index := int(block.Index)
	state := a.fixedInput(index)
	a.transfer(block, state)
	if a.outputs[index] != nil && maps.EqualFunc(a.outputs[index], state, slices.Equal[[]pdg.Ref]) {
		return false
	}
	a.outputs[index] = state
	return true
}
