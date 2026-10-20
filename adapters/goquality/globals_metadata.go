package goquality

import (
	"go/ast"
	"go/types"
)

func metadataType(typ types.Type, path, name string) bool {
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok {
		return false
	}
	object := named.Obj()
	return object.Pkg() != nil && object.Pkg().Path() == path && object.Name() == name
}

func regexpValue(typ types.Type) bool {
	pointer, ok := types.Unalias(typ).(*types.Pointer)
	return ok && metadataType(pointer.Elem(), "regexp", "Regexp")
}

func reflectTypeTable(typ types.Type) bool {
	switch typ := underlying(typ).(type) {
	case *types.Slice:
		return metadataType(typ.Elem(), "reflect", "Type")
	case *types.Array:
		return metadataType(typ.Elem(), "reflect", "Type")
	}
	return false
}

// Only audited constructors establish immutable metadata; interface types alone
// do not establish that a value has the standard library's implementation.
func stableMetadataInitializer(typ types.Type, value ast.Expr, info *types.Info) bool {
	if regexpValue(typ) {
		call := metadataCall(value, "regexp.MustCompile", info)
		return call != nil && len(call.Args) == 1 && info.Types[call.Args[0]].Value != nil
	}
	literal, ok := ast.Unparen(value).(*ast.CompositeLit)
	if !ok || !reflectTypeTable(typ) {
		return false
	}
	return reflectTypeElements(literal.Elts, info)
}

func metadataCall(value ast.Expr, name string, info *types.Info) *ast.CallExpr {
	call, ok := ast.Unparen(value).(*ast.CallExpr)
	if !ok {
		return nil
	}
	selector, ok := unwrappedCallee(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	function, ok := info.ObjectOf(selector.Sel).(*types.Func)
	if !ok || resolvedName(function) != name {
		return nil
	}
	return call
}

func reflectTypeElements(elements []ast.Expr, info *types.Info) bool {
	for _, element := range elements {
		if metadataCall(element, "reflect.TypeOf", info) == nil && metadataCall(element, "reflect.TypeFor", info) == nil {
			return false
		}
	}
	return true
}

func stableRegexpRead(id *ast.Ident, parents map[ast.Node]ast.Node, info *types.Info) bool {
	selector, ok := parents[id].(*ast.SelectorExpr)
	if !ok || selector.X != id {
		return false
	}
	selection := info.Selections[selector]
	if selection == nil {
		return false
	}
	call, ok := parents[selector].(*ast.CallExpr)
	return ok && call.Fun == selector && readonlyRegexpMethod(selection.Obj().Name())
}

// These calls return matches in the input or fresh result slices, never mutable
// regexp metadata. Copy and SubexpNames can expose shared internal storage.
func readonlyRegexpMethod(name string) bool {
	switch name {
	case "Match", "MatchString", "FindStringSubmatch", "FindStringSubmatchIndex", "FindSubmatch", "FindSubmatchIndex", "FindString", "Find", "FindIndex", "FindStringIndex", "String", "NumSubexp", "SubexpIndex", "LiteralPrefix", "Split":
		return true
	}
	return false
}
