package resourceflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// FactorySummary is a partial-return contract: the first result is fresh or
// empty. FailureEmpty additionally proves that a nonempty first result cannot
// accompany a non-nil second result. Neither field proves termination.
type FactorySummary struct{ Fresh, FailureEmpty bool }

type factoryRequirement struct {
	target                                      string
	primitive, isolated, safe, paired, errorNil bool
}

type factoryDescriptor struct {
	valid        bool
	requirements []factoryRequirement
}

type factoryValueKey struct {
	ref      pdg.Ref
	position pdg.Position
	symbol   bool
}

type factoryAnalysis struct {
	isolated map[string]bool
	query    *Query
	helpers  map[string]bool
	values   map[factoryValueKey]bindingflow.Origins
	inputs   []factoryState
}

func (a *factoryAnalysis) origins(ref pdg.Ref, position pdg.Position) bindingflow.Origins {
	key := factoryValueKey{ref: ref, position: position}
	if cached, found := a.values[key]; found {
		return cached
	}
	var value bindingflow.Origins
	if position == 0 {
		value = a.query.paths.OriginsOf(ref, a.query.facts)
	} else {
		value = a.query.paths.OriginsAt(ref, position, a.query.facts)
	}
	a.values[key] = value
	return value
}

func (a *factoryAnalysis) symbolOrigins(symbol pdg.Ref) bindingflow.Origins {
	key := factoryValueKey{ref: symbol, symbol: true}
	if cached, found := a.values[key]; found {
		return cached
	}
	value := a.query.paths.SymbolOrigins(symbol, a.query.facts)
	a.values[key] = value
	return value
}

type factoryState struct {
	issued, bad map[pdg.Ref]bool
	unknownExit bool
}

// Acquires and FailureEmpty read canonical target/contract facts. Source syntax
// identifies a report boundary but cannot replace a missing call identity.
func (q *Query) Acquires(ref pdg.Ref, sources map[string]bool) bool {
	if q == nil || ref == 0 || int(ref) > len(q.graph.Nodes) || q.graph.Nodes[ref-1].Category != pdg.Call {
		return false
	}
	return q.primitiveFile(ref) || q.helper(ref, Policy{Helpers: sources})
}

func (q *Query) FailureEmpty(ref pdg.Ref, failures map[string]bool) bool {
	if q == nil || ref == 0 || int(ref) > len(q.graph.Nodes) || q.graph.Nodes[ref-1].Category != pdg.Call {
		return false
	}
	return q.primitiveFile(ref) || q.helper(ref, Policy{Helpers: failures})
}

func (q *Query) AcquisitionName(ref pdg.Ref) string {
	if q == nil || ref == 0 || int(ref) > len(q.graph.Nodes) {
		return ""
	}
	attrs := q.graph.NodeAttributes(q.graph.Nodes[ref-1])
	if attrs.CallForm == pdg.DirectCall && attrs.Callee.Status == pdg.Known {
		return q.graph.Tables.Text(attrs.Callee.Value)
	}
	return q.methods[ref]
}

// OSFileAcquisition identifies the standard OS acquisitions also reported by
// the typed quality adapter. Other file contracts still need ownership findings.
func (q *Query) OSFileAcquisition(ref pdg.Ref) bool {
	if !q.Acquires(ref, nil) {
		return false
	}
	return q.isolatedPrimitive(ref)
}

func (q *Query) factory(helpers, isolated map[string]bool) factoryDescriptor {
	if !q.HasFlow() || len(q.graph.Function.Outputs) == 0 {
		return factoryDescriptor{}
	}
	analysis := factoryAnalysis{isolated: isolated, query: q, helpers: helpers, values: map[factoryValueKey]bindingflow.Origins{}, inputs: make([]factoryState, len(q.flow.Blocks))}
	if !analysis.possibleReturns() {
		return factoryDescriptor{}
	}
	analysis.solve()
	return analysis.describe()
}

func (q *Query) primitiveFile(ref pdg.Ref) bool {
	if q.primitiveResource(directResourceCall(q.graph, q.graph.Nodes[ref-1])) {
		return true
	}
	return q.protocol.Resource == "" && q.methods[ref] == "io/fs.FS.Open"
}

func (q *Query) sourceRequirement(source bindingflow.ValueSource, state factoryState) (factoryRequirement, bool) {
	if source.Result > 1 || q.graph.Nodes[source.Node-1].Category != pdg.Call {
		return factoryRequirement{}, false
	}
	requirement := factoryRequirement{primitive: q.primitiveFile(source.Node), isolated: q.isolatedPrimitive(source.Node), safe: !state.bad[source.Node]}
	if requirement.primitive {
		return requirement, true
	}
	attrs := q.graph.NodeAttributes(q.graph.Nodes[source.Node-1])
	if attrs.CallForm != pdg.DirectCall || attrs.Callee.Status != pdg.Known {
		return factoryRequirement{}, false
	}
	requirement.target = q.graph.Tables.Text(attrs.Callee.Value)
	return requirement, true
}

func (q *Query) emptySource(source bindingflow.ValueSource) bool {
	if source.Result != 0 {
		return false
	}
	if q.facts.NilConstant(source.Node) {
		return true
	}
	if !source.Zero || source.Symbol == 0 {
		return false
	}
	return nilResourceSymbol(q.graph, source.Symbol)
}

func (q *Query) isolatedPrimitive(ref pdg.Ref) bool {
	return q.primitiveResource(directResourceCall(q.graph, q.graph.Nodes[ref-1]))
}

func primitiveFileName(name string) bool {
	switch name {
	case "os.Open", "os.OpenFile", "os.Create", "os.CreateTemp":
		return true
	default:
		return false
	}
}

func nilResourceSymbol(graph *pdg.Graph, symbol pdg.Ref) bool {
	for _, extension := range graph.Symbols[symbol-1].Extensions {
		if graph.Tables.Text(extension.Name) != "gstyle.memory:kind" {
			continue
		}
		value := graph.Tables.Text(extension.Value)
		return value == `"pointer"` || value == `"interface"`
	}
	return false
}
