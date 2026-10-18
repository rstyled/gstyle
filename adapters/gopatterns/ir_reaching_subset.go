package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
)

func (b *irBuilder) reachingSubset(body *ast.BlockStmt) bool {
	return b.reachingSourceSubset(body, false)
}

func (b *irBuilder) extendedReachingSubset(body *ast.BlockStmt) bool {
	return b.reachingSourceSubset(body, true)
}

func (b *irBuilder) reachingSourceSubset(body *ast.BlockStmt, extended bool) bool {
	for object := range b.symbols {
		if !scalarBinding(object) {
			return false
		}
	}
	supported := true
	ast.Inspect(body, func(node ast.Node) bool {
		if !supported {
			return false
		}
		supported = reachingSource(node, extended) && b.scalarSourceBinding(node) && b.scalarSourceType(node)
		return supported
	})
	return supported
}

func (b *irBuilder) scalarSourceBinding(node ast.Node) bool {
	id, ok := node.(*ast.Ident)
	if !ok {
		return true
	}
	object := b.info.ObjectOf(id)
	if _, ok := object.(*types.Var); !ok {
		return true
	}
	return scalarBinding(object)
}

func (b *irBuilder) scalarSourceType(node ast.Node) bool {
	if declaration, ok := node.(*ast.GenDecl); ok {
		return declaration.Tok == token.VAR
	}
	expression, ok := node.(ast.Expr)
	if !ok {
		return true
	}
	value, ok := b.info.Types[expression]
	if !ok || value.IsType() {
		return true
	}
	basic, ok := value.Type.Underlying().(*types.Basic)
	return ok && basic.Info()&(types.IsInteger|types.IsBoolean) != 0
}

func scalarBinding(object types.Object) bool {
	variable, ok := object.(*types.Var)
	if !ok || variable.IsField() {
		return false
	}
	if variable.Pkg() != nil && variable.Parent() == variable.Pkg().Scope() {
		return false
	}
	basic, ok := variable.Type().Underlying().(*types.Basic)
	return ok && basic.Info()&(types.IsInteger|types.IsBoolean) != 0
}

func scalarSource(node ast.Node) bool {
	if statement, ok := node.(ast.Stmt); ok {
		return scalarStatement(statement)
	}
	switch node := node.(type) {
	case *ast.CallExpr, *ast.FuncLit, *ast.SelectorExpr, *ast.IndexExpr, *ast.IndexListExpr, *ast.SliceExpr, *ast.StarExpr, *ast.CompositeLit, *ast.TypeAssertExpr:
		return false
	default:
		return scalarExpression(node)
	}
}

func scalarExpression(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.UnaryExpr:
		return node.Op != token.AND && node.Op != token.ARROW
	default:
		return true
	}
}

func scalarStatement(statement ast.Stmt) bool {
	switch statement.(type) {
	case *ast.BlockStmt, *ast.EmptyStmt, *ast.DeclStmt, *ast.AssignStmt, *ast.IncDecStmt, *ast.IfStmt, *ast.ReturnStmt:
		return true
	default:
		return false
	}
}

// Extended scalar flow admits local loops and transfers, with no memory or call
// effects. It is separate from the frozen acyclic matching profile.
func reachingSource(node ast.Node, extended bool) bool {
	if !extended {
		return scalarSource(node)
	}
	switch node.(type) {
	case *ast.ForStmt, *ast.BranchStmt, *ast.LabeledStmt, *ast.SwitchStmt, *ast.CaseClause:
		return true
	default:
		return scalarSource(node)
	}
}
