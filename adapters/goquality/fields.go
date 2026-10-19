package goquality

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/quality"
)

type fieldKey struct {
	root  types.Object
	field types.Object
}

// Field ownership is a per-function proof, separate from bare-variable bindings.
type fieldOwnership struct {
	parents  map[ast.Node]ast.Node
	writes   map[fieldKey]referenceBinding
	roots    map[types.Object]bool
	uses     map[types.Object][]*ast.Ident
	deferred bool
}

func (x *extractor) directField(value ast.Expr) (fieldKey, bool) {
	selector, ok := ast.Unparen(value).(*ast.SelectorExpr)
	if !ok {
		return fieldKey{}, false
	}
	selection := x.pkg.TypesInfo.Selections[selector]
	if selection == nil || selection.Kind() != types.FieldVal || len(selection.Index()) != 1 {
		return fieldKey{}, false
	}
	id, ok := ast.Unparen(selector.X).(*ast.Ident)
	if !ok {
		return fieldKey{}, false
	}
	return fieldKey{root: x.pkg.TypesInfo.ObjectOf(id), field: selection.Obj()}, true
}

// Index all header writes before evaluating provenance, including later writes
// and writes in closures. No source-order assumption establishes ownership.
func (x *extractor) collectFieldWrites(body *ast.BlockStmt) {
	x.fields = fieldOwnership{parents: map[ast.Node]ast.Node{}, writes: map[fieldKey]referenceBinding{}, roots: map[types.Object]bool{}, uses: map[types.Object][]*ast.Ident{}}
	indexParents(body, x.fields.parents)
	ast.Inspect(body, x.collectFieldNode)
	for object, uses := range x.fields.uses {
		x.fields.roots[object] = x.closedRootUses(uses)
	}
}

func (x *extractor) bindField(place, value ast.Expr) {
	key, ok := x.directField(place)
	if !ok {
		return
	}
	binding := x.fields.writes[key]
	if value == nil {
		binding.unknown = true
	} else if x.fieldSelfReturn(key, value) {
		binding.selfReturned = true
	} else {
		binding.values = append(binding.values, value)
	}
	x.fields.writes[key] = binding
}

func (x *extractor) fieldSelfReturn(key fieldKey, value ast.Expr) bool {
	call, ok := ast.Unparen(value).(*ast.CallExpr)
	if !ok || !x.selfContainerCall(call) {
		return false
	}
	target, ok := x.directField(x.containerArgs(call)[0])
	return ok && target == key
}

func (x *extractor) ownedField(value *ast.SelectorExpr, seen map[types.Object]bool) quality.Provenance {
	key, ok := x.directField(value)
	if !ok {
		return quality.Unknown
	}
	root, ok := x.bindings[key.root]
	if !ok || root.unknown || len(root.values) == 0 {
		return quality.Unknown
	}
	return x.fieldBindingOrigin(key, root.values, value, seen)
}

func (x *extractor) fieldBindingOrigin(key fieldKey, initializers []ast.Expr, current ast.Node, seen map[types.Object]bool) quality.Provenance {
	if seen[key.root] || (!x.fields.roots[key.root] && !x.rootClosedAt(key.root, initializers, current)) {
		return quality.Unknown
	}
	seen[key.root] = true
	defer delete(seen, key.root)
	if x.bindingOrigin(x.fields.writes[key], seen) != quality.Local {
		return quality.Unknown
	}
	return x.initializedFieldOrigin(key, initializers, seen)
}

func (x *extractor) initializedFieldOrigin(key fieldKey, initializers []ast.Expr, seen map[types.Object]bool) quality.Provenance {
	for _, initializer := range initializers {
		field, ok := x.literalField(initializer, key.field)
		if !ok || x.origin(field, seen) != quality.Local {
			return quality.Unknown
		}
	}
	return quality.Local
}

func (x *extractor) literalField(value ast.Expr, field types.Object) (ast.Expr, bool) {
	if call, ok := ast.Unparen(value).(*ast.CallExpr); ok {
		function, _ := x.calledObject(unwrappedCallee(call.Fun)).(*types.Func)
		value = x.factories[function]
	}
	value = ast.Unparen(value)
	if address, ok := value.(*ast.UnaryExpr); ok && address.Op == token.AND {
		value = ast.Unparen(address.X)
	}
	literal, ok := value.(*ast.CompositeLit)
	if !ok {
		return nil, false
	}
	shape, ok := underlying(x.pkg.TypesInfo.TypeOf(literal)).(*types.Struct)
	if !ok {
		return nil, false
	}
	return x.literalFieldValue(literal, shape, field), true
}

func (x *extractor) literalFieldValue(literal *ast.CompositeLit, shape *types.Struct, field types.Object) ast.Expr {
	for i, element := range literal.Elts {
		if pair, ok := element.(*ast.KeyValueExpr); ok {
			id, _ := pair.Key.(*ast.Ident)
			if x.pkg.TypesInfo.ObjectOf(id) == field {
				return pair.Value
			}
		} else if shape.Field(i) == field {
			return element
		}
	}
	return nil // An omitted field has its zero value.
}

// A bare alias or call can replace any field through another reference. Direct
// field operations are checked separately; returning the result ends construction.
func (x *extractor) closedRootUses(uses []*ast.Ident) bool {
	for _, id := range uses {
		if !x.closedRootUse(id) {
			return false
		}
	}
	return true
}

func (x *extractor) closedRootUse(id *ast.Ident) bool {
	var node ast.Node = id
	for parent := x.fields.parents[node]; parent != nil; parent = x.fields.parents[node] {
		switch parent := parent.(type) {
		case *ast.ParenExpr:
			node = parent
			continue
		case *ast.SelectorExpr:
			return x.closedFieldUse(parent)
		case *ast.ReturnStmt:
			return x.outerFieldReturn(parent)
		case *ast.AssignStmt:
			return assignmentPlace(node, parent)
		}
		return false
	}
	return false
}

func (x *extractor) closedFieldUse(selector *ast.SelectorExpr) bool {
	if _, ok := x.directField(selector); !ok {
		return false
	}
	var node ast.Node = selector
	for parent := x.fields.parents[node]; parent != nil; parent = x.fields.parents[node] {
		switch parent := parent.(type) {
		case *ast.ParenExpr:
			node = parent
			continue
		case *ast.UnaryExpr:
			return parent.Op != token.AND
		case *ast.SelectorExpr:
			return false // Methods can implicitly take the field's address.
		}
		return true
	}
	return true
}

// A closure result can publish the root while construction is still running.
func (x *extractor) outerFieldReturn(node ast.Node) bool {
	if x.fields.deferred {
		return false
	}
	for parent := x.fields.parents[node]; parent != nil; parent = x.fields.parents[parent] {
		if _, closure := parent.(*ast.FuncLit); closure {
			return false
		}
	}
	return true
}

func (x *extractor) collectFieldNode(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.DeferStmt:
		x.fields.deferred = true
	case *ast.Ident:
		object := x.pkg.TypesInfo.Uses[node]
		x.fields.uses[object] = append(x.fields.uses[object], node)
	case *ast.AssignStmt:
		for i, place := range node.Lhs {
			x.bindField(place, assignmentValue(node.Rhs, i, len(node.Lhs)))
		}
	case *ast.RangeStmt:
		x.bindField(node.Key, nil)
		x.bindField(node.Value, nil)
	}
	return true
}
