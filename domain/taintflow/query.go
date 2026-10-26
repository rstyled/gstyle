package taintflow

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"slices"
)

type query struct {
	cells        map[pdg.Ref]cell
	captures     []pdg.Ref
	environments map[string]map[pdg.Ref]pdg.Ref
	key          string
	functions    map[int][]string
	calls        *callflow.CallSites
	facts        bindingflow.Facts
	graph        *pdg.Graph
	paths        *bindingflow.Query
	flow         *pdg.NormalFlow
	models       Models
	name         string
	inputs       map[pdg.Ref]int
	addresses    map[pdg.Ref]map[pdg.Position]pdg.Ref
	live         map[pdg.Ref]bool
}
type evaluation struct {
	solver        *summarySolver
	arguments     map[pdg.Position]value
	argumentPaths map[argumentPath]value
	query         *query
	catalog       map[string]*query
	summaries     map[string]summary
	state         state
	result        *summary
	exits         map[pdg.Ref]state
}

// Analyze solves symbolic parameter/result/field summaries to a fixed point.
// Unknown external calls preserve argument taint. No summary proves a path
// feasible, validates an allowlist, or establishes that an unreported path is safe.
func Analyze(graphs map[string]*pdg.Graph, models Models) ([]Finding, error) {
	queries, err := prepare(graphs, models)
	if err != nil {
		return nil, err
	}
	if err := linkEnvironments(queries); err != nil {
		return nil, err
	}
	if !hasEndpoints(queries) {
		return nil, nil
	}
	summaries := map[string]summary{}
	for name := range queries {
		summaries[name] = emptySummary()
	}
	solveSummaries(queries, summaries)
	return findings(summaries), nil
}
func prepare(graphs map[string]*pdg.Graph, models Models) (map[string]*query, error) {
	result := map[string]*query{}
	for name, graph := range graphs {
		q, err := newQuery(graph, models)
		if err != nil {
			return nil, fmt.Errorf("taint %s: %w", name, err)
		}
		if q == nil {
			continue
		}
		if q.name == "" || name != q.name {
			return nil, fmt.Errorf("taint function identity does not match catalog key %q", name)
		}
		result[name] = q
	}
	return result, nil
}
func newQuery(graph *pdg.Graph, models Models) (*query, error) {
	prepared, err := callflow.NewQueries(graph)
	paths := prepared.Bindings
	if err != nil || graph == nil {
		return nil, err
	}
	flow := pdg.BodyFlow(graph)
	if flow == nil {
		return nil, nil
	}
	q := &query{graph: graph, paths: paths, flow: flow, models: models, name: graph.Tables.Text(graph.Function.ID), inputs: map[pdg.Ref]int{}, addresses: map[pdg.Ref]map[pdg.Position]pdg.Ref{}}
	q.calls, q.facts = prepared.Calls, prepared.Facts
	q.indexInputs()
	for _, edge := range graph.Edges {
		q.indexAddress(edge)
	}
	q.key = q.name
	q.indexLiveNodes()
	return q, nil
}
func (q *query) indexAddress(edge pdg.Edge) {
	if edge.Kind != pdg.Data || q.graph.Tables.Text(edge.Subkind) != "write-address" {
		return
	}
	if q.addresses[edge.Target] == nil {
		q.addresses[edge.Target] = map[pdg.Position]pdg.Ref{}
	}
	q.addresses[edge.Target][edge.Position] = edge.Source
}
func (q *query) initial() state {
	result := emptyState()
	for symbol, index := range q.inputs {
		item := q.initialInput(symbol, index)
		item.Opaque = true
		if names := q.functions[index]; len(names) != 0 {
			item.Opaque = false
			for _, name := range names {
				item.Functions[name] = true
			}
		}
		result.Bindings[symbol] = item
		if key, found := q.cells[symbol]; found {
			result.Memory[key] = item
		}
	}
	return result
}
func findings(summaries map[string]summary) []Finding {
	seen := map[Finding]bool{}
	for _, item := range summaries {
		collectFindings(item, seen)
	}
	result := make([]Finding, 0, len(seen))
	for item := range seen {
		result = append(result, item)
	}
	slices.SortFunc(result, compareFinding)
	return result
}
func collectFindings(item summary, seen map[Finding]bool) {
	for sink, deps := range item.Sinks {
		for dep := range deps {
			if dep.Source.Node != 0 {
				seen[Finding{Source: dep.Source, Sink: sink.Site, Kind: sink.Kind}] = true
			}
		}
	}
}

func (q *query) indexInputs() {
	for index, input := range q.graph.Function.Inputs {
		if input.Symbol != 0 {
			q.inputs[input.Symbol] = index + 1
		}
	}
	q.indexCells()
}
