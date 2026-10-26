package taintflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
)

func (e *evaluation) uncachedInvokeKnown(ref pdg.Ref, callee *query) {
	arguments := e.callArguments(ref, callee)
	specialized := e.solver.specialize(callee, e.functionArguments(callee, arguments))
	e.solver.callers[specialized.key] = appendUnique(e.solver.callers[specialized.key], e.query.key)
	e.arguments = arguments
	e.uncachedLocalCall(ref, specialized, e.summaries[specialized.key])
	e.arguments = nil
}

func (e *evaluation) uncachedLocalCall(ref pdg.Ref, callee *query, summary summary) {
	for index := range callee.graph.Function.Outputs {
		position := pdg.Position(index + 1)
		e.store(slot{Node: ref, Result: position}, e.uncachedSubstitute(summary.Results[position], ref))
	}
	e.store(slot{Node: ref}, e.state.Nodes[slot{Node: ref, Result: 1}])
	for sink, deps := range summary.Sinks {
		e.uncachedInheritedSink(ref, sink, deps)
	}
	e.uncachedCallWrites(ref, summary)
}

func (e *evaluation) uncachedInheritedSink(call pdg.Ref, sink sinkKey, deps dependencies) {
	translated := e.uncachedSubstitute(value{Data: deps}, call)
	if e.result.Sinks[sink] == nil {
		e.result.Sinks[sink] = dependencies{}
	}
	joinSinkDependencies(e.result.Sinks[sink], translated.Data, sink.Kind)
}

func (e *evaluation) uncachedSubstitute(item value, call pdg.Ref) value {
	result := emptyValue()
	result.Functions, result.Opaque, result.Nil = item.Functions, item.Opaque, item.Nil
	for dep := range item.Data {
		translated := e.uncachedSubstituteAtom(dep, call)
		if dep.SafeFor != "" {
			translated = encodeValue(translated, dep.SafeFor)
		}
		maps.Copy(result.Data, translated.Data)
	}
	for root := range item.Objects {
		e.uncachedSubstituteObject(result.Objects, root, call)
	}
	return result
}

func (e *evaluation) uncachedSubstituteAtom(dep atom, call pdg.Ref) value {
	if dep.Input == 0 {
		return value{Data: dependencies{dep: true}, Objects: map[object]bool{}}
	}
	actual := e.actual(call, pdg.Position(dep.Input))
	if dep.Fields == "" {
		return actual
	}
	return pathValue(e.state, actual, dep.Fields)
}

func (e *evaluation) uncachedSubstituteObject(target map[object]bool, root object, call pdg.Ref) {
	if root.Input == 0 {
		root = e.rebaseObject(root, call)
		target[root] = true
		return
	}
	actual := e.actual(call, pdg.Position(root.Input))
	for item := range pathValue(e.state, actual, root.Fields).Objects {
		target[item] = true
	}
}

func (e *evaluation) uncachedCallWrites(call pdg.Ref, summary summary) {
	projected := map[cell]value{}
	for root := range summary.Repeated {
		targets := map[object]bool{}
		e.uncachedSubstituteObject(targets, root, call)
		for target := range targets {
			e.state.Repeated[target] = true
		}
	}
	for key, item := range summary.Writes {
		e.uncachedProjectWrite(call, key, item, projected)
	}
	for key, item := range projected {
		e.state.Memory[key] = item
	}
}

func (e *evaluation) uncachedProjectWrite(call pdg.Ref, key cell, item value, projected map[cell]value) {
	targets := e.projectTargets(call, key)
	translated := e.uncachedSubstitute(item, call)
	for _, target := range targets {
		if key.Object.Input == 0 {
			if _, found := projected[target]; found {
				e.state.Repeated[target.Object] = true
			}
		}
		next := translated
		if e.weakTarget(key, target, targets) {
			next = joinValue(readCell(e.state, target), next)
		}
		projected[target] = joinValue(projected[target], next)
	}
}
