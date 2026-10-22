package gopatterns

import (
	"go/types"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

type assertionRequirement struct {
	path bindingflow.Path
	// A nil type requests only a non-nil interface proof.
	typ types.Type
}

type assertionEdge struct {
	block   pdg.Ref
	outcome int
}

type assertionPoint struct {
	block  pdg.Ref
	offset int
}

func assertionPredecessors(flow *pdg.NormalFlow) [][]assertionEdge {
	previous := make([][]assertionEdge, len(flow.Blocks))
	for index, block := range flow.Blocks {
		if !block.Reachable {
			continue
		}
		for outcome, next := range block.Successors {
			previous[next-1] = append(previous[next-1], assertionEdge{pdg.Ref(index + 1), outcome})
		}
	}
	return previous
}

// Walk backwards until every incoming path crosses a successful matching
// guard. A write, unguarded entry, absent operation or budget exhaustion fails
// the proof. Cycles are visited once; this does not claim loop termination.
func (q *assertionGuardQuery) protected(ref pdg.Ref, required assertionRequirement) bool {
	start, found := q.assertionPoint(ref)
	if !found {
		return false
	}
	pending := []assertionPoint{start}
	seen := map[assertionPoint]bool{}
	for budget := 16384; len(pending) > 0 && budget > 0; budget-- {
		point := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[point] {
			continue
		}
		seen[point] = true
		next, valid := q.priorPoints(point, required)
		if !valid {
			return false
		}
		pending = append(pending, next...)
	}
	return len(pending) == 0
}

func (q *assertionGuardQuery) assertionPoint(ref pdg.Ref) (assertionPoint, bool) {
	for index, block := range q.flow.Blocks {
		if !block.Reachable {
			continue
		}
		for offset, operation := range block.Operations {
			if operation == ref {
				return assertionPoint{pdg.Ref(index + 1), offset}, true
			}
		}
	}
	return assertionPoint{}, false
}

func (q *assertionGuardQuery) priorPoints(point assertionPoint, required assertionRequirement) ([]assertionPoint, bool) {
	block := q.flow.Blocks[point.block-1]
	proved, valid := q.priorAssertion(block.Operations[:point.offset], required)
	if proved {
		return nil, true
	}
	if !valid || point.block == q.flow.Entry {
		return nil, false
	}
	var previous []assertionPoint
	for _, edge := range q.previous[point.block-1] {
		if q.guards(edge, required) {
			continue
		}
		previous = append(previous, assertionPoint{edge.block, len(q.flow.Blocks[edge.block-1].Operations)})
	}
	return previous, len(q.previous[point.block-1]) > 0
}

// A prior single-result assertion must succeed for normal execution to reach
// this point. Scan in reverse so an intervening write cancels that proof.
func (q *assertionGuardQuery) priorAssertion(operations []pdg.Ref, required assertionRequirement) (bool, bool) {
	for index := len(operations) - 1; index >= 0; index-- {
		ref := operations[index]
		if q.writesAssertionBinding(ref, required.path.Root) {
			return false, false
		}
		if q.successfulAssertion(ref, required) {
			return true, true
		}
	}
	return false, true
}

func (q *assertionGuardQuery) writesAssertionBinding(ref, symbol pdg.Ref) bool {
	for _, write := range q.graph.NodeAttributes(q.graph.Nodes[ref-1]).Writes {
		if write.Symbol == symbol {
			return true
		}
	}
	return false
}
