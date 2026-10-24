package goquality

import (
	"go/ast"
	"go/token"
	"go/types"
)

// Range keys do not expose rows. A newly declared row used only by builtin
// len/cap also cannot publish or change the row's storage or its children.
func (x *extractor) lengthOnlyRange(node, parent ast.Node) bool {
	statement, ok := parent.(*ast.RangeStmt)
	if !ok {
		return false
	}
	if statement.X != node {
		return false
	}
	if statement.Value == nil {
		return true
	}
	id, ok := ast.Unparen(statement.Value).(*ast.Ident)
	if !ok || statement.Tok != token.DEFINE {
		return false
	}
	return x.lengthOnlyUses(x.pkg.TypesInfo.ObjectOf(id))
}

func (x *extractor) lengthOnlyUses(object types.Object) bool {
	if object == nil {
		return false
	}
	proof := receiverProof{parents: x.fields.parents}
	for _, use := range x.fields.uses[object] {
		node, parent := proof.parent(use)
		call, ok := parent.(*ast.CallExpr)
		if !ok || proof.function(use) != nil || !tableLength(call, node, x.pkg.TypesInfo) {
			return false
		}
	}
	return true
}
