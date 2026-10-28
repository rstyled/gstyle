package govalidation

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
)

type valueEdge struct {
	from, to   pdg.Ref
	position   pdg.Position
	controller pdg.Ref
	outcome    string
}

type valueCheck struct {
	g        *pdg.Graph
	fset     *token.FileSet
	info     *types.Info
	nodes    map[site]pdg.Ref
	ids      map[string]pdg.Ref
	retained []bool
	expected map[valueEdge]int
	err      error
}

func (a *valueCheck) ref(node ast.Node, category pdg.Category, synthetic bool) pdg.Ref {
	return a.nodes[site{sourceSpan(a.fset, node), category, synthetic}]
}

func (a *valueCheck) edge(from, to pdg.Ref, position pdg.Position) {
	if from == 0 || to == 0 {
		a.err = fmt.Errorf("value operation correspondence is missing")
		return
	}
	if !a.retained[to-1] {
		return
	}
	if !a.retained[from-1] {
		a.err = fmt.Errorf("reachable value has an unreachable source")
		return
	}
	a.expected[valueEdge{from: from, to: to, position: position}]++
}

func (a *valueCheck) source(node ast.Node) bool {
	if expression, ok := node.(ast.Expr); ok {
		a.expression(expression)
		return true
	}
	a.sourceStatement(node)
	return true
}

func (a *valueCheck) sourceStatement(node ast.Node) {
	switch node := node.(type) {
	case *ast.AssignStmt:
		a.assignment(node)
	case *ast.ValueSpec:
		a.declaration(node)
	case *ast.ReturnStmt:
		a.returnValue(node)
	case *ast.IfStmt:
		a.edge(a.expressionRef(node.Cond), a.ref(node, pdg.Control, false), 1)
	case *ast.IncDecStmt:
		a.edge(a.ref(unparen(node.X), pdg.Read, true), a.ref(node, pdg.Assignment, false), 1)
	}
}

func (a *valueCheck) assignment(node *ast.AssignStmt) {
	category := pdg.Assignment
	if node.Tok == token.DEFINE {
		category = pdg.Declaration
	}
	owner := a.ref(node, category, false)
	for i, value := range node.Rhs {
		a.edge(a.expressionRef(value), owner, pdg.Position(i+1))
	}
}

func (a *valueCheck) declaration(node *ast.ValueSpec) {
	for i, value := range node.Values {
		a.edge(a.expressionRef(value), a.ref(node, pdg.Declaration, false), pdg.Position(i+1))
	}
}

func (a *valueCheck) returnValue(node *ast.ReturnStmt) {
	owner := a.ref(node, pdg.Return, false)
	if owner == 0 || !a.retained[owner-1] {
		return
	}
	for i, value := range node.Results {
		a.edge(a.expressionRef(value), owner, pdg.Position(i+1))
	}
	if len(node.Results) == 0 {
		if read := a.ref(node, pdg.Read, true); read != 0 {
			a.edge(read, owner, 1)
		}
	}
	a.returnSlots(owner)
}

func (a *valueCheck) returnSlots(owner pdg.Ref) {
	for _, output := range a.g.Nodes {
		if output.Category == pdg.FormalOutput {
			a.edge(owner, a.nodes[site{a.g.Spans[output.Span-1], pdg.FormalOutput, false}], a.g.NodeAttributes(output).Position)
		}
	}
}

func (a *valueCheck) expressionRef(expression ast.Expr) pdg.Ref {
	span := sourceSpan(a.fset, expression)
	for _, key := range []site{{span, pdg.Read, false}, {span, pdg.Computation, false}, {span, pdg.Merge, true}} {
		if ref := a.nodes[key]; ref != 0 {
			return ref
		}
	}
	return 0
}

func (a *valueCheck) expression(expression ast.Expr) {
	if a.info.Types[expression].IsType() {
		return
	}
	owner := a.expressionRef(expression)
	if owner == 0 {
		return
	} // Binding declarations and write targets have no value site.
	switch node := expression.(type) {
	case *ast.ParenExpr:
		a.edge(a.expressionRef(node.X), owner, 1)
	case *ast.UnaryExpr:
		a.edge(a.expressionRef(node.X), owner, 1)
	case *ast.BinaryExpr:
		a.binary(node, owner)
	}
}

func (a *valueCheck) binary(node *ast.BinaryExpr, owner pdg.Ref) {
	if node.Op == token.LAND || node.Op == token.LOR {
		a.logical(node, owner)
		return
	}
	a.edge(a.expressionRef(node.X), owner, 1)
	a.edge(a.expressionRef(node.Y), owner, 2)
}

func (a *valueCheck) compare() error {
	for _, edge := range a.g.Edges {
		if err := a.consume(edge); err != nil {
			return err
		}
	}
	for edge, count := range a.expected {
		if count != 0 {
			return fmt.Errorf("missing value edge %d -> %d", edge.from, edge.to)
		}
	}
	return nil
}

func (a *valueCheck) consume(edge pdg.Edge) error {
	if edge.Kind != pdg.Data || edge.Definition != 0 {
		return nil
	}
	if !a.retained[edge.Target-1] {
		return nil
	}
	if !a.retained[edge.Source-1] {
		return fmt.Errorf("reachable value has an unreachable source")
	}
	key, err := a.key(edge)
	if err != nil {
		return err
	}
	return a.consumeKey(key)
}

func (a *valueCheck) consumeKey(key valueEdge) error {
	if a.expected[key] == 0 {
		return fmt.Errorf("unexpected value edge %d -> %d", key.from, key.to)
	}
	a.expected[key]--
	return nil
}

func Values(g *pdg.Graph, fn *ast.FuncDecl, fset *token.FileSet, info *types.Info) error {
	if err := pdg.Validate(g); err != nil {
		return err
	}
	a := valueCheck{g: g, fset: fset, info: info, nodes: map[site]pdg.Ref{}, ids: map[string]pdg.Ref{}, retained: pdg.NormalProjection(g), expected: map[valueEdge]int{}}
	if a.retained == nil {
		return fmt.Errorf("value reachability is missing")
	}
	for i, node := range g.Nodes {
		a.nodes[site{g.Spans[node.Span-1], node.Category, node.Synthetic}] = pdg.Ref(i + 1)
		a.ids[g.Tables.Text(node.ID)] = pdg.Ref(i + 1)
	}
	ast.Inspect(fn.Body, a.source)
	if a.err != nil {
		return a.err
	}
	return a.compare()
}
