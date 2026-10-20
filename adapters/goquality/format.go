package goquality

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/quality"
)

// The writer's concrete implementation cannot perform external IO. Retain a
// mutation even for fresh records: a Buffer can wrap borrowed backing bytes.
func (x *extractor) memoryFormat(call *ast.CallExpr, callee string) bool {
	if !quality.MemoryWriterCall(callee) || call.Ellipsis.IsValid() || len(call.Args) == 0 {
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
