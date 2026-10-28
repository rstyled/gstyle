package gopatterns

import (
	"go/ast"
	"go/token"
)

// scalarOrderKnown checks the whole body, including unreachable syntax. The
// profile rejects independently evaluated sibling operands rather than treating
// the producer's traversal order as a language ordering guarantee.
func scalarOrderKnown(body *ast.BlockStmt) bool {
	known := true
	ast.Inspect(body, func(node ast.Node) bool {
		if !known {
			return false
		}
		known = scalarOrderedSite(node)
		return known
	})
	return known
}

func scalarOrderedSite(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.BinaryExpr:
		return node.Op == token.LAND || node.Op == token.LOR
	case *ast.AssignStmt:
		return orderedAssignment(node)
	case *ast.ValueSpec:
		return orderedDeclaration(node)
	case *ast.ReturnStmt:
		return len(node.Results) <= 1
	default:
		return true
	}
}

func orderedDeclaration(node *ast.ValueSpec) bool {
	if len(node.Values) == 0 {
		return true
	}
	return len(node.Values) <= 1 && len(node.Names) <= 1
}

func orderedAssignment(node *ast.AssignStmt) bool {
	if len(node.Rhs) > 1 {
		return false
	}
	return node.Tok == token.ASSIGN || node.Tok == token.DEFINE
}
