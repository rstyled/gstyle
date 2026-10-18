package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
	"slices"
)

func reachableHeap(heap environmentHeap, roots []string) environmentHeap {
	result := environmentHeap{}
	for i := 0; i < len(roots); i++ {
		address := roots[i]
		if _, found := result[address]; found {
			continue
		}
		cell, found := heap[address]
		if !found {
			cell = environmentCell{Repeated: true}
		}
		result[address] = cell
		roots = append(roots, callableAddresses(callableSet(cell.Value))...)
	}
	return result
}
func tupleAddresses(tuple string) []string {
	var roots []string
	for _, value := range decodeTuple(tuple) {
		roots = append(roots, callableAddresses(callableSet(value))...)
	}
	return roots
}
func (evaluation *exitEvaluation) publishHeap(value completion, mode Outcomes) {
	index := completionIndex(mode)
	if index < 0 {
		return
	}
	roots := slices.Collect(maps.Keys(evaluation.incoming))
	if mode == Returns {
		roots = append(roots, tupleAddresses(evaluation.completedReturn(value))...)
	}
	roots = append(roots, tupleAddresses(evaluation.captureTuple(evaluation.completionPoint(value)))...)
	heap := reachableHeap(evaluation.values[value.values].heap, roots)
	evaluation.heapResults[index] = joinHeapEvidence(evaluation.heapResults[index], encodeHeap(heap))
}
func (calls *CallSites) contextHeap(summary ExitSummary, invocation exitInvocationContext, mode Outcomes) environmentHeap {
	index := completionIndex(mode)
	if index < 0 {
		return nil
	}
	which := 0
	if invocation.inherited {
		which = 1
	}
	key := calls.inputKey(summary.invocations, invocation)
	if key != "" {
		if state, found := summary.invocations.lookup(key); found {
			return decodeHeap(state.heaps[which][index])
		}
	}
	return decodeHeap(summary.heaps[which][index])
}

type environmentProjection struct {
	writes   map[pdg.Ref]exitValue
	fresh    map[string]bool
	returned string
	heap     environmentHeap
	known    bool
}

func (calls *CallSites) projectEnvironment(summaries ExitSummaries, invocation exitInvocationContext, mode Outcomes) environmentProjection {
	result := environmentProjection{heap: environmentHeap{}, fresh: map[string]bool{}, known: true}
	targets := calls.invocationTargets(invocation.ref, invocation.values)
	if !targets.known {
		return environmentProjection{}
	}
	var alternatives environmentAlternatives
	for _, target := range targets.callables() {
		invocation = calls.targetInvocation(summaries, invocation, target)
		outcomes := calls.targetWithValues(target.Name, summaries, invocation)
		result.known = result.known && outcomes&UnknownExit == 0
		if outcomes&mode == 0 {
			continue
		}
		next := calls.projectTarget(summaries, invocation, mode)
		result.merge(next)
		alternatives = append(alternatives, next)
	}
	result.completeUnknownResult()
	result.retainUntouched(alternatives, invocation.values)
	result.writes = joinCaptureWrites(alternatives.captureWrites(), invocation.values.Bindings)
	return result
}
func (projection *environmentProjection) merge(next environmentProjection) {
	projection.known = projection.known && next.known
	projection.returned = joinReturned(projection.returned, next.returned)
	projection.heap = joinHeaps(projection.heap, next.heap)
	maps.Copy(projection.fresh, next.fresh)
}
func (projection *environmentProjection) completeUnknownResult() {
	if !projection.known {
		projection.returned = unknownReturned(len(decodeTuple(projection.returned)))
	}
}
func (calls *CallSites) projectTarget(summaries ExitSummaries, invocation exitInvocationContext, mode Outcomes) environmentProjection {
	summary := summaries[invocation.target.Name]
	heap := calls.contextHeap(summary, invocation, mode)
	returned := ""
	complete := true
	if mode == Returns {
		var missing bool
		returned, missing = calls.targetReturned(invocation.target.Name, summaries, invocation)
		_, loaded := summaries[invocation.target.Name]
		complete = !missing || (loaded && summary.arity == 0)
	}
	entry := readEnvironmentInput(calls.inputKey(summary.invocations, invocation))
	writes := map[pdg.Ref]exitValue{}
	if calls.targetInvocationMemory(invocation.target.Name, summaries, invocation).confined {
		writes = calls.targetCaptureModeWrites(invocation.target.Name, summaries, captureProjection{invocation: invocation, outcomes: calls.targetWithValues(invocation.target.Name, summaries, invocation), mode: mode})
	}
	return rebaseEnvironment(environmentProjection{returned: returned, heap: heap, writes: writes, known: complete}, heapFromEntries(entry.Heap), calls.invocationSite(invocation.ref))
}
func (calls *CallSites) invocationSite(ref pdg.Ref) string {
	if calls.graph == nil || calls.graph.Tables == nil {
		return ""
	}
	return ownedJSON(struct {
		Owner, Revision string
		Ref             pdg.Ref
	}{calls.graph.Tables.Text(calls.graph.Function.ID), graphRevision(calls.graph), ref})
}
func rebaseEnvironment(projection environmentProjection, entry environmentHeap, site string) environmentProjection {
	if len(projection.heap) == 0 {
		return projection
	}
	renames, repeated := rebasedAddresses(projection.heap, entry, site)
	result := environmentProjection{returned: renameTuple(projection.returned, renames), heap: environmentHeap{}, writes: map[pdg.Ref]exitValue{}, fresh: map[string]bool{}, known: projection.known}
	for address, cell := range projection.heap {
		target := address
		if mapped := renames[address]; mapped != "" {
			target = mapped
			result.fresh[target] = true
		}
		cell.Value = renameValue(cell.Value, renames)
		cell.Repeated = cell.Repeated || repeated[address]
		result.heap.mergeAllocation(target, cell)
	}
	for symbol, value := range projection.writes {
		result.writes[symbol] = renameValue(value, renames)
	}
	return result
}
func rebasedAddresses(heap, entry environmentHeap, site string) (map[string]string, map[string]bool) {
	renames := map[string]string{}
	repeated := map[string]bool{}
	for address := range heap {
		if _, input := entry[address]; input {
			continue
		}
		next := readOwnedJSON[environmentAddress](address)
		repeated[address] = slices.Contains(next.Sites, site)
		next.Sites = append(next.Sites, site)
		slices.Sort(next.Sites)
		next.Sites = slices.Compact(next.Sites)
		renames[address] = ownedJSON(next)
	}
	return renames, repeated
}
func (heap environmentHeap) mergeAllocation(address string, cell environmentCell) {
	if old, found := heap[address]; found {
		cell.Value = joinValue(old.Value, cell.Value)
		cell.Repeated = true
	}
	heap[address] = cell
}
func renameValue(value exitValue, renames map[string]string) exitValue {
	if len(renames) == 0 {
		return value
	}
	set := callableSet(value)
	if !set.Known() {
		return value
	}
	targets := set.Targets()
	for i := range targets {
		for j, cell := range targets[i].Cells {
			if next := renames[cell.Address]; next != "" {
				targets[i].Cells[j].Address = next
			}
		}
	}
	return callableValue(bindingflow.NewBoundCallableSet(targets...))
}
func renameTuple(tuple string, renames map[string]string) string {
	if len(renames) == 0 {
		return tuple
	}
	values := decodeTuple(tuple)
	for i, value := range values {
		values[i] = renameValue(value, renames)
	}
	return encodeTuple(values)
}
