package govalidation

import (
	"fmt"
	"strconv"

	"github.com/rstyled/gstyle/domain/ccgraph"
	"github.com/rstyled/gstyle/domain/pdg"
)

var categories = map[pdg.Category]string{
	pdg.Declaration: "declaration", pdg.Read: "read", pdg.Assignment: "assignment", pdg.Computation: "computation", pdg.Control: "control",
	pdg.FormalInput: "formal_input", pdg.FormalOutput: "formal_output", pdg.Return: "return", pdg.Merge: "merge",
}

// Project preserves operation/edge occurrence identities and records every
// removal. Callers must first verify source/CFG/dependency correspondence.
func Project(g *pdg.Graph) (ccgraph.Projection, error) {
	if err := pdg.Validate(g); err != nil {
		return ccgraph.Projection{}, err
	}
	retained := pdg.NormalProjection(g)
	if retained == nil {
		return ccgraph.Projection{}, fmt.Errorf("validated normal projection context is missing")
	}
	operations, err := projectionOperations(g, retained)
	if err != nil {
		return ccgraph.Projection{}, err
	}
	input := ccgraph.ProjectionInput{Nodes: operations}
	input.Edges = projectionEdges(g)
	projection, err := ccgraph.Project(input)
	if err != nil {
		return ccgraph.Projection{}, err
	}
	if projection.Features[8] != g.Function.ReferenceCount.Value {
		return ccgraph.Projection{}, fmt.Errorf("projected references disagree with validated root counting policy")
	}
	return projection, nil
}

func projectionEdges(g *pdg.Graph) []ccgraph.Edge {
	var edges []ccgraph.Edge
	for _, edge := range g.Edges {
		family := map[pdg.EdgeKind]ccgraph.Family{pdg.Data: ccgraph.Data, pdg.ControlEdge: ccgraph.Ctrl, pdg.Execution: ccgraph.Execution}[edge.Kind]
		edges = append(edges, ccgraph.Edge{From: g.Tables.Text(g.Nodes[edge.Source-1].ID), To: g.Tables.Text(g.Nodes[edge.Target-1].ID), Family: family})
	}
	return edges
}

func projectionOperations(g *pdg.Graph, retained []bool) ([]ccgraph.Operation, error) {
	var operations []ccgraph.Operation
	for i, node := range g.Nodes {
		category := categories[node.Category]
		if category == "" {
			return nil, fmt.Errorf("unregistered native category %d", node.Category)
		}
		operation := ccgraph.Operation{ID: g.Tables.Text(node.ID), Category: category, Reachable: &retained[i]}
		for _, read := range g.NodeAttributes(node).Reads {
			operation.ReadSymbols = append(operation.ReadSymbols, strconv.Itoa(int(read.Symbol)))
		}
		operations = append(operations, operation)
	}
	return operations, nil
}
