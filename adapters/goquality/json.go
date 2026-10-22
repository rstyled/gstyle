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
	// RawMessage's standard methods only read bytes or copy into the decode
	// destination; jsonv2 aliases RawMessage to jsontext.Value. Destination
	// ownership is checked separately.
	if metadataType(typ, "encoding/json", "RawMessage") || metadataType(typ, "encoding/json/jsontext", "Value") {
		return true
	}
	for i := 0; i < typ.NumMethods(); i++ {
		if slices.Contains([]string{method + "JSON", method + "Text"}, typ.Method(i).Name()) {
			return false
		}
	}
	return plainJSONType(typ.Underlying(), seen, method)
}

// Decode only into an unaliased local; reference-bearing values start at zero.
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
	if x.scalarJSONValue(object, id) {
		return true
	}
	binding, found := x.bindings[object]
	if !found || len(binding.values) != 1 || binding.values[0] != nil {
		return false
	}
	return x.returnedJSONUses(object, id)
}

// Scalar values have no mutable child storage. Reads and value copies do not
// expose the destination slot, but addresses, methods and captures can.
func (x *extractor) scalarJSONValue(object types.Object, destination *ast.Ident) bool {
	variable, ok := object.(*types.Var)
	if !ok || !isLocalVar(variable) || x.parameters[object] {
		return false
	}
	basic, ok := underlying(variable.Type()).(*types.Basic)
	if !ok || basic.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsString) == 0 {
		return false
	}
	return x.returnedJSONUses(object, destination)
}

func (x *extractor) scalarJSONUse(node ast.Node) bool {
	for parent := x.fields.parents[node]; parent != nil; parent = x.fields.parents[node] {
		switch parent.(type) {
		case *ast.GoStmt, *ast.DeferStmt:
			return false
		}
		if !tableValueContext(node, parent, x.pkg.TypesInfo) {
			return false
		}
		node = parent
	}
	return true
}

func (x *extractor) returnedJSONUses(object types.Object, destination *ast.Ident) bool {
	for _, use := range x.fields.uses[object] {
		if use == destination {
			continue
		}
		if !x.outerFieldReturn(use) || !(x.scalarJSONUse(use) || x.returnedJSONAddress(use)) {
			return false
		}
	}
	return true
}

// A direct outer return publishes the local slot only after construction ends.
// Field addresses and intermediate aliases do not establish that boundary.
func (x *extractor) returnedJSONAddress(use *ast.Ident) bool {
	proof := receiverProof{parents: x.fields.parents}
	_, parent := proof.parent(use)
	address, ok := parent.(*ast.UnaryExpr)
	if !ok || address.Op != token.AND {
		return false
	}
	_, parent = proof.parent(address)
	_, returned := parent.(*ast.ReturnStmt)
	return returned
}

func (x *extractor) synchronousJSONCall(call *ast.CallExpr) bool {
	if x.loopSafeJSONCall(call) {
		return x.synchronousContainerCall(call)
	}
	var body *ast.BlockStmt
	for parent := ast.Node(call); parent != nil; parent = x.fields.parents[parent] {
		if block, ok := parent.(*ast.BlockStmt); ok {
			body = block
		}
		switch parent.(type) {
		case *ast.FuncLit, *ast.GoStmt, *ast.DeferStmt, *ast.ForStmt, *ast.RangeStmt:
			return false
		}
	}
	return body != nil && !labeledBlock(body)
}

// Scalars cannot publish mutable children through copies. Reference-bearing
// results retain the older return-only proof inside repeated control flow.
func (x *extractor) loopSafeJSONCall(call *ast.CallExpr) bool {
	address := ast.Unparen(call.Args[1]).(*ast.UnaryExpr)
	destination, ok := ast.Unparen(address.X).(*ast.Ident)
	if !ok {
		return false
	}
	object := x.pkg.TypesInfo.ObjectOf(destination)
	return x.scalarJSONValue(object, destination) || x.returnOnlyJSONUses(object, destination)
}

func (x *extractor) returnOnlyJSONUses(object types.Object, destination *ast.Ident) bool {
	returned := false
	for _, use := range x.fields.uses[object] {
		if use == destination {
			continue
		}
		statement, ok := x.fields.parents[use].(*ast.ReturnStmt)
		if !ok || !x.outerFieldReturn(statement) {
			return false
		}
		returned = true
	}
	return returned
}
