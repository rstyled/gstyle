package goquality

import (
	"go/ast"
	"go/types"
)

// A direct constructor cannot have a configured CharsetReader or TokenReader.
// Decode still mutates its destination, including shared child storage.
func (x *extractor) memoryXMLDecode(call *ast.CallExpr, callee string) bool {
	if callee != "encoding/xml.Decoder.Decode" || len(call.Args) != 1 {
		return false
	}
	input := x.freshXMLInput(call)
	if input == nil {
		return false
	}
	return x.memoryReader(input, map[types.Object]bool{}) && plainXMLType(x.pkg.TypesInfo.TypeOf(call.Args[0]), map[types.Type]bool{})
}

func (x *extractor) freshXMLInput(call *ast.CallExpr) ast.Expr {
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	constructor, ok := ast.Unparen(selector.X).(*ast.CallExpr)
	if !ok || x.readerCallee(constructor) != "encoding/xml.NewDecoder" || len(constructor.Args) != 1 {
		return nil
	}
	return constructor.Args[0]
}

// Reject all methods, including promoted XML, attribute and text callbacks.
// Inspect every field conservatively, even ignored and unexported fields.
func plainXMLType(typ types.Type, seen map[types.Type]bool) bool {
	if seen[typ] {
		return true
	}
	seen[typ] = true
	if types.NewMethodSet(typ).Len() != 0 || types.NewMethodSet(types.NewPointer(typ)).Len() != 0 {
		return false
	}
	return plainXMLShape(underlying(typ), seen)
}

func plainXMLShape(typ types.Type, seen map[types.Type]bool) bool {
	switch typ := typ.(type) {
	case *types.Basic:
		return true
	case *types.Pointer:
		return plainXMLType(typ.Elem(), seen)
	case *types.Slice:
		return plainXMLType(typ.Elem(), seen)
	case *types.Struct:
		return plainXMLFields(typ, seen)
	default:
		return false
	}
}

func plainXMLFields(typ *types.Struct, seen map[types.Type]bool) bool {
	for i := 0; i < typ.NumFields(); i++ {
		if !plainXMLType(typ.Field(i).Type(), seen) {
			return false
		}
	}
	return true
}
