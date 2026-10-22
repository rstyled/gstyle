package goquality

import (
	"go/ast"
	"go/types"
)

// Typed string scans read only their input string. Retain mutation evidence
// because scalar destinations can point to shared storage.
func (x *extractor) memoryScan(call *ast.CallExpr, callee string) bool {
	if call.Ellipsis.IsValid() {
		return false
	}
	switch callee {
	case "fmt.Sscan", "fmt.Sscanln":
		return x.scanScalars(call.Args[1:])
	case "fmt.Sscanf":
		return x.scanScalars(call.Args[2:])
	default:
		return false
	}
}

// fmt invokes Scanner before its scalar conversion. Reject open types and
// methods on either the destination or its element, including named scalars.
func (x *extractor) scanScalars(arguments []ast.Expr) bool {
	for _, argument := range arguments {
		pointer, ok := types.Unalias(x.pkg.TypesInfo.TypeOf(argument)).(*types.Pointer)
		if !ok || !formatScalar(pointer.Elem()) {
			return false
		}
	}
	return len(arguments) != 0
}
