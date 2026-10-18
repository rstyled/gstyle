package gopatterns

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
)

type controlSelection struct {
	tag   ast.Expr
	typed bool
}

func (a *scalarCFGBuilder) selectionPredicate(clause *ast.CaseClause) pdg.Ref {
	index := a.selectors[clause]
	a.selectors[clause]++
	expression := clause.List[index]
	selection := a.selections[expression]
	b := a.builder
	attrs := b.selectionAttributes(selection, expression)
	ref := b.scalarEntity(b.scalarSourceSpan(clause), pdg.Control, attrs)
	b.approximateControlNode(ref)
	b.controlValueEdge(b.nodes[selection.tag], ref, 1)
	if !selection.typed {
		position := pdg.Position(2)
		if selection.tag == nil {
			position = 1
		}
		b.controlValueEdge(b.nodes[expression], ref, position)
	}
	return ref
}

func (b *irBuilder) approximateControlNode(ref pdg.Ref) {
	node := &b.graph.Nodes[ref-1]
	node.Evidence = b.pool.Evidence(pdg.Approximated, semanticControlDescription)
	node.Provenance = b.pool.Origin(pdg.Analysis, semanticControlDescription)
}

func (b *irBuilder) controlValueEdge(source, target pdg.Ref, position pdg.Position) {
	if source == 0 {
		return
	}
	b.graph.Edges = append(b.graph.Edges, pdg.Edge{ID: b.pool.Text("control-value-e" + strconv.Itoa(len(b.graph.Edges)+1)), Source: source, Target: target, Kind: pdg.Data, Position: position, Subkind: b.pool.Text("control-value"), Evidence: b.pool.Evidence(pdg.Supported, "Value consumed by a source control decision."), Provenance: b.sourceOrigin})
}

func (b *irBuilder) selectionAttributes(selection controlSelection, expression ast.Expr) pdg.Attributes {
	operator := "=="
	switch {
	case selection.typed:
		operator = "type-is"
	case selection.tag == nil:
		operator = "truth-test"
	}
	attrs := pdg.Attributes{Construct: b.pool.Text("switch-case"), Operator: b.pool.Known(operator), Outcomes: []pdg.Text{b.pool.Text("true"), b.pool.Text("false")}}
	if selection.typed {
		attrs.Extensions = b.pool.SingletonExtension("gstyle.control:case-type", strconv.Quote(b.info.TypeOf(expression).String()))
	}
	if selectionNormalCompletion(selection, expression, b.info) {
		b.runtimeExtension(&attrs, "normal-completion", "true")
	}
	return attrs
}

func controlSelections(body *ast.BlockStmt) map[ast.Expr]controlSelection {
	out := map[ast.Expr]controlSelection{}
	ast.Inspect(body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncLit:
			return false
		case *ast.SwitchStmt:
			selectionClauses(out, node.Body, controlSelection{tag: node.Tag})
		case *ast.TypeSwitchStmt:
			selectionClauses(out, node.Body, controlSelection{tag: typeSwitchValue(node.Assign), typed: true})
		}
		return true
	})
	return out
}

func typeSwitchValue(statement ast.Stmt) ast.Expr {
	var expression ast.Expr
	switch node := statement.(type) {
	case *ast.AssignStmt:
		expression = node.Rhs[0]
	case *ast.ExprStmt:
		expression = node.X
	}
	return unparen(expression).(*ast.TypeAssertExpr).X
}

func selectionClauses(out map[ast.Expr]controlSelection, body *ast.BlockStmt, selection controlSelection) {
	for _, statement := range body.List {
		for _, expression := range statement.(*ast.CaseClause).List {
			out[expression] = selection
		}
	}
}

func selectionNormalCompletion(selection controlSelection, expression ast.Expr, info *types.Info) bool {
	return selection.typed || selection.tag == nil || normalEquality(selection.tag, expression, info)
}
