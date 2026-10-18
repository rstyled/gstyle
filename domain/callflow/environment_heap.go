package callflow

import (
	"encoding/json"
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
	"slices"
	"unicode/utf8"
)

type environmentCell struct {
	Value    exitValue
	Repeated bool
}
type environmentHeap map[string]environmentCell
type environmentAddress struct {
	Owner, Revision string
	Symbol          pdg.Ref
	Sites           []string
}
type cellLink struct {
	Symbol  pdg.Ref
	Address string
}
type heapEntry struct {
	Address string
	Cell    environmentCell
}
type heapEffects [5]string

func ownedJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}
func readOwnedJSON[T any](value string) T {
	var result T
	if value != "" {
		if err := json.Unmarshal([]byte(value), &result); err != nil {
			panic(err)
		}
	}
	return result
}
func heapEntries(heap environmentHeap) []heapEntry {
	var entries []heapEntry
	for _, address := range slices.Sorted(maps.Keys(heap)) {
		entries = append(entries, heapEntry{address, heap[address]})
	}
	return entries
}
func heapFromEntries(entries []heapEntry) environmentHeap {
	heap := environmentHeap{}
	for _, entry := range entries {
		heap[entry.Address] = entry.Cell
	}
	return heap
}
func encodeHeap(heap environmentHeap) string {
	if len(heap) == 0 {
		return "null"
	}
	return ownedJSON(heapEntries(heap))
}
func decodeHeap(value string) environmentHeap {
	if value == "" || value == "null" {
		return environmentHeap{}
	}
	return heapFromEntries(readOwnedJSON[[]heapEntry](value))
}
func joinHeaps(a, b environmentHeap) environmentHeap {
	result := maps.Clone(a)
	if result == nil {
		result = environmentHeap{}
	}
	for address, cell := range b {
		if old, found := result[address]; found {
			cell.Value = joinValue(old.Value, cell.Value)
			cell.Repeated = old.Repeated || cell.Repeated
		}
		result[address] = cell
	}
	return result
}
func joinHeapEvidence(old, next string) string {
	if old == "" {
		return next
	}
	if next == "" {
		return old
	}
	return encodeHeap(joinHeaps(decodeHeap(old), decodeHeap(next)))
}
func (pass exitPass) heapEffects(old, next heapEffects) heapEffects {
	if pass != finitePass {
		return old
	}
	for i := range old {
		old[i] = joinHeapEvidence(old[i], next[i])
	}
	return old
}
func unknownHeap(heap environmentHeap) environmentHeap {
	result := maps.Clone(heap)
	for address, cell := range result {
		cell.Value = unknownValue
		result[address] = cell
	}
	return result
}
func writeEnvironmentCell(heap environmentHeap, address string, value exitValue) {
	cell := heap[address]
	if cell.Repeated {
		value = joinValue(cell.Value, value)
	}
	cell.Value = value
	heap[address] = cell
}
func graphRevision(graph *pdg.Graph) string {
	if graph == nil || graph.Tables == nil || graph.Function.Span.Source == 0 {
		return ""
	}
	source := graph.Sources[graph.Function.Span.Source-1]
	if source.ContentIdentity.Status != pdg.Known {
		return ""
	}
	value := graph.Tables.Text(source.ContentIdentity.Value)
	if !utf8.ValidString(value) {
		return ""
	}
	return value
}
func (targets callTargets) callables() []bindingflow.Callable {
	if len(targets.bound) != 0 {
		return targets.bound
	}
	return bindingflow.NewCallableSet(targets.names...).Targets()
}

func cloneEnvironmentHeap(heap environmentHeap) environmentHeap { return maps.Clone(heap) }
