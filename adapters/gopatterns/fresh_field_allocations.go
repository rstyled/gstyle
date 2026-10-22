package gopatterns

import (
	"go/ast"
	"go/token"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (q *assertionGuardQuery) freshLiteral(base *ast.Ident) *ast.CompositeLit {
	origins := q.bindings.OriginsOf(q.refs[base], q.facts)
	ref := freshAllocationOrigin(origins)
	return q.allocationLiteral(ref)
}

func freshAllocationOrigin(origins bindingflow.Origins) pdg.Ref {
	if !origins.Known || len(origins.Sources) != 1 {
		return 0
	}
	origin := origins.Sources[0]
	if origin.Zero || origin.Representation != bindingflow.UnchangedRepresentation {
		return 0
	}
	return origin.Node
}

func (q *assertionGuardQuery) allocationLiteral(ref pdg.Ref) *ast.CompositeLit {
	address := q.canonicalAddress(ref)
	if address == nil {
		return nil
	}
	literal, ok := ast.Unparen(address.X).(*ast.CompositeLit)
	if !ok || q.bindings.Operand(ref, 1) != q.refs[address.X] {
		return nil
	}
	if !q.bindings.IsAllocation(q.refs[literal]) {
		return nil
	}
	return literal
}

func (q *assertionGuardQuery) canonicalAddress(ref pdg.Ref) *ast.UnaryExpr {
	address, ok := q.source[ref].(*ast.UnaryExpr)
	if !ok || address.Op != token.AND {
		return nil
	}
	attrs := q.graph.NodeAttributes(q.graph.Nodes[ref-1])
	if attrs.Operator.Status != pdg.Known || q.graph.Tables.Text(attrs.Operator.Value) != "&" {
		return nil
	}
	return address
}
