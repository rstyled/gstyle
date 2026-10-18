package callflow

import (
	"encoding/json"
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
	"slices"
)

func (operations *OperationEffects) indexInvocationOrder(ref pdg.Ref, value string) error {
	if operations.graph.Nodes[ref-1].Category != pdg.Call || operations.orders[ref] != 0 {
		return fmt.Errorf("invalid or duplicate invocation order owner")
	}
	order, err := decodeInvocationOrder(value)
	if err != nil {
		return err
	}
	for _, existing := range operations.orders {
		if existing == order {
			return fmt.Errorf("duplicate invocation ordinal")
		}
	}
	operations.orders[ref] = order
	return nil
}

func (operations *OperationEffects) orderedRegion(ref pdg.Ref) bool {
	if !operations.admittedInvocation(ref) {
		return false
	}
	for _, candidate := range operations.members[operations.regions[ref]] {
		if operations.graph.Nodes[candidate-1].Category == pdg.Call && operations.orders[candidate] == 0 {
			return false
		}
	}
	return true
}

func (operations *OperationEffects) mustBefore(candidate, call pdg.Ref, group uint32) bool {
	if operations.dependsOn(call, candidate, group, map[pdg.Ref]bool{}) {
		return true
	}
	for _, anchor := range operations.members[group] {
		if operations.orders[anchor] != 0 && operations.orders[anchor] < operations.orders[call] && operations.dependsOn(anchor, candidate, group, map[pdg.Ref]bool{}) {
			return true
		}
	}
	return false
}

func (operations *OperationEffects) mustAfter(candidate, call pdg.Ref, group uint32) bool {
	if operations.dependsOn(candidate, call, group, map[pdg.Ref]bool{}) {
		return true
	}
	for _, anchor := range operations.members[group] {
		if operations.orders[anchor] > operations.orders[call] && operations.dependsOn(candidate, anchor, group, map[pdg.Ref]bool{}) {
			return true
		}
	}
	return false
}

func (operations *OperationEffects) admittedInvocation(ref pdg.Ref) bool {
	if !operations.SameRegion(ref, ref) {
		return false
	}
	return operations.orders[ref] != 0 && slices.Contains(operations.members[operations.regions[ref]], ref)
}

func decodeInvocationOrder(value string) (uint32, error) {
	var order uint32
	if err := json.Unmarshal([]byte(value), &order); err != nil {
		return 0, err
	}
	if order == 0 {
		return 0, fmt.Errorf("missing invocation ordinal")
	}
	return order, nil
}
