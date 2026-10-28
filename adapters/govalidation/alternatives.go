package govalidation

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (a *valueCheck) logical(node *ast.BinaryExpr, owner pdg.Ref) {
	predicate := a.ref(node.X, pdg.Control, true)
	left, right := a.expressionRef(node.X), a.expressionRef(node.Y)
	a.edge(left, predicate, 1)
	if node.Op == token.LOR {
		a.alternative(left, owner, predicate, true)
		a.alternative(right, owner, predicate, false)
		return
	}
	a.alternative(right, owner, predicate, true)
	a.alternative(left, owner, predicate, false)
}

func (a *valueCheck) alternative(from, to, controller pdg.Ref, selected bool) {
	if from == 0 || to == 0 || controller == 0 {
		a.err = fmt.Errorf("alternative operation correspondence is missing")
		return
	}
	if !a.retained[to-1] {
		return
	}
	position, outcome := pdg.Position(2), "false"
	if selected {
		position, outcome = 1, "true"
	}
	a.expected[valueEdge{from: from, to: to, position: position, controller: controller, outcome: outcome}]++
}

func (a *valueCheck) key(edge pdg.Edge) (valueEdge, error) {
	key := valueEdge{from: edge.Source, to: edge.Target, position: edge.Position}
	if a.g.Tables.Text(edge.Subkind) != "value_alternative" {
		return key, nil
	}
	if edge.Annotations == 0 {
		return key, fmt.Errorf("alternative lacks a guard annotation")
	}
	for _, extension := range a.g.Annotations[edge.Annotations-1] {
		if a.g.Tables.Text(extension.Name) == "gstyle.go:guards" {
			return a.guardedKey(key, edge.Outcome, extension.Value)
		}
	}
	return key, fmt.Errorf("alternative lacks a selecting decision")
}

func (a *valueCheck) guardedKey(key valueEdge, outcome, value pdg.Text) (valueEdge, error) {
	var guards [][]map[string]string
	err := json.Unmarshal([]byte(a.g.Tables.Text(value)), &guards)
	if err != nil || len(guards) != 1 || len(guards[0]) != 1 {
		return key, fmt.Errorf("alternative requires one selecting guard atom")
	}
	atom := guards[0][0]
	key.controller = a.ids[atom["controller_pdg_node_id"]]
	key.outcome = atom["outcome"]
	if key.controller == 0 || key.outcome != a.g.Tables.Text(outcome) {
		return key, fmt.Errorf("alternative guard and edge disagree")
	}
	return key, nil
}
