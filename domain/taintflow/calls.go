package taintflow

import "github.com/rstyled/gstyle/domain/pdg"

func (e *evaluation) call(ref pdg.Ref) {
	e.directSinks(ref)
	if e.query.models.Source != nil && e.query.models.Source(e.query.graph, ref) {
		e.sourceCall(ref)
		return
	}
	target := e.target(ref)
	if callee := e.catalog[target]; callee != nil {
		e.localCall(ref, callee, e.summaries[target])
		return
	}
	e.store(slot{Node: ref}, e.externalCall(ref))
}
func (e *evaluation) target(ref pdg.Ref) string {
	attrs := e.query.graph.NodeAttributes(e.query.graph.Nodes[ref-1])
	if attrs.CallForm != pdg.DirectCall || attrs.Callee.Status != pdg.Known {
		return ""
	}
	return e.query.graph.Tables.Text(attrs.Callee.Value)
}
func (e *evaluation) sourceCall(ref pdg.Ref) {
	item := emptyValue()
	item.Data[atom{Source: Site{Function: e.query.name, Node: ref}}] = true
	e.store(slot{Node: ref}, item)
	e.store(slot{Node: ref, Result: 1}, item)
}
func (e *evaluation) localCall(ref pdg.Ref, callee *query, summary summary) {
	for index := range callee.graph.Function.Outputs {
		position := pdg.Position(index + 1)
		e.store(slot{Node: ref, Result: position}, e.substitute(summary.Results[position], ref))
	}
	e.store(slot{Node: ref}, e.state.Nodes[slot{Node: ref, Result: 1}])
	for sink, deps := range summary.Sinks {
		e.inheritedSink(ref, sink, deps)
	}
	e.callWrites(ref, summary)
}
func (e *evaluation) inheritedSink(call pdg.Ref, sink sinkKey, deps dependencies) {
	translated := e.substitute(value{Data: deps}, call)
	if e.result.Sinks[sink] == nil {
		e.result.Sinks[sink] = dependencies{}
	}
	joinSinkDependencies(e.result.Sinks[sink], translated.Data, sink.Kind)
}
func (e *evaluation) directSinks(ref pdg.Ref) {
	if e.query.models.Sinks == nil {
		return
	}
	for _, sink := range e.query.models.Sinks(e.query.graph, ref) {
		e.directSink(ref, sink)
	}
}
func (e *evaluation) directSink(ref pdg.Ref, sink Sink) {
	key := sinkKey{Site: Site{Function: e.query.name, Node: ref}, Kind: sink.Kind}
	if e.result.Sinks[key] == nil {
		e.result.Sinks[key] = dependencies{}
	}
	for _, position := range sink.Positions {
		joinSinkDependencies(e.result.Sinks[key], e.input(ref, position).Data, sink.Kind)
	}
}
