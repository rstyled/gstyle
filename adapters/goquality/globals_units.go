package goquality

import (
	"go/ast"
	"go/types"
)

// A concrete empty struct has one possible value, including when the binding
// is exported. Its address and pointer receiver still expose storage identity.
func unitGlobal(object types.Object) bool {
	variable, ok := object.(*types.Var)
	if !ok || !packageVariable(variable) {
		return false
	}
	shape, ok := variable.Type().Underlying().(*types.Struct)
	return ok && shape.NumFields() == 0
}

func unitGlobals(info *types.Info, stable map[types.Object]bool) {
	for _, object := range info.Uses {
		if unitGlobal(object) {
			stable[object] = true
		}
	}
}

func stableGlobalRead(id *ast.Ident, parents map[ast.Node]ast.Node, info *types.Info) bool {
	object := info.ObjectOf(id)
	if !unitGlobal(object) && !styleValueType(object.Type()) && !errorValueType(object.Type()) {
		return stableTableRead(id, parents, info)
	}
	var node ast.Node = id
	for parent := parents[node]; parent != nil; parent = parents[node] {
		if !unitValueContext(node, parent, info) {
			return false
		}
		node = parent
	}
	return true
}

func unitValueContext(node, parent ast.Node, info *types.Info) bool {
	selector, ok := parent.(*ast.SelectorExpr)
	if ok {
		return unitValueSelection(info.Selections[selector])
	}
	return tableValueContext(node, parent, info)
}

func unitValueSelection(selection *types.Selection) bool {
	if selection == nil || selection.Kind() == types.FieldVal {
		return true
	}
	signature, ok := selection.Obj().Type().(*types.Signature)
	if !ok {
		return false
	}
	_, pointer := underlying(signature.Recv().Type()).(*types.Pointer)
	return !pointer
}
