package gopatterns

import (
	"go/ast"
	"go/token"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (b *irBuilder) readPosition(owner pdg.Ref, child ast.Node) pdg.Position {
	for i := len(b.stack) - 1; i >= 0; i-- {
		parent := b.stack[i]
		if b.nodes[parent] == owner {
			return sourceOperandPosition(parent, child)
		}
		child = parent
	}
	return 0
}

func sourceOperandPosition(parent, child ast.Node) pdg.Position {
	switch parent := parent.(type) {
	case *ast.ValueSpec:
		return operandIndex(parent.Values, child)
	case ast.Stmt:
		return statementOperandPosition(parent, child)
	case ast.Expr:
		return expressionOperandPosition(parent, child)
	default:
		return 0
	}
}

func statementOperandPosition(parent ast.Stmt, child ast.Node) pdg.Position {
	switch parent := parent.(type) {
	case *ast.AssignStmt:
		return assignmentOperandPosition(parent, child)
	case *ast.ReturnStmt:
		return operandIndex(parent.Results, child)
	case *ast.IfStmt:
		return operandIndex([]ast.Expr{parent.Cond}, child)
	default:
		return 0
	}
}

func assignmentOperandPosition(parent *ast.AssignStmt, child ast.Node) pdg.Position {
	position := operandIndex(parent.Rhs, child)
	if position == 0 {
		return 0
	}
	if parent.Tok == token.ASSIGN || parent.Tok == token.DEFINE {
		return position
	}
	// A compound operation reads its old target in slot zero, then the RHS.
	return position + 1
}

func expressionOperandPosition(parent ast.Expr, child ast.Node) pdg.Position {
	switch parent := parent.(type) {
	case *ast.BinaryExpr:
		return operandIndex([]ast.Expr{parent.X, parent.Y}, child)
	case *ast.UnaryExpr:
		return operandIndex([]ast.Expr{parent.X}, child)
	case *ast.ParenExpr:
		return operandIndex([]ast.Expr{parent.X}, child)
	default:
		return aggregateOperandPosition(parent, child)
	}
}

func aggregateOperandPosition(parent ast.Expr, child ast.Node) pdg.Position {
	switch parent := parent.(type) {
	case *ast.CompositeLit:
		return operandIndex(parent.Elts, child)
	case *ast.KeyValueExpr:
		return operandIndex([]ast.Expr{parent.Key, parent.Value}, child)
	case *ast.SliceExpr:
		return operandIndex([]ast.Expr{parent.X, parent.Low, parent.High, parent.Max}, child)
	case *ast.TypeAssertExpr:
		return operandIndex([]ast.Expr{parent.X}, child)
	default:
		return 0
	}
}

func operandIndex(operands []ast.Expr, child ast.Node) pdg.Position {
	for i, operand := range operands {
		if operand == child {
			return pdg.Position(i + 1)
		}
	}
	return 0
}
