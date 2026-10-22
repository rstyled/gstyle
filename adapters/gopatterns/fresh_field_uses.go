package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
)

func (q *assertionGuardQuery) freshFieldUses(base *ast.Ident, field *types.Var) bool {
	object := q.info.ObjectOf(base)
	parents := assertionParents(q.body)
	for ident, used := range q.info.Uses {
		if used == object && (assertionClosureUse(ident, parents) || !q.freshFieldUse(ident, field, parents)) {
			return false
		}
	}
	return true
}

func (q *assertionGuardQuery) freshFieldUse(ident *ast.Ident, field *types.Var, parents map[ast.Node]ast.Node) bool {
	parent := assertionParent(ident, parents)
	if _, returned := parent.(*ast.ReturnStmt); returned {
		return true
	}
	selector, ok := parent.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	selection := q.info.Selections[selector]
	if selection == nil || selection.Kind() != types.FieldVal {
		return false
	}
	return selection.Obj() != field || !freshFieldTarget(selector, assertionParent(selector, parents))
}

func assertionParents(body *ast.BlockStmt) map[ast.Node]ast.Node {
	parents := map[ast.Node]ast.Node{}
	var stack []ast.Node
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	return parents
}

func assertionParent(node ast.Node, parents map[ast.Node]ast.Node) ast.Node {
	for node = parents[node]; node != nil; node = parents[node] {
		if _, wrapped := node.(*ast.ParenExpr); !wrapped {
			return node
		}
	}
	return nil
}

func freshFieldTarget(selector *ast.SelectorExpr, parent ast.Node) bool {
	switch parent := parent.(type) {
	case *ast.AssignStmt:
		return assertionExpressionListed(selector, parent.Lhs)
	case *ast.RangeStmt:
		return ast.Unparen(parent.Key) == selector || ast.Unparen(parent.Value) == selector
	case *ast.IncDecStmt:
		return true
	case *ast.UnaryExpr:
		return parent.Op == token.AND
	default:
		return false
	}
}

func assertionExpressionListed(expression ast.Expr, list []ast.Expr) bool {
	for _, entry := range list {
		if ast.Unparen(entry) == expression {
			return true
		}
	}
	return false
}

func assertionClosureUse(node ast.Node, parents map[ast.Node]ast.Node) bool {
	for node = parents[node]; node != nil; node = parents[node] {
		if _, closure := node.(*ast.FuncLit); closure {
			return true
		}
	}
	return false
}
