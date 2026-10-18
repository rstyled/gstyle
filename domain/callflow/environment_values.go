package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
)

func (values exitValues) liveBindings() map[pdg.Ref]exitValue {
	result := maps.Clone(values.bindings)
	if result == nil {
		result = map[pdg.Ref]exitValue{}
	}
	for symbol, address := range values.locations {
		if cell := values.heap[address]; !cell.Repeated {
			result[symbol] = cell.Value
		}
	}
	return result
}
func (values exitValues) synchronize(writes []bindingflow.ValueWrite) exitValues {
	values.heap = maps.Clone(values.heap)
	for _, write := range writes {
		if write.UnknownTarget() {
			values.heap = unknownHeap(values.heap)
			continue
		}
		if address := values.locations[write.Symbol]; address != "" && write.Location == 0 {
			writeEnvironmentCell(values.heap, address, abstractValue(write.Value))
		}
	}
	return values
}
func (values exitValues) repeatedHeap() map[string]bool {
	result := map[string]bool{}
	for address, cell := range values.heap {
		if cell.Repeated {
			result[address] = true
		}
	}
	return result
}
func (evaluation *exitEvaluation) bindClosure(point exitPoint, ref pdg.Ref) exitPoint {
	graph := evaluation.body.graph
	if graphRevision(graph) == "" {
		return point
	}
	attrs := graph.NodeAttributes(graph.Nodes[ref-1])
	if graph.Tables.Text(attrs.Construct) != "closure" {
		return point
	}
	name, err := bindingflow.CallableIdentity(graph, ref)
	if err != nil {
		return point
	}
	summary := evaluation.summaries[name]
	if summary.invocations == nil {
		return point
	}
	links := summary.invocations.captureBindings(graph)
	if links == nil {
		return point
	}
	return evaluation.constructEnvironment(point, ref, name, links)
}
func (evaluation *exitEvaluation) constructEnvironment(point exitPoint, ref pdg.Ref, name string, links map[pdg.Ref]pdg.Ref) exitPoint {
	values := evaluation.values[point.values]
	values.heap = maps.Clone(values.heap)
	if values.heap == nil {
		values.heap = environmentHeap{}
	}
	values.locations = maps.Clone(values.locations)
	if values.locations == nil {
		values.locations = map[pdg.Ref]string{}
	}
	var cells []bindingflow.CallableCell
	for capture, symbol := range links {
		address := evaluation.localCell(&values, symbol)
		cells = append(cells, bindingflow.CallableCell{Symbol: capture, Address: address})
	}
	target := bindingflow.Callable{Name: name, Revision: graphRevision(evaluation.body.graph), Cells: cells}
	values.nodes = cloneValueMap(values.nodes)
	values.nodes[ref] = callableValue(bindingflow.NewBoundCallableSet(target))
	point.values = evaluation.internValues(values)
	return point
}
func (evaluation *exitEvaluation) localCell(values *exitValues, symbol pdg.Ref) string {
	if address := values.locations[symbol]; address != "" {
		return address
	}
	graph := evaluation.body.graph
	address := ownedJSON(environmentAddress{Owner: graph.Tables.Text(graph.Function.ID), Revision: graphRevision(graph), Symbol: symbol})
	cell := environmentCell{Value: values.bindings[symbol], Repeated: !evaluation.body.facts.SingleCellSymbols[symbol]}
	if old, found := values.heap[address]; found {
		cell.Value = joinValue(old.Value, cell.Value)
		cell.Repeated = true
	}
	values.heap[address] = cell
	values.locations[symbol] = address
	return address
}
