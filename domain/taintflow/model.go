// Package taintflow follows symbolic input dependencies over canonical PDGs.
// Queries borrow validated graphs; summaries contain no compiler objects.
package taintflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
)

type Site struct {
	Function string
	Node     pdg.Ref
}
type Sink struct {
	Kind      string
	Positions []pdg.Position
}
type Models struct {
	Source func(*pdg.Graph, pdg.Ref) bool
	Sinks  func(*pdg.Graph, pdg.Ref) []Sink
}
type Finding struct {
	Source, Sink Site
	Kind         string
}
type atom struct {
	Input  int
	Fields string
	Source Site
}
type dependencies map[atom]bool
type object struct {
	Input      int
	Node       pdg.Ref
	Allocation string
	Fields     string
}
type cell struct {
	Object object
	Fields string
}
type value struct {
	Data    dependencies
	Objects map[object]bool
}
type slot struct {
	Node   pdg.Ref
	Result pdg.Position
}
type sinkKey struct {
	Site Site
	Kind string
}
type summary struct {
	Results map[pdg.Position]value
	Writes  map[cell]value
	Sinks   map[sinkKey]dependencies
}
type state struct {
	Bindings map[pdg.Ref]value
	Memory   map[cell]value
	Nodes    map[slot]value
}

func emptyValue() value { return value{Data: dependencies{}, Objects: map[object]bool{}} }
func emptyState() state {
	return state{Bindings: map[pdg.Ref]value{}, Memory: map[cell]value{}, Nodes: map[slot]value{}}
}
func emptySummary() summary {
	return summary{Results: map[pdg.Position]value{}, Writes: map[cell]value{}, Sinks: map[sinkKey]dependencies{}}
}
func joinDependencies(to, from dependencies) {
	for item := range from {
		to[item] = true
	}
}
func joinValue(to, from value) value {
	if valueContains(to, from) {
		return to
	}
	if valueContains(from, to) {
		return from
	}
	result := emptyValue()
	joinDependencies(result.Data, to.Data)
	joinDependencies(result.Data, from.Data)
	for item := range to.Objects {
		result.Objects[item] = true
	}
	for item := range from.Objects {
		result.Objects[item] = true
	}
	return result
}
func cloneState(old state) state {
	result := emptyState()
	for key, item := range old.Bindings {
		result.Bindings[key] = item
	}
	for key, item := range old.Memory {
		result.Memory[key] = item
	}
	for key, item := range old.Nodes {
		result.Nodes[key] = item
	}
	return result
}

func setContains[K comparable](to, from map[K]bool) bool {
	if len(from) > len(to) {
		return false
	}
	for item := range from {
		if !to[item] {
			return false
		}
	}
	return true
}
func valueContains(to, from value) bool {
	return setContains(to.Data, from.Data) && setContains(to.Objects, from.Objects)
}
func valueEqual(left, right value) bool {
	return maps.Equal(left.Data, right.Data) && maps.Equal(left.Objects, right.Objects)
}
func stateEqual(left, right state) bool {
	return maps.EqualFunc(left.Bindings, right.Bindings, valueEqual) && maps.EqualFunc(left.Memory, right.Memory, valueEqual) && maps.EqualFunc(left.Nodes, right.Nodes, valueEqual)
}
func summaryEqual(left, right summary) bool {
	return maps.EqualFunc(left.Results, right.Results, valueEqual) && maps.EqualFunc(left.Writes, right.Writes, valueEqual) && maps.EqualFunc(left.Sinks, right.Sinks, maps.Equal[dependencies, dependencies])
}
