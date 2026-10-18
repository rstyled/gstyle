package callflow

import (
	"hash/maphash"
	"maps"
)

// State IDs belong to one evaluation. Zero denotes the empty state.
type valueStateID uint64

type valueStateIndex struct {
	seed    maphash.Seed
	buckets map[uint64]valueStateID
	links   map[valueStateID]valueStateID
}

func (evaluation *exitEvaluation) internValues(values exitValues) valueStateID {
	if len(values.heap)+len(values.locations)+len(values.bindings)+len(values.nodes)+len(values.tuples)+len(values.cells)+len(values.unorderedReads) == 0 {
		return 0
	}
	if evaluation.stateIndex.buckets == nil {
		evaluation.stateIndex = valueStateIndex{seed: maphash.MakeSeed(), buckets: map[uint64]valueStateID{}, links: map[valueStateID]valueStateID{}}
	}
	return evaluation.internValueState(values, values.stateHash(evaluation.stateIndex.seed))
}

// Values are copy-on-write snapshots. Hash collisions always use exact equality.
func (evaluation *exitEvaluation) internValueState(values exitValues, hash uint64) valueStateID {
	prior := evaluation.stateIndex.buckets[hash]
	for id := prior; id != 0; id = evaluation.stateIndex.links[id] {
		if values.equalState(evaluation.values[id]) {
			return id
		}
	}
	id := valueStateID(uint64(len(evaluation.values)) + 1)
	evaluation.values[id] = values
	if prior != 0 {
		evaluation.stateIndex.links[id] = prior
	}
	evaluation.stateIndex.buckets[hash] = id
	return id
}

func (values exitValues) stateHash(seed maphash.Seed) uint64 {
	parts := [7]uint64{
		stateMapHash(seed, values.heap), stateMapHash(seed, values.locations),
		stateMapHash(seed, values.bindings), stateMapHash(seed, values.nodes),
		stateMapHash(seed, values.cells), stateMapHash(seed, values.tuples),
		stateMapHash(seed, values.unorderedReads),
	}
	return maphash.Comparable(seed, parts)
}

func stateMapHash[K comparable, V comparable](seed maphash.Seed, values map[K]V) uint64 {
	hash := uint64(len(values))
	for key, value := range values {
		hash ^= maphash.Comparable(seed, struct {
			Key   K
			Value V
		}{key, value})
	}
	return hash
}

func (values exitValues) equalState(other exitValues) bool {
	return values.equalScalars(other) && values.equalMemory(other) && maps.Equal(values.unorderedReads, other.unorderedReads)
}

func (values exitValues) equalScalars(other exitValues) bool {
	return maps.Equal(values.bindings, other.bindings) && maps.Equal(values.nodes, other.nodes) && maps.Equal(values.tuples, other.tuples)
}

func (values exitValues) equalMemory(other exitValues) bool {
	return maps.Equal(values.heap, other.heap) && maps.Equal(values.locations, other.locations) && maps.Equal(values.cells, other.cells)
}
