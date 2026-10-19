package goquality

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
)

// Reject the selected JSON callbacks and open interfaces; visit each type once.
func plainJSONType(typ types.Type, seen map[types.Type]bool, method string) bool {
	if seen[typ] {
		return true
	}
	seen[typ] = true
	if named, ok := types.Unalias(typ).(*types.Named); ok {
		return plainJSONNamed(named, seen, method)
	}
	return plainJSONShape(underlying(typ), seen, method)
}

func plainJSONShape(typ types.Type, seen map[types.Type]bool, method string) bool {
	switch typ := typ.(type) {
	case *types.Basic:
		return true
	case *types.Map:
		return plainJSONType(typ.Key(), seen, method) && plainJSONType(typ.Elem(), seen, method)
	case *types.Struct:
		return plainJSONFields(typ, seen, method)
	case interface{ Elem() types.Type }:
		return plainJSONType(typ.Elem(), seen, method)
	default:
		return false
	}
}

func plainJSONFields(typ *types.Struct, seen map[types.Type]bool, method string) bool {
	for i := 0; i < typ.NumFields(); i++ {
		if !plainJSONType(typ.Field(i).Type(), seen, method) {
			return false
		}
	}
	return true
}

func plainJSONNamed(typ *types.Named, seen map[types.Type]bool, method string) bool {
	for i := 0; i < typ.NumMethods(); i++ {
		if slices.Contains([]string{method + "JSON", method + "Text"}, typ.Method(i).Name()) {
			return false
		}
	}
	return plainJSONType(typ.Underlying(), seen, method)
}

// Decode only into an unaliased zero-value local whose other uses return it.
// The proof belongs to this call; it is not forwarded from caller storage.
func (x *extractor) localJSONDecode(call *ast.CallExpr, callee string) bool {
	if callee != "encoding/json.Unmarshal" {
		return false
	}
	address, ok := ast.Unparen(call.Args[1]).(*ast.UnaryExpr)
	if !ok || address.Op != token.AND {
		return false
	}
	return x.returnedJSONValue(address.X) && x.synchronousJSONCall(call) && plainJSONType(x.pkg.TypesInfo.TypeOf(address), map[types.Type]bool{}, "Unmarshal")
}

func (x *extractor) returnedJSONValue(value ast.Expr) bool {
	id, ok := ast.Unparen(value).(*ast.Ident)
	if !ok {
		return false
	}
	object := x.pkg.TypesInfo.ObjectOf(id)
	binding, found := x.bindings[object]
	if !found || len(binding.values) != 1 || binding.values[0] != nil {
		return false
	}
	return x.returnedJSONUses(object, id)
}

func (x *extractor) returnedJSONUses(object types.Object, destination *ast.Ident) bool {
	for _, use := range x.fields.uses[object] {
		if use == destination {
			continue
		}
		parent, ok := x.fields.parents[use].(*ast.ReturnStmt)
		if !ok || !x.outerFieldReturn(parent) {
			return false
		}
	}
	return true
}

func (x *extractor) synchronousJSONCall(call *ast.CallExpr) bool {
	for parent := ast.Node(call); parent != nil; parent = x.fields.parents[parent] {
		switch parent.(type) {
		case *ast.FuncLit, *ast.GoStmt, *ast.DeferStmt:
			return false
		}
	}
	return true
}
