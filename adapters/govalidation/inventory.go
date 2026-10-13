// Package govalidation checks native extraction against typed source facts.
// It does not trust producer capability claims or reuse producer lowering.
package govalidation

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
)

type site struct {
	span      pdg.Span
	category  pdg.Category
	synthetic bool
}

type inventory struct {
	fset         *token.FileSet
	info         *types.Info
	expected     map[site]int
	sources      map[site]ast.Node
	writes       map[ast.Node]bool
	namedResults int
}

// Inventory verifies every source operation occurrence, including unreachable
// syntax, and all actual interface value entities. It leaves CFG and dependency
// completeness to separate checks; success alone is not conformance.
func Inventory(g *pdg.Graph, fn *ast.FuncDecl, fset *token.FileSet, info *types.Info) error {
	if err := pdg.Validate(g); err != nil {
		return err
	}
	if err := functionIdentity(g, fn, fset); err != nil {
		return err
	}
	a := inventory{fset: fset, info: info, expected: map[site]int{}, sources: map[site]ast.Node{}, writes: map[ast.Node]bool{}}
	a.fields(fn.Recv, pdg.FormalInput)
	a.fields(fn.Type.Params, pdg.FormalInput)
	a.fields(fn.Type.Results, pdg.FormalOutput)
	if fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			a.namedResults += len(field.Names)
		}
	}
	a.targets(fn.Body)
	ast.Inspect(fn.Body, a.source)
	return a.compare(g)
}

func functionIdentity(g *pdg.Graph, fn *ast.FuncDecl, fset *token.FileSet) error {
	if g.Function.Span != sourceSpan(fset, fn) {
		return fmt.Errorf("function boundary disagrees with source")
	}
	if g.Function.Name.Status != pdg.Known || g.Tables.Text(g.Function.Name.Value) != fn.Name.Name {
		return fmt.Errorf("function name disagrees with source")
	}
	return nil
}

func (a *inventory) add(node ast.Node, category pdg.Category, synthetic bool) {
	span := pdg.Span{Source: 1, Start: uint32(a.fset.PositionFor(node.Pos(), false).Offset), End: uint32(a.fset.PositionFor(node.End(), false).Offset)}
	a.expected[site{span, category, synthetic}]++
	a.sources[site{span, category, synthetic}] = node
}

func (a *inventory) fields(fields *ast.FieldList, category pdg.Category) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		for range max(1, len(field.Names)) {
			a.add(field, category, false)
		}
	}
}

func (a *inventory) targets(body *ast.BlockStmt) {
	ast.Inspect(body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.AssignStmt:
			for _, lhs := range node.Lhs {
				a.target(lhs)
			}
		case *ast.ValueSpec:
			for _, name := range node.Names {
				a.writes[name] = true
			}
		case *ast.IncDecStmt:
			a.target(node.X)
		}
		return true
	})
}

func (a *inventory) source(node ast.Node) bool {
	if expression, ok := node.(ast.Expr); ok && a.info.Types[expression].IsType() {
		return false
	}
	if statement, ok := node.(ast.Stmt); ok {
		a.statement(statement)
		return true
	}
	switch node := node.(type) {
	case *ast.ValueSpec:
		a.add(node, pdg.Declaration, false)
	case ast.Expr:
		a.expression(node)
	}
	return true
}

func (a *inventory) statement(node ast.Stmt) {
	switch node := node.(type) {
	case *ast.AssignStmt:
		a.assignment(node)
	case *ast.ReturnStmt:
		a.returnSite(node)
	case *ast.IncDecStmt:
		a.add(node, pdg.Assignment, false)
		a.add(unparen(node.X), pdg.Read, true)
	case *ast.IfStmt:
		a.add(node, pdg.Control, false)
	}
}

func (a *inventory) returnSite(node *ast.ReturnStmt) {
	a.add(node, pdg.Return, false)
	if len(node.Results) != 0 {
		return
	}
	for range a.namedResults {
		a.add(node, pdg.Read, true)
	}
}

func (a *inventory) assignment(node *ast.AssignStmt) {
	category := pdg.Assignment
	if node.Tok == token.DEFINE {
		category = pdg.Declaration
	}
	a.add(node, category, false)
	if node.Tok == token.DEFINE || node.Tok == token.ASSIGN {
		return
	}
	for _, lhs := range node.Lhs {
		a.add(unparen(lhs), pdg.Read, true)
	}
}

func (a *inventory) expression(expression ast.Expr) {
	if a.writes[expression] {
		return
	}
	switch expression := expression.(type) {
	case *ast.BasicLit, *ast.UnaryExpr, *ast.ParenExpr:
		a.add(expression, pdg.Computation, false)
	case *ast.BinaryExpr:
		a.binary(expression)
	case *ast.Ident:
		a.identifier(expression)
	}
}

func (a *inventory) binary(binary *ast.BinaryExpr) {
	if binary.Op != token.LAND && binary.Op != token.LOR {
		a.add(binary, pdg.Computation, false)
		return
	}
	a.add(binary, pdg.Merge, true)
	a.add(binary.X, pdg.Control, true)
	a.sources[site{sourceSpan(a.fset, binary.X), pdg.Control, true}] = binary
}

func (a *inventory) identifier(id *ast.Ident) {
	if a.writes[id] {
		return
	}
	switch a.info.Uses[id].(type) {
	case *types.Var:
		a.add(id, pdg.Read, false)
	case *types.Const:
		a.add(id, pdg.Computation, false)
	}
}

func (a *inventory) compare(g *pdg.Graph) error {
	for _, node := range g.Nodes {
		if err := a.consume(g, node); err != nil {
			return err
		}
	}
	for key, count := range a.expected {
		if count != 0 {
			return fmt.Errorf("operation inventory: missing %d category %d occurrences at bytes %d:%d", count, key.category, key.span.Start, key.span.End)
		}
	}
	return nil
}

func (a *inventory) consume(g *pdg.Graph, node pdg.Node) error {
	if node.Span == 0 {
		return fmt.Errorf("operation inventory: missing source span")
	}
	key := site{g.Spans[node.Span-1], node.Category, node.Synthetic}
	if a.expected[key] > 0 {
		a.expected[key]--
		return a.operationFacts(g, node, a.sources[key])
	}
	if derivedBindingJoin(g, node) {
		return nil
	}
	return fmt.Errorf("operation inventory: unexpected category %d at bytes %d:%d", node.Category, key.span.Start, key.span.End)
}

func unparen(expression ast.Expr) ast.Expr {
	for {
		paren, ok := expression.(*ast.ParenExpr)
		if !ok {
			return expression
		}
		expression = paren.X
	}
}

func (a *inventory) target(expression ast.Expr) {
	for {
		a.writes[expression] = true
		paren, ok := expression.(*ast.ParenExpr)
		if !ok {
			return
		}
		expression = paren.X
	}
}

func derivedBindingJoin(g *pdg.Graph, node pdg.Node) bool {
	attrs := g.NodeAttributes(node)
	return node.Category == pdg.Merge && node.Synthetic && len(attrs.Defines) == 1 && len(attrs.Symbols) == 1
}
