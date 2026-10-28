package govalidation

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/rstyled/gstyle/domain/pdg"
)

func implicitReturnSite(fn *ast.FuncDecl, fset *token.FileSet) (site, bool) {
	if fn.Type.Results != nil && fn.Type.Results.NumFields() != 0 {
		return site{}, false
	}
	if !statementFallsThrough(fn.Body) {
		return site{}, false
	}
	offset := uint32(fset.PositionFor(fn.Body.Rbrace, false).Offset)
	return site{span: pdg.Span{Source: 1, Start: offset, End: offset}, category: pdg.Return, synthetic: true}, true
}

func statementFallsThrough(statement ast.Stmt) bool {
	switch node := statement.(type) {
	case *ast.ReturnStmt:
		return false
	case *ast.BlockStmt:
		return statementsFallThrough(node.List)
	case *ast.IfStmt:
		return node.Else == nil || statementFallsThrough(node.Body) || statementFallsThrough(node.Else)
	default:
		return true
	}
}

func statementsFallThrough(statements []ast.Stmt) bool {
	for _, statement := range statements {
		if !statementFallsThrough(statement) {
			return false
		}
	}
	return true
}

func (a *inventory) implicitReturn(fn *ast.FuncDecl) {
	key, expected := implicitReturnSite(fn, a.fset)
	if !expected {
		return
	}
	a.expected[key] = 1
	a.sources[key] = fn.Body
}

func implicitReturnFacts(g *pdg.Graph, node pdg.Node) error {
	attrs := g.NodeAttributes(node)
	if g.Tables.Text(attrs.Construct) != "implicit-return" || attrs.ExitKind != pdg.NormalExit {
		return fmt.Errorf("void fallthrough must be an implicit normal return")
	}
	return nil
}

func (a *sourceFlow) implicitReturn(fn *ast.FuncDecl) []pdg.Ref {
	key, expected := implicitReturnSite(fn, a.fset)
	if !expected {
		return nil
	}
	ref := a.nodes[key]
	if ref == 0 {
		a.err = fmt.Errorf("void fallthrough return is missing from the operation inventory")
		return nil
	}
	a.next[ref] = nil
	return []pdg.Ref{ref}
}
