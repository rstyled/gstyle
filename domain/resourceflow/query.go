// Package resourceflow follows canonical operation paths for file-protocol
// release evidence. Close is a protocol action, not a proof of a foreign
// implementation's operating-system behavior. Local exit and operation effects use canonical callflow queries.
package resourceflow

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

type Policy struct {
	// recognizeDeferred is advisory presence evidence, never a release proof.
	recognizeDeferred          bool
	releaseMethod              string
	resourceFields             []uint32
	aliases                    map[pdg.Ref]bool
	Resource, AcquisitionError pdg.Ref
	TransferReturn             bool
	TransferCalls              bool
	Helpers                    map[string]bool
	Exits                      callflow.ExitSummaries
}

type point struct {
	block, offset int
	present       bool
	unstable      bool
}

type Query struct {
	prepared   callflow.Queries
	graph      *pdg.Graph
	flow       *pdg.NormalFlow
	paths      *bindingflow.Query
	machine    deferflow.Machine[bool]
	calls      *callflow.CallSites
	operations *callflow.OperationEffects
	sites      []point
	facts      bindingflow.Facts
	methods    map[pdg.Ref]string
}

func New(graph *pdg.Graph) (*Query, error) {
	if graph == nil {
		return nil, nil
	}
	flow := pdg.BodyFlow(graph)
	if flow == nil {
		return nil, pdg.Validate(graph)
	}
	return newQuery(graph, flow)
}

func newQuery(graph *pdg.Graph, flow *pdg.NormalFlow) (*Query, error) {
	queries, err := callflow.NewQueries(graph)
	if err != nil {
		return nil, err
	}
	machine := deferflow.NewMachineWithIndex[bool](queries.Deferred)
	query := &Query{prepared: queries, graph: graph, flow: flow, paths: queries.Bindings, machine: machine, calls: queries.Calls, operations: queries.Operations, sites: make([]point, len(graph.Nodes)), facts: queries.Facts, methods: map[pdg.Ref]string{}}
	if err := query.index(); err != nil {
		return nil, err
	}
	return query, nil
}

func (q *Query) Reachable(ref pdg.Ref) bool {
	if q == nil || ref == 0 || int(ref) > len(q.sites) {
		return false
	}
	site := q.sites[ref-1]
	return site.present && q.flow.Blocks[site.block].Reachable
}

// Operand exposes the selected canonical producer for a source boundary that
// identifies a return/acquisition site. Missing/conflicting inputs stay zero.
func (q *Query) Operand(ref pdg.Ref, position pdg.Position) pdg.Ref {
	if q == nil {
		return 0
	}
	return q.paths.Operand(ref, position)
}

func (q *Query) index() error {
	q.indexSites()
	for i, node := range q.graph.Nodes {
		if err := q.indexFacts(pdg.Ref(i+1), q.graph.NodeAttributes(node)); err != nil {
			return err
		}
	}
	return nil
}

func (q *Query) indexFacts(ref pdg.Ref, attrs pdg.Attributes) error {
	for _, extension := range attrs.Extensions {
		if err := q.indexFact(ref, extension); err != nil {
			return err
		}
	}
	return nil
}

func (q *Query) indexFact(ref pdg.Ref, extension pdg.Extension) error {
	name, value := q.graph.Tables.Text(extension.Name), q.graph.Tables.Text(extension.Value)
	switch name {
	case "gstyle.go:method-declaration":
		return q.methodFact(ref, value)
	}
	return nil
}

func (q *Query) methodFact(ref pdg.Ref, value string) error {
	var identity string
	if err := json.Unmarshal([]byte(value), &identity); err != nil {
		return fmt.Errorf("decode resource method declaration: %w", err)
	}
	if identity == "" || q.methods[ref] != "" {
		return fmt.Errorf("empty or duplicate resource method declaration")
	}
	q.methods[ref] = identity
	return nil
}

func (q *Query) symbol(ref pdg.Ref) pdg.Ref {
	path := q.paths.Path(ref, q.facts)
	if len(path.Fields) != 0 {
		return 0
	}
	return path.Root
}

func (q *Query) resource(ref pdg.Ref, policy Policy) bool {
	if policy.recognizeDeferred {
		path := q.paths.Path(ref, q.facts)
		return ownedSymbol(path.Root, policy) && slices.Equal(path.Fields, policy.resourceFields)
	}
	return ownedSymbol(q.symbol(ref), policy)
}

func (q *Query) helper(ref pdg.Ref, policy Policy) bool {
	attrs := q.graph.NodeAttributes(q.graph.Nodes[ref-1])
	return attrs.CallForm == pdg.DirectCall && attrs.Callee.Status == pdg.Known && policy.Helpers[q.graph.Tables.Text(attrs.Callee.Value)]
}

func (q *Query) closeMethod(ref pdg.Ref) bool { return strings.HasSuffix(q.methods[ref], ".Close") }

func (q *Query) oneInput(ref pdg.Ref) bool {
	inputs, known := q.paths.RequiredOperands(ref)
	return known && len(inputs) == 1 && inputs[0] == 1
}

func (q *Query) releaseCall(ref pdg.Ref, policy Policy) bool {
	if q.graph.Nodes[ref-1].Category != pdg.Call || !q.oneInput(ref) {
		return false
	}
	if !q.closeMethod(ref) && !q.helper(ref, policy) {
		return false
	}
	return q.resource(q.paths.Operand(ref, 1), policy)
}

func (q *Query) indexSites() {
	for block, flow := range q.flow.Blocks {
		for offset, ref := range flow.Operations {
			q.sites[ref-1] = point{block: block, offset: offset, present: true}
		}
	}
}

// CallQueries shares immutable indexes. Proof states remain local to each query.
func (q *Query) CallQueries() callflow.Queries { return q.prepared }
