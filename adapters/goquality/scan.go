package goquality

import (
	"go/ast"
	"go/types"
)

// Typed scalar scans read only their string or JSON input. Retain mutation
// evidence because scalar destinations can point to shared storage.
func (x *extractor) memoryScan(call *ast.CallExpr, callee string) bool {
	if call.Ellipsis.IsValid() {
		return false
	}
	switch callee {
	case "fmt.Sscan", "fmt.Sscanln", "encoding/json.Unmarshal":
		return x.scanScalars(call.Args[1:])
	case "fmt.Sscanf":
		return x.scanScalars(call.Args[2:])
	default:
		return false
	}
}

// fmt and JSON can invoke decoding methods before scalar conversion. Reject
// open types and methods on either the destination or its element.
func (x *extractor) scanScalars(arguments []ast.Expr) bool {
	for _, argument := range arguments {
		pointer, ok := types.Unalias(x.pkg.TypesInfo.TypeOf(argument)).(*types.Pointer)
		if !ok || !formatScalar(pointer.Elem()) {
			return false
		}
	}
	return len(arguments) != 0
}
