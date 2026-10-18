package callflow

import (
	"encoding/json"
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
)

// PriorOperations returns non-call effects that can precede the selected call
// within its operand region. Direct canonical operand dependencies establish
// ordering; a source traversal order does not. Results own their slice.
func (operations *OperationEffects) PriorOperations(ref pdg.Ref) ([]pdg.Ref, bool) {
	if !operations.orderedRegion(ref) {
		return nil, false
	}
	group := operations.regions[ref]
	var candidates []pdg.Ref
	for _, candidate := range operations.members[group] {
		if operations.couldPreempt(candidate, ref, group) {
			candidates = append(candidates, candidate)
		}
	}
	return candidates, true
}

// SameRegion identifies one admitted operand-evaluation region.
func (operations *OperationEffects) SameRegion(left, right pdg.Ref) bool {
	if operations == nil || operations.incompleteRegions {
		return false
	}
	group := operations.regions[left]
	return group != 0 && group == operations.regions[right]
}

// LaterOperations retains effects that can follow a call's normal return.
// Required operand producers cannot move after the call that consumes them.
func (operations *OperationEffects) LaterOperations(ref pdg.Ref) ([]pdg.Ref, bool) {
	if !operations.orderedRegion(ref) {
		return nil, false
	}
	group := operations.regions[ref]
	var candidates []pdg.Ref
	for _, candidate := range operations.members[group] {
		if candidate == ref {
			continue
		}
		if operations.mustBefore(candidate, ref, group) {
			continue
		}
		candidates = append(candidates, candidate)
	}
	return candidates, true
}

func (operations *OperationEffects) indexRegion(ref pdg.Ref, value string) error {
	if operations.regions[ref] != 0 {
		return fmt.Errorf("duplicate evaluation region")
	}
	var group uint32
	if err := json.Unmarshal([]byte(value), &group); err != nil {
		return err
	}
	if group == 0 {
		return fmt.Errorf("missing evaluation region identity")
	}
	operations.regions[ref] = group
	return nil
}

func (operations *OperationEffects) indexRegionMembers() {
	flow := pdg.ExitBodyFlow(operations.graph)
	if flow == nil {
		operations.incompleteRegions = true
		return
	}
	for _, block := range flow.Blocks {
		if !block.Reachable {
			continue
		}
		for _, ref := range block.Operations {
			operations.regionMember(ref)

		}
	}
}

func (operations *OperationEffects) couldPreempt(candidate, call pdg.Ref, group uint32) bool {
	if candidate == call || operations.graph.Nodes[candidate-1].Category == pdg.Call {
		return false
	}
	if operations.Outcomes(candidate)&^Returns == 0 {
		return false
	}
	return !operations.mustAfter(candidate, call, group)
}

func (operations *OperationEffects) dependsOn(ref, target pdg.Ref, group uint32, seen map[pdg.Ref]bool) bool {
	if ref == target {
		return true
	}
	if operations.regions[ref] != group || seen[ref] {
		return false
	}
	seen[ref] = true
	return operations.inputDependsOn(ref, target, group, seen) || operations.guardDependsOn(ref, target, group, seen)
}

func (operations *OperationEffects) inputDependsOn(ref, target pdg.Ref, group uint32, seen map[pdg.Ref]bool) bool {
	positions, known := operations.paths.RequiredOperands(ref)
	if !known {
		return false
	}
	for _, position := range positions {
		if operations.dependsOn(operations.paths.Operand(ref, position), target, group, seen) {
			return true
		}
	}
	return false
}

func (operations *OperationEffects) regionMember(ref pdg.Ref) {
	group := operations.regions[ref]
	if group != 0 {
		operations.members[group] = append(operations.members[group], ref)
		return
	}
	if operations.graph.Nodes[ref-1].Category == pdg.Call || operations.Outcomes(ref)&^Returns != 0 {
		operations.incompleteRegions = true
	}
}
