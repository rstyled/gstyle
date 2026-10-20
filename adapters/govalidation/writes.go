package govalidation

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

type sourceAccesses struct {
	fset     *token.FileSet
	info     *types.Info
	refs     map[*types.Var]pdg.Ref
	expected map[site][]pdg.Ref
}

func (a *sourceAccesses) source(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.AssignStmt:
		category := pdg.Assignment
		if node.Tok == token.DEFINE {
			category = pdg.Declaration
		}
		a.record(node, category, node.Lhs)
	case *ast.IncDecStmt:
		a.record(node, pdg.Assignment, []ast.Expr{node.X})
	case *ast.ValueSpec:
		var names []ast.Expr
		for _, name := range node.Names {
			names = append(names, name)
		}
		a.record(node, pdg.Declaration, names)
	}
	return true
}

func (a *sourceAccesses) record(node ast.Node, category pdg.Category, expressions []ast.Expr) {
	key := site{sourceSpan(a.fset, node), category, false}
	for _, expression := range expressions {
		id, ok := unparen(expression).(*ast.Ident)
		if !ok || id.Name == "_" {
			continue
		}
		variable, _ := a.info.ObjectOf(id).(*types.Var)
		a.expected[key] = append(a.expected[key], a.refs[variable])
	}
}

func (a *sourceAccesses) checkWrites(g *pdg.Graph, fn *ast.FuncDecl) error {
	ast.Inspect(fn.Body, a.source)
	for _, node := range g.Nodes {
		if node.Category != pdg.Assignment && node.Category != pdg.Declaration {
			continue
		}
		key := site{g.Spans[node.Span-1], node.Category, node.Synthetic}
		attrs := g.NodeAttributes(node)
		actual := make([]pdg.Ref, 0, len(attrs.Writes))
		for _, write := range attrs.Writes {
			actual = append(actual, write.Symbol)
		}
		if !slices.Equal(actual, a.expected[key]) {
			return fmt.Errorf("write binding disagrees with typed source")
		}
	}
	return nil
}
