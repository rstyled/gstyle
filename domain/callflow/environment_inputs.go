package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
	"slices"
)

type environmentInput struct {
	Values   []exitValue
	Heap     []heapEntry
	Captures []cellLink
}

func readEnvironmentInput(key string) environmentInput {
	if key == "" {
		return environmentInput{}
	}
	if key[0] == '[' {
		return environmentInput{Values: decodeTuple(key)}
	}
	return readOwnedJSON[environmentInput](key)
}
func (calls *CallSites) boundCaptures(invocations *exitInvocations, invocation exitInvocationContext) map[pdg.Ref]string {
	if !matchingEnvironment(invocations, invocation.target) {
		return nil
	}
	result := map[pdg.Ref]string{}
	for _, cell := range invocation.target.Cells {
		if !validBoundCell(invocations, cell, invocation.values) {
			return nil
		}
		result[cell.Symbol] = cell.Address
	}
	return result
}
func matchingEnvironment(invocations *exitInvocations, target bindingflow.Callable) bool {
	return len(target.Cells) > 0 && len(target.Cells) == len(invocations.captures) && target.Name == invocations.graph.Tables.Text(invocations.graph.Function.ID) && target.Revision == graphRevision(invocations.graph)
}
func validBoundCell(invocations *exitInvocations, cell bindingflow.CallableCell, context bindingflow.Evaluation) bool {
	_, present := context.ClosureCells[cell.Address]
	return present && slices.Contains(invocations.captures, cell.Symbol)
}
func (calls *CallSites) environmentKey(invocations *exitInvocations, invocation exitInvocationContext) string {
	values, known := calls.paths.CallValues(invocation.ref, invocations.inputs, invocation.values)
	if !known {
		return ""
	}
	input := calls.environmentArguments(invocations, invocation, values)
	if len(input.Heap)+len(input.Captures) != 0 {
		return ownedJSON(input)
	}
	if !knownTuple(input.Values) {
		return ""
	}
	return encodeTuple(input.Values)
}
func (calls *CallSites) environmentArguments(invocations *exitInvocations, invocation exitInvocationContext, values []bindingflow.Value) environmentInput {
	slots := make([]exitValue, len(invocations.inputs)+len(invocations.captures))
	copy(slots, decodeTuple(calls.completeInputKey(invocations, values, invocation.values)))
	links := calls.boundCaptures(invocations, invocation)
	for i, symbol := range invocations.captures {
		if address := links[symbol]; address != "" {
			slots[len(invocations.inputs)+i] = abstractValue(invocation.values.ClosureCells[address])
		}
	}
	captures := locationEntries(links)
	roots := argumentEnvironmentRoots(values, captures)
	return environmentInput{Values: slots, Heap: heapEntries(reachableContextHeap(invocation.values, roots)), Captures: captures}
}
func argumentEnvironmentRoots(values []bindingflow.Value, captures []cellLink) []string {
	var roots []string
	for _, value := range values {
		roots = append(roots, callableAddresses(value.Callables)...)
	}
	for _, link := range captures {
		roots = append(roots, link.Address)
	}
	return roots
}
func locationEntries(locations map[pdg.Ref]string) []cellLink {
	var entries []cellLink
	for _, symbol := range slices.Sorted(maps.Keys(locations)) {
		entries = append(entries, cellLink{symbol, locations[symbol]})
	}
	return entries
}
func callableAddresses(set bindingflow.CallableSet) []string {
	var addresses []string
	for _, target := range set.Targets() {
		for _, cell := range target.Cells {
			addresses = append(addresses, cell.Address)
		}
	}
	return addresses
}
func reachableContextHeap(context bindingflow.Evaluation, roots []string) environmentHeap {
	heap := environmentHeap{}
	for i := 0; i < len(roots); i++ {
		address := roots[i]
		if _, found := heap[address]; found {
			continue
		}
		value := context.ClosureCells[address]
		heap[address] = environmentCell{Value: abstractValue(value), Repeated: context.RepeatedCells[address]}
		roots = append(roots, callableAddresses(value.Callables)...)
	}
	return heap
}

func (calls *CallSites) targetInvocation(summaries ExitSummaries, invocation exitInvocationContext, target bindingflow.Callable) exitInvocationContext {
	invocation.target = target
	invocation.keyReady = false
	invocation.key = calls.inputKey(summaries[target.Name].invocations, invocation)
	invocation.keyReady = true
	return invocation
}

func (calls *CallSites) validTargetEnvironment(summary ExitSummary, invocation exitInvocationContext) bool {
	if len(invocation.target.Cells) == 0 {
		return true
	}
	if summary.invocations == nil {
		return false
	}
	return calls.boundCaptures(summary.invocations, invocation) != nil
}
