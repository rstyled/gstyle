package patterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/taintflow"
)

// QueryTaintFlows solves canonical summaries across the supplied functions.
// Imported graphs without body flow keep the existing local dependency query.
func QueryTaintFlows(facts []*MiningFacts) ([]TaintFlow, error) {
	graphs := map[string]*pdg.Graph{}
	views := map[string]*MiningFacts{}
	for _, fact := range facts {
		if localOnlyTaintFact(fact) {
			continue
		}
		graphs[fact.ID] = fact.Canonical
		views[fact.ID] = semanticFact(fact)
	}
	models := taintflow.Models{
		Source: func(graph *pdg.Graph, ref pdg.Ref) bool {
			return semanticNode(graph, graph.Nodes[ref-1]).isTaintSource()
		},
		Sinks: func(graph *pdg.Graph, ref pdg.Ref) []taintflow.Sink {
			return summarySinks(views[graph.Tables.Text(graph.Function.ID)], ref)
		},
	}
	findings, err := taintflow.Analyze(graphs, models)
	if err != nil {
		return nil, err
	}
	result := make([]TaintFlow, 0, len(findings))
	for _, finding := range findings {
		result = append(result, summaryTaintFlow(finding, views))
	}
	return append(result, localOnlyTaintFlows(facts)...), nil
}
func summarySinks(fact *MiningFacts, ref pdg.Ref) []taintflow.Sink {
	if fact == nil {
		return nil
	}
	index := int(ref) - 1
	sink := fact.Pdg.Nodes[index].matchTaintSink()
	if sink == nil {
		return nil
	}
	sink = commandSink(fact, index, sink)
	positions := make([]pdg.Position, len(sink.dangerousArgs))
	for i, argument := range sink.dangerousArgs {
		positions[i] = pdg.Position(argument + 1)
	}
	return []taintflow.Sink{{Kind: sink.desc, Positions: positions}}
}
func summaryTaintFlow(finding taintflow.Finding, facts map[string]*MiningFacts) TaintFlow {
	source, sink := facts[finding.Source.Function], facts[finding.Sink.Function]
	from, to := int(finding.Source.Node)-1, int(finding.Sink.Node)-1
	return TaintFlow{FuncID: sink.ID, SourceFuncID: source.ID, SourceNode: canonicalNodeID(source.Canonical, from), SinkNode: canonicalNodeID(sink.Canonical, to), SourceLine: source.Pdg.Nodes[from].Line, SinkLine: sink.Pdg.Nodes[to].Line, SourceCall: source.Pdg.Nodes[from].CalleeID, SinkCall: sink.Pdg.Nodes[to].CalleeID, Risk: finding.Kind}
}

func localOnlyTaintFlows(facts []*MiningFacts) []TaintFlow {
	var local []*MiningFacts
	for _, fact := range facts {
		if localOnlyTaintFact(fact) {
			local = append(local, fact)
		}
	}
	return FindTaintFlows(local)
}
func localOnlyTaintFact(fact *MiningFacts) bool {
	if fact == nil || fact.Canonical == nil {
		return true
	}
	return pdg.BodyFlow(fact.Canonical) == nil
}
