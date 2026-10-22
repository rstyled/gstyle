package goquality

import (
	"go/ast"
	"go/token"
	"go/types"
)

// No-escape checks are separate from return identity. They prevent a fresh
// result or an identity helper's private input from acquiring published aliases.
func (x *extractor) pointerObjectClosed(object types.Object, seen map[types.Object]bool) bool {
	if object == nil || packageVariable(object) || seen[object] {
		return false
	}
	seen[object] = true
	defer delete(seen, object)
	for _, use := range x.fields.uses[object] {
		if !x.pointerValueClosed(use, seen) {
			return false
		}
	}
	return true
}

func (x *extractor) pointerInputClosed(value ast.Expr) bool {
	seen := map[types.Object]bool{}
	switch value := ast.Unparen(value).(type) {
	case *ast.Ident:
		return x.pointerObjectClosed(x.pkg.TypesInfo.ObjectOf(value), seen)
	case *ast.UnaryExpr:
		if value.Op == token.AND {
			object, _ := x.placeRoot(value.X)
			if object != nil {
				return x.pointerObjectClosed(object, seen)
			}
		}
	}
	return x.pointerValueClosed(value, seen)
}

func (x *extractor) pointerValueClosed(value ast.Node, seen map[types.Object]bool) bool {
	proof := receiverProof{parents: x.fields.parents}
	if _, captured := proof.function(value).(*ast.FuncLit); captured {
		return false
	}
	node, parent := proof.parent(value)
	switch parent := parent.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.AssignStmt:
		return x.pointerAssignment(node, parent, seen)
	case *ast.CallExpr:
		return x.pointerForwardedUse(node, parent, seen)
	default:
		return x.pointerScalarUse(parent)
	}
}

// Only the callee's certified non-publishing formal can forward this pointer.
func (x *extractor) pointerForwardedUse(value ast.Node, call *ast.CallExpr, seen map[types.Object]bool) bool {
	return x.synchronousContainerCall(call) && ast.Unparen(x.pointerReturnArgument(call)) == value && x.pointerValueClosed(call, seen)
}

func (x *extractor) pointerAssignment(value ast.Node, statement *ast.AssignStmt, seen map[types.Object]bool) bool {
	if len(statement.Lhs) != 1 || len(statement.Rhs) != 1 || ast.Unparen(statement.Rhs[0]) != value {
		return false
	}
	id, ok := ast.Unparen(statement.Lhs[0]).(*ast.Ident)
	if !ok {
		return false
	}
	if id.Name == "_" {
		return true
	}
	return x.pointerObjectClosed(x.pkg.TypesInfo.ObjectOf(id), seen)
}

func (x *extractor) pointerScalarUse(parent ast.Node) bool {
	switch parent := parent.(type) {
	case *ast.StarExpr:
		return x.pointerScalarContext(parent)
	case *ast.SelectorExpr:
		return !embeddedPointer(x.pkg.TypesInfo.Selections[parent]) && x.pointerScalarContext(parent)
	case *ast.BinaryExpr:
		return parent.Op == token.EQL || parent.Op == token.NEQ
	default:
		return false
	}
}

func (x *extractor) pointerScalarContext(value ast.Expr) bool {
	if _, scalar := underlying(x.pkg.TypesInfo.TypeOf(value)).(*types.Basic); !scalar {
		return false
	}
	proof := receiverFields{receiverProof: receiverProof{pkg: x.pkg, parents: x.fields.parents}}
	return proof.elementUse(value)
}
