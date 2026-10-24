package goquality

import (
	"go/ast"
	"go/types"
)

// A copied value record can replace a borrowed field before using it. Require
// one complete header replacement after construction, in the same unlabeled
// block as the use's enclosing statement. The caller still checks ownership
// of every replacement and excludes root escapes, aliases and deferred work.
func (x *extractor) replacedValueField(key fieldKey, initializers []ast.Expr, current ast.Node) bool {
	if _, value := underlying(key.root.Type()).(*types.Struct); !value {
		return false
	}
	binding := x.fields.writes[key]
	if len(initializers) != 1 || len(binding.values) != 1 || binding.selfReturned {
		return false
	}
	return x.fieldReplacementPrecedes(initializers[0], binding.values[0], current)
}

func (x *extractor) fieldReplacementPrecedes(initial, value ast.Expr, current ast.Node) bool {
	initializer, block := x.unlabeledFieldBlock(initial)
	if block == nil {
		return false
	}
	replacement, replacementBlock := x.fieldStatement(value, nil)
	if replacementBlock != block || initializer.End() >= replacement.Pos() {
		return false
	}
	statement, _ := x.fieldStatement(current, block)
	return statement != nil && replacement.End() < statement.Pos()
}

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
