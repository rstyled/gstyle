package govalidation

import (
	"fmt"
	"go/ast"
	"go/token"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

type sourceFlow struct {
	fset     *token.FileSet
	nodes    map[site]pdg.Ref
	next     map[pdg.Ref][]pdg.Ref
	branches map[pdg.Ref][2][]pdg.Ref
	err      error
}

func (a *sourceFlow) ref(node ast.Node, category pdg.Category, synthetic bool) pdg.Ref {
	span := pdg.Span{Source: 1, Start: uint32(a.fset.PositionFor(node.Pos(), false).Offset), End: uint32(a.fset.PositionFor(node.End(), false).Offset)}
	ref := a.nodes[site{span, category, synthetic}]
	if ref == 0 {
		a.err = fmt.Errorf("source CFG operation is missing at bytes %d:%d", span.Start, span.End)
	}
	return ref
}

func (a *sourceFlow) operation(node ast.Node, category pdg.Category, synthetic bool, next []pdg.Ref) []pdg.Ref {
	ref := a.ref(node, category, synthetic)
	a.next[ref] = next
	return []pdg.Ref{ref}
}

func (a *sourceFlow) statements(statements []ast.Stmt, next []pdg.Ref) []pdg.Ref {
	for i := len(statements) - 1; i >= 0; i-- {
		next = a.statement(statements[i], next)
	}
	return next
}

func (a *sourceFlow) statement(statement ast.Stmt, next []pdg.Ref) []pdg.Ref {
	switch node := statement.(type) {
	case *ast.BlockStmt:
		return a.statements(node.List, next)
	case *ast.IfStmt:
		return a.branch(node, next)
	case *ast.ReturnStmt:
		return a.returnFlow(node)
	case *ast.AssignStmt:
		return a.assignFlow(node, next)
	default:
		return a.simpleStatement(statement, next)
	}
}

func (a *sourceFlow) simpleStatement(statement ast.Stmt, next []pdg.Ref) []pdg.Ref {
	switch node := statement.(type) {
	case *ast.EmptyStmt:
		return next
	case *ast.DeclStmt:
		return a.declarations(node, next)
	case *ast.IncDecStmt:
		next = a.operation(node, pdg.Assignment, false, next)
		return a.operation(unparen(node.X), pdg.Read, true, next)
	default:
		a.err = fmt.Errorf("unsupported source CFG statement %T", statement)
		return nil
	}
}

func (a *sourceFlow) branch(node *ast.IfStmt, next []pdg.Ref) []pdg.Ref {
	left := a.statements(node.Body.List, next)
	right := next
	if node.Else != nil {
		right = a.statement(node.Else, next)
	}
	predicate := a.operation(node, pdg.Control, false, append(slices.Clone(left), right...))
	a.branches[predicate[0]] = [2][]pdg.Ref{left, right}
	entry := a.expression(node.Cond, predicate)
	if node.Init != nil {
		entry = a.statement(node.Init, entry)
	}
	return entry
}

func (a *sourceFlow) assignFlow(node *ast.AssignStmt, next []pdg.Ref) []pdg.Ref {
	if len(node.Rhs) != 1 {
		a.err = fmt.Errorf("ambiguous assignment evaluation order")
		return nil
	}
	category := pdg.Assignment
	if node.Tok == token.DEFINE {
		category = pdg.Declaration
	}
	if node.Tok != token.DEFINE && node.Tok != token.ASSIGN {
		a.err = fmt.Errorf("compound evaluation order is not admitted")
		return nil
	}
	return a.expression(node.Rhs[0], a.operation(node, category, false, next))
}

func (a *sourceFlow) returnFlow(node *ast.ReturnStmt) []pdg.Ref {
	next := a.operation(node, pdg.Return, false, nil)
	if len(node.Results) > 1 {
		a.err = fmt.Errorf("multiple-result order is not admitted")
		return nil
	}
	if len(node.Results) == 1 {
		return a.expression(node.Results[0], next)
	}
	span := pdg.Span{Source: 1, Start: uint32(a.fset.PositionFor(node.Pos(), false).Offset), End: uint32(a.fset.PositionFor(node.End(), false).Offset)}
	if a.nodes[site{span, pdg.Read, true}] != 0 {
		return a.operation(node, pdg.Read, true, next)
	}
	return next
}

func (a *sourceFlow) declarations(node *ast.DeclStmt, next []pdg.Ref) []pdg.Ref {
	declaration, ok := node.Decl.(*ast.GenDecl)
	if !ok || declaration.Tok != token.VAR {
		a.err = fmt.Errorf("unsupported declaration")
		return nil
	}
	for i := len(declaration.Specs) - 1; i >= 0; i-- {
		next = a.valueSpec(declaration.Specs[i], next)
	}
	return next
}

func (a *sourceFlow) valueSpec(spec ast.Spec, next []pdg.Ref) []pdg.Ref {
	value, ok := spec.(*ast.ValueSpec)
	if !ok || len(value.Values) > 1 {
		a.err = fmt.Errorf("ambiguous declaration evaluation order")
		return nil
	}
	next = a.operation(value, pdg.Declaration, false, next)
	if len(value.Values) == 1 {
		next = a.expression(value.Values[0], next)
	}
	return next
}

func (a *sourceFlow) expression(expression ast.Expr, next []pdg.Ref) []pdg.Ref {
	switch node := expression.(type) {
	case *ast.ParenExpr:
		return a.expression(node.X, a.operation(node, pdg.Computation, false, next))
	case *ast.UnaryExpr:
		return a.expression(node.X, a.operation(node, pdg.Computation, false, next))
	case *ast.BinaryExpr:
		return a.logical(node, next)
	case *ast.BasicLit:
		return a.operation(node, pdg.Computation, false, next)
	default:
		return a.atomicExpression(expression, next)
	}
}

func (a *sourceFlow) atomicExpression(expression ast.Expr, next []pdg.Ref) []pdg.Ref {
	span := pdg.Span{Source: 1, Start: uint32(a.fset.PositionFor(expression.Pos(), false).Offset), End: uint32(a.fset.PositionFor(expression.End(), false).Offset)}
	category := pdg.Read
	if a.nodes[site{span, pdg.Computation, false}] != 0 {
		category = pdg.Computation
	}
	return a.operation(expression, category, false, next)
}

func (a *sourceFlow) logical(node *ast.BinaryExpr, next []pdg.Ref) []pdg.Ref {
	if node.Op != token.LAND && node.Op != token.LOR {
		a.err = fmt.Errorf("unspecified operand order is not admitted")
		return nil
	}
	right := a.expression(node.Y, next)
	branches := append(slices.Clone(right), next...)
	predicate := a.operation(node.X, pdg.Control, true, branches)
	a.branches[predicate[0]] = [2][]pdg.Ref{right, next}
	if node.Op == token.LOR {
		a.branches[predicate[0]] = [2][]pdg.Ref{next, right}
	}
	return a.expression(node.X, predicate)
}

// CFG checks operation transitions and entry correspondence against a separate
// structured-source walk. Empty compiler blocks are navigation only. This check
// accepts only the scalar source subset with known evaluation order.
func CFG(g *pdg.Graph, fn *ast.FuncDecl, fset *token.FileSet) error {
	if err := pdg.Validate(g); err != nil {
		return err
	}
	if g.NormalFlow == nil {
		return fmt.Errorf("paired normal CFG is missing")
	}
	nodes, err := sourceSites(g)
	if err != nil {
		return err
	}
	a := sourceFlow{fset: fset, nodes: nodes, next: map[pdg.Ref][]pdg.Ref{}, branches: map[pdg.Ref][2][]pdg.Ref{}}
	entry := a.statements(fn.Body.List, a.implicitReturn(fn))
	if a.err != nil {
		return a.err
	}
	if err := compareSourceFlow(g.NormalFlow, entry, a.next); err != nil {
		return err
	}
	return compareSourceBranches(g.NormalFlow, a.branches)
}

func sourceSites(g *pdg.Graph) (map[site]pdg.Ref, error) {
	nodes := map[site]pdg.Ref{}
	for i, node := range g.Nodes {
		if node.Span == 0 {
			return nil, fmt.Errorf("operation has no source span")
		}
		nodes[site{g.Spans[node.Span-1], node.Category, node.Synthetic}] = pdg.Ref(i + 1)
	}
	return nodes, nil
}
