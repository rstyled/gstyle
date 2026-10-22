package goquality

import (
	"go/ast"
	"go/types"
)

// A closed decoder over memory has no external reader. Plain destinations
// exclude JSON and text callbacks; shared destination writes remain effects.
func (x *extractor) memoryJSONDecode(call *ast.CallExpr, callee string) bool {
	if callee != "encoding/json.Decoder.Decode" || len(call.Args) != 1 {
		return false
	}
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return plainJSONType(x.pkg.TypesInfo.TypeOf(call.Args[0]), map[types.Type]bool{}, "Unmarshal") && x.readerSynchronous(call) && x.memoryReader(selector.X, map[types.Object]bool{})
}
