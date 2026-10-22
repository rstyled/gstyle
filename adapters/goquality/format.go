package goquality

import (
	"go/ast"
	"go/types"
)

// Each proof excludes callbacks and external IO while retaining memory writes.
func (x *extractor) memoryWrite(call *ast.CallExpr, callee string) bool {
	return x.memoryFormat(call, callee) || x.memoryScan(call, callee) || x.memoryXMLDecode(call, callee) || x.memoryJSONDecode(call, callee)
}

// The writer's concrete implementation cannot perform external IO. Retain a
// mutation even for fresh records: a Buffer can wrap borrowed backing bytes.
func (x *extractor) memoryFormat(call *ast.CallExpr, callee string) bool {
	switch callee {
	case "fmt.Fprint", "fmt.Fprintf", "fmt.Fprintln", "io.WriteString":
	default:
		return false
	}
	if call.Ellipsis.IsValid() || len(call.Args) == 0 {
		return false
	}
	return memoryWriterType(x.pkg.TypesInfo.TypeOf(call.Args[0])) && x.formatScalars(call.Args[1:])
}

func (x *extractor) formatScalars(arguments []ast.Expr) bool {
	for _, argument := range arguments {
		if !formatScalar(x.pkg.TypesInfo.TypeOf(argument)) {
			return false
		}
	}
	return true
}

func memoryWriterType(typ types.Type) bool {
	pointer, ok := types.Unalias(typ).(*types.Pointer)
	return ok && (metadataType(pointer.Elem(), "bytes", "Buffer") || metadataType(pointer.Elem(), "strings", "Builder"))
}

// fmt may call Format, String, GoString or Error, including on nested values.
// Restrict the first model to scalars with no value or pointer methods.
func formatScalar(typ types.Type) bool {
	if _, basic := underlying(typ).(*types.Basic); !basic {
		return false
	}
	return types.NewMethodSet(typ).Len() == 0 && types.NewMethodSet(types.NewPointer(typ)).Len() == 0
}
