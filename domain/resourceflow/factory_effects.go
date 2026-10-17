package resourceflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (a *factoryAnalysis) effect(ref pdg.Ref, state *factoryState) {
	q := a.query
	node := q.graph.Nodes[ref-1]
	attrs := q.graph.NodeAttributes(node)
	if q.graph.Tables.Text(attrs.Construct) == "closure" {
		a.captureEffect(attrs, state)
		return
	}
	if q.graph.Tables.Text(attrs.Operator.Value) == "&" {
		a.addressEffect(ref, state)
		return
	}
	if site, found := q.machine.Site(ref); found {
		a.deferEffect(site.Invocation, site.Captures, state)
		return
	}
	if node.Category == pdg.Call {
		a.callEffect(ref, state)
		return
	}
	a.localEffect(ref, node.Category, attrs, state)
}

func (a *factoryAnalysis) localEffect(ref pdg.Ref, category pdg.Category, attrs pdg.Attributes, state *factoryState) {
	switch category {
	case pdg.Assignment, pdg.Declaration:
		a.storeEffect(ref, attrs, state)
	case pdg.Computation:
		if a.query.graph.Tables.Text(attrs.Operator.Value) == "composite" {
			a.inputsEffect(ref, state)
		}
	}
}

func (a *factoryAnalysis) callEffect(ref pdg.Ref, state *factoryState) {
	if !a.query.deferredReturns(ref) {
		a.invalidateExternal(state)
		a.inputsEffect(ref, state)
	}
	state.issued[ref] = true
}

func (a *factoryAnalysis) inputsEffect(ref pdg.Ref, state *factoryState) {
	positions, known := a.query.paths.RequiredOperands(ref)
	if !known {
		invalidateIssued(state)
		return
	}
	for _, position := range positions {
		a.invalidate(a.origins(ref, position), state)
	}
}

func (a *factoryAnalysis) invalidate(origins bindingflow.Origins, state *factoryState) {
	if !origins.Known {
		invalidateIssued(state)
	}
	for _, source := range origins.Sources {
		if a.query.graph.Nodes[source.Node-1].Category == pdg.Call {
			state.bad[source.Node] = true
		}
	}
}

func (a *factoryAnalysis) storeEffect(ref pdg.Ref, attrs pdg.Attributes, state *factoryState) {
	for _, write := range attrs.Writes {
		if !factoryStore(a.query.graph, write) {
			continue
		}
		a.invalidate(a.origins(ref, write.Position), state)
	}
}

func (a *factoryAnalysis) captureEffect(attrs pdg.Attributes, state *factoryState) {
	for _, symbol := range attrs.Symbols {
		a.invalidate(a.symbolOrigins(symbol), state)
	}
}

func (a *factoryAnalysis) addressEffect(ref pdg.Ref, state *factoryState) {
	path := a.query.paths.Path(a.query.paths.Operand(ref, 1), a.query.facts)
	if path.Root == 0 {
		invalidateIssued(state)
		return
	}
	a.invalidate(a.symbolOrigins(path.Root), state)
}

func (a *factoryAnalysis) invalidateExternal(state *factoryState) {
	for origin := range state.issued {
		if !a.isolatedOrigin(origin) {
			state.bad[origin] = true
		}
	}
}

func (a *factoryAnalysis) isolatedOrigin(ref pdg.Ref) bool {
	if a.query.isolatedPrimitive(ref) {
		return true
	}
	attrs := a.query.graph.NodeAttributes(a.query.graph.Nodes[ref-1])
	return attrs.CallForm == pdg.DirectCall && attrs.Callee.Status == pdg.Known && a.isolated[a.query.graph.Tables.Text(attrs.Callee.Value)]
}

func invalidateIssued(state *factoryState) { joinFactorySet(state.bad, state.issued) }

func factoryStore(graph *pdg.Graph, write pdg.Access) bool {
	if !write.DirectSymbol() {
		return true
	}
	return graph.Symbols[write.Symbol-1].Role == pdg.GlobalRole
}
