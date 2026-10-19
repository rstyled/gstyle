package goquality

import (
	"go/ast"
	"go/types"
)

// A fresh root stays private until a later statement publishes it. Statements
// within the same loop/branch are not ordered against one another by this proof.
func (x *extractor) rootClosedAt(root types.Object, initializers []ast.Expr, current ast.Node) bool {
	if len(initializers) != 1 || !x.outerFieldReturn(current) {
		return false
	}
	initializer, block := x.unlabeledFieldBlock(initializers[0])
	if block == nil {
		return false
	}
	statement, _ := x.fieldStatement(current, block)
	if statement == nil || statement.Pos() <= initializer.Pos() {
		return false
	}
	return x.fieldUsesAfter(root, block, statement)
}

func (x *extractor) unlabeledFieldBlock(node ast.Node) (ast.Stmt, *ast.BlockStmt) {
	statement, block := x.fieldStatement(node, nil)
	if block == nil || labeledBlock(block) {
		return nil, nil
	}
	return statement, block
}

func (x *extractor) fieldUsesAfter(root types.Object, block *ast.BlockStmt, current ast.Stmt) bool {
	for _, use := range x.fields.uses[root] {
		statement, _ := x.fieldStatement(use, block)
		if statement == nil {
			return false
		}
		if !x.closedRootUse(use) && statement.Pos() <= current.Pos() {
			return false
		}
	}
	return true
}

// Find the statement directly in the requested block; reject closure contexts.
func (x *extractor) fieldStatement(node ast.Node, wanted *ast.BlockStmt) (ast.Stmt, *ast.BlockStmt) {
	for node != nil {
		parent := x.fields.parents[node]
		if _, closure := parent.(*ast.FuncLit); closure {
			return nil, nil
		}
		if block, ok := parent.(*ast.BlockStmt); ok && (wanted == nil || block == wanted) {
			statement, _ := node.(ast.Stmt)
			return statement, block
		}
		node = parent
	}
	return nil, nil
}

func labeledBlock(block *ast.BlockStmt) bool {
	labeled := false
	ast.Inspect(block, func(node ast.Node) bool {
		if _, ok := node.(*ast.LabeledStmt); ok {
			labeled = true
		}
		return !labeled
	})
	return labeled
}
