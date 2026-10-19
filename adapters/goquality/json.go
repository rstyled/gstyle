package goquality

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"slices"
	"strings"
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
		field := typ.Field(i)
		tag := reflect.StructTag(typ.Tag(i)).Get("json")
		// Embedded fields can promote callbacks even when their tag is ignored.
		if !field.Anonymous() && (!field.Exported() || tag == "-") {
			continue
		}
		if !plainJSONField(field.Type(), tag, seen, method) {
			return false
		}
	}
	return true
}

func plainJSONField(typ types.Type, tag string, seen map[types.Type]bool, method string) bool {
	_, options, _ := strings.Cut(tag, ",")
	if method == "Marshal" && slices.Contains(strings.Split(options, ","), "omitzero") && jsonZeroMethod(typ) {
		return false
	}
	return plainJSONType(typ, seen, method)
}

func jsonZeroMethod(typ types.Type) bool {
	object, _, _ := types.LookupFieldOrMethod(typ, true, nil, "IsZero")
	method, ok := object.(*types.Func)
	if !ok {
		return false
	}
	signature := method.Type().(*types.Signature)
	if signature.Params().Len() != 0 || signature.Results().Len() != 1 {
		return false
	}
	result, ok := types.Unalias(signature.Results().At(0).Type()).(*types.Basic)
	return ok && result.Kind() == types.Bool
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
