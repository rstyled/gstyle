// Package taintflow follows symbolic input dependencies over canonical PDGs.
// Queries borrow validated graphs; summaries contain no compiler objects.
package taintflow

import (
	"github.com/rstyled/gstyle/domain/callflow"
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
type ArgumentEncoding struct {
	Position pdg.Position
	SafeFor  string
}
type Models struct {
	// PreservesValueEncoding identifies expressions that retain encoded text.
	PreservesValueEncoding func(*pdg.Graph, pdg.Ref) bool
	// Encodings describe proven output alphabets for individual call arguments.
	Encodings func(*pdg.Graph, pdg.Ref) []ArgumentEncoding
	// PreservesEncoding identifies calls that copy encoded text without decoding it.
	PreservesEncoding func(*pdg.Graph, pdg.Ref) bool
	Source            func(*pdg.Graph, pdg.Ref) bool
	Sinks             func(*pdg.Graph, pdg.Ref) []Sink
}
type Finding struct {
	Source, Sink Site
	Kind         string
}
type atom struct {
	SafeFor string
	Input   int
	Fields  string
	Source  Site
}

// dependencies is a set; stored values are always true.
type dependencies map[atom]bool
type object struct {
	Captured   bool
	History    callflow.AllocationHistory
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
	Functions map[string]bool
	Opaque    bool
	Nil       bool
	Data      dependencies
	Objects   map[object]bool
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
	Repeated map[object]bool
	Results  map[pdg.Position]value
	Writes   map[cell]value
	Sinks    map[sinkKey]dependencies
}
type state struct {
	Repeated map[object]bool
	Bindings map[pdg.Ref]value
	Memory   map[cell]value
	Nodes    map[slot]value
}

func emptyValue() value {
	return value{Data: dependencies{}, Objects: map[object]bool{}, Functions: map[string]bool{}}
}
func emptyState() state {
	return state{Repeated: map[object]bool{}, Bindings: map[pdg.Ref]value{}, Memory: map[cell]value{}, Nodes: map[slot]value{}}
}
func emptySummary() summary {
	return summary{Repeated: map[object]bool{}, Results: map[pdg.Position]value{}, Writes: map[cell]value{}, Sinks: map[sinkKey]dependencies{}}
}
func joinValue(to, from value) value {
	if valueContains(to, from) {
		return to
	}
	if valueContains(from, to) {
		return from
	}
	result := emptyValue()
	result.Opaque = to.Opaque || from.Opaque
	result.Nil = to.Nil || from.Nil
	maps.Copy(result.Functions, to.Functions)
	maps.Copy(result.Functions, from.Functions)
	maps.Copy(result.Data, to.Data)
	maps.Copy(result.Data, from.Data)
	maps.Copy(result.Objects, to.Objects)
	maps.Copy(result.Objects, from.Objects)
	return result
}
func cloneState(old state) state {
	result := emptyState()
	maps.Copy(result.Repeated, old.Repeated)
	maps.Copy(result.Bindings, old.Bindings)
	maps.Copy(result.Memory, old.Memory)
	maps.Copy(result.Nodes, old.Nodes)
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
	return containsCallable(to, from) && setContains(to.Data, from.Data) && setContains(to.Objects, from.Objects)
}
func valueEqual(left, right value) bool {
	return left.Nil == right.Nil && left.Opaque == right.Opaque && maps.Equal(left.Functions, right.Functions) && maps.Equal(left.Data, right.Data) && maps.Equal(left.Objects, right.Objects)
}
func stateEqual(left, right state) bool {
	return maps.Equal(left.Repeated, right.Repeated) && maps.EqualFunc(left.Bindings, right.Bindings, valueEqual) && maps.EqualFunc(left.Memory, right.Memory, valueEqual) && maps.EqualFunc(left.Nodes, right.Nodes, valueEqual)
}
func summaryEqual(left, right summary) bool {
	return maps.Equal(left.Repeated, right.Repeated) && maps.EqualFunc(left.Results, right.Results, valueEqual) && maps.EqualFunc(left.Writes, right.Writes, valueEqual) && maps.EqualFunc(left.Sinks, right.Sinks, maps.Equal[dependencies, dependencies])
}

func containsCallable(to, from value) bool {
	return (!from.Nil || to.Nil) && (!from.Opaque || to.Opaque) && setContains(to.Functions, from.Functions)
}
