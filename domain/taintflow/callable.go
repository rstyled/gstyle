package taintflow

import (
	"encoding/json"
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
	"slices"
)

func (e *evaluation) functionValue(ref pdg.Ref) (value, bool) {
	name, err := bindingflow.CallableIdentity(e.query.graph, ref)
	if err != nil {
		return value{}, false
	}
	item := e.closureValue(ref, name)
	item.Functions[name] = true
	return item, true
}
func callableFact(item value) bindingflow.Value {
	if item.Opaque {
		return bindingflow.Value{}
	}
	if item.Nil {
		return nilCallableFact(item)
	}
	if len(item.Functions) == 0 {
		return bindingflow.Value{}
	}
	return bindingflow.Value{NonNil: true, Callables: bindingflow.NewCallableSet(slices.Collect(maps.Keys(item.Functions))...)}
}
func nilCallableFact(item value) bindingflow.Value {
	if len(item.Functions) != 0 {
		return bindingflow.Value{}
	}
	return bindingflow.Value{Nil: true}
}
func (e *evaluation) callContext() bindingflow.Evaluation {
	context := bindingflow.Evaluation{Facts: e.query.facts, Bindings: map[pdg.Ref]bindingflow.Value{}, Nodes: map[pdg.Ref]bindingflow.Value{}}
	for symbol := range e.state.Bindings {
		context.Bindings[symbol] = callableFact(e.symbolValue(symbol))
	}
	for key, item := range e.state.Nodes {
		if key.Result == 0 {
			context.Nodes[key.Node] = callableFact(item)
		}
	}
	return context
}
func (e *evaluation) invoke(ref pdg.Ref) {
	targets, known := e.query.calls.InvocationTargets(ref, e.callContext())
	known = known && !e.targetValue(ref).Opaque
	if known && len(targets) == 0 {
		e.store(slot{Node: ref}, emptyValue())
		return
	}
	e.invokeAlternatives(ref, targets, known)
}

// Each possible target starts from the same caller state. Joining happens only
// after its effects and return value have been projected into the caller.
func (e *evaluation) invokeAlternatives(ref pdg.Ref, targets []bindingflow.Callable, known bool) {
	e.havocAddressArguments(ref)
	before := cloneState(e.state)
	var joined state
	for _, target := range targets {
		callee := e.catalog[target.Name]
		if callee == nil {
			known = false
			continue
		}
		e.state = cloneState(before)
		e.invokeKnown(ref, callee)
		joined = joinAlternative(joined, e.state)
	}
	if !known {
		e.state = before
		item := e.unknownCall(ref)
		item.Opaque = true
		e.store(slot{Node: ref}, item)
		joined = joinAlternative(joined, e.state)
	}
	e.state = joined
}
func joinAlternative(joined, next state) state {
	if joined.Bindings == nil {
		return next
	}
	return joinState(joined, next)
}
func (e *evaluation) invokeKnown(ref pdg.Ref, callee *query) {
	arguments := e.callArguments(ref, callee)
	specialized := e.solver.specialize(callee, e.functionArguments(callee, arguments))
	e.solver.callers[specialized.key] = appendUnique(e.solver.callers[specialized.key], e.query.key)
	e.arguments = arguments
	e.localCall(ref, specialized, e.summaries[specialized.key])
	e.arguments = nil
}
func appendUnique(items []string, item string) []string {
	if slices.Contains(items, item) {
		return items
	}
	return append(items, item)
}
func (e *evaluation) callArguments(ref pdg.Ref, callee *query) map[pdg.Position]value {
	result := map[pdg.Position]value{}
	for index := range callee.graph.Function.Inputs {
		position := pdg.Position(index + 1)
		result[position] = e.formalArgument(ref, callee, position)
	}
	e.captureArguments(ref, callee, result)
	return result
}
func (e *evaluation) formalArgument(ref pdg.Ref, callee *query, position pdg.Position) value {
	if !variadicPosition(callee, position) {
		return e.input(ref, position)
	}
	result := emptyValue()
	positions, _ := e.query.paths.RequiredOperands(ref)
	for _, actual := range positions {
		if actual >= position {
			result = joinValue(result, e.input(ref, actual))
		}
	}
	return result
}
func (solver *summarySolver) specialize(callee *query, arguments map[pdg.Position]value) *query {
	functions := map[int][]string{}
	for position, item := range arguments {
		if item.Opaque || len(item.Functions) == 0 {
			continue
		}
		functions[int(position)] = slices.Sorted(maps.Keys(item.Functions))
	}
	if len(functions) == 0 {
		return callee
	}
	data, _ := json.Marshal(functions)
	key := callee.name + "\n" + string(data)
	if found := solver.queries[key]; found != nil {
		return found
	}
	next := *callee
	next.key, next.functions = key, functions
	solver.queries[key] = &next
	solver.summaries[key] = emptySummary()
	solver.enqueue(key)
	return &next
}

// Identity conversions use the shared value query, including named function
// types and parentheses. Other conversions cannot preserve a callable proof.
func (e *evaluation) callableConversion(ref pdg.Ref, item value) (value, bool) {
	if !e.callableCopy(ref) {
		return value{}, false
	}
	context := e.callContext()
	delete(context.Nodes, ref)
	fact := e.query.paths.Evaluate(ref, context)
	if fact.Callables.Known() {
		item.Functions = map[string]bool{}
		for _, name := range fact.Callables.Names() {
			item.Functions[name] = true
		}
		item.Opaque = false
		return item, true
	}
	if fact.Nil {
		item.Nil = true
		item.Opaque = false
		return item, true
	}
	return value{}, false
}

func (e *evaluation) callableCopy(ref pdg.Ref) bool {
	if e.query.facts.Nodes[ref].Conversion == bindingflow.IdentityConversion {
		return true
	}
	attrs := e.query.graph.NodeAttributes(e.query.graph.Nodes[ref-1])
	return e.query.graph.Tables.Text(attrs.Operator.Value) == "parentheses"
}
