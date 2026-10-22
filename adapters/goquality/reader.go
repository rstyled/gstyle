package goquality

import (
	"go/ast"
	"go/types"
)

// Memory readers stay private: only constructor chains, reads and Close may use them.
func (x *extractor) localMemoryRead(call *ast.CallExpr) bool {
	switch x.readerCallee(call) {
	case "io.ReadAll":
		return len(call.Args) == 1 && x.readerSynchronous(call) && x.memoryReader(call.Args[0], map[types.Object]bool{})
	case "io.Closer.Close", "encoding/json.Decoder.Token", "encoding/json.Decoder.More":
		return x.readerSynchronous(call) && x.localMemoryMethod(call)
	default:
		return false
	}
}

func (x *extractor) localMemoryMethod(call *ast.CallExpr) bool {
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return false
	}
	selection := x.pkg.TypesInfo.Selections[selector]
	if selection == nil {
		return false
	}
	if selection.Kind() == types.MethodExpr {
		return len(call.Args) == 1 && x.memoryReader(call.Args[0], map[types.Object]bool{})
	}
	return len(call.Args) == 0 && x.memoryReader(selector.X, map[types.Object]bool{})
}

func (x *extractor) readerCallee(call *ast.CallExpr) string {
	function, ok := x.calledObject(unwrappedCallee(call.Fun)).(*types.Func)
	if !ok {
		return ""
	}
	return resolvedName(function)
}

func (x *extractor) memoryReader(value ast.Expr, seen map[types.Object]bool) bool {
	switch value := ast.Unparen(value).(type) {
	case *ast.CallExpr:
		switch x.readerCallee(value) {
		case "strings.NewReader", "bytes.NewReader", "bytes.NewBuffer", "bytes.NewBufferString":
			// Buffer reads change only the private header, never borrowed bytes.
			return len(value.Args) == 1
		case "compress/flate.NewReader", "io.LimitReader", "encoding/json.NewDecoder":
			return len(value.Args) != 0 && x.memoryReader(value.Args[0], seen)
		}
	case *ast.Ident:
		return x.memoryReaderBinding(value, seen)
	}
	return false
}

func (x *extractor) closedReaderUse(node ast.Node) bool {
	for {
		var parent ast.Node
		node, parent = (receiverProof{parents: x.fields.parents}).parent(node)
		if selector, ok := parent.(*ast.SelectorExpr); ok {
			return x.readerMethodUse(selector)
		}
		call := readerArgument(parent, node)
		if call == nil {
			return false
		}
		switch x.readerCallee(call) {
		case "io.ReadAll", "encoding/json.Decoder.Token", "encoding/json.Decoder.More":
			return true
		case "io.LimitReader", "compress/flate.NewReader", "encoding/json.NewDecoder":
			node = call
		default:
			return false
		}
	}
}

func readerArgument(parent, node ast.Node) *ast.CallExpr {
	call, ok := parent.(*ast.CallExpr)
	if !ok || len(call.Args) == 0 || call.Args[0] != node {
		return nil
	}
	return call
}

func (x *extractor) readerMethodUse(selector *ast.SelectorExpr) bool {
	call, ok := x.fields.parents[selector].(*ast.CallExpr)
	if !ok || call.Fun != selector || len(call.Args) != 0 {
		return false
	}
	switch x.readerCallee(call) {
	case "io.Closer.Close", "encoding/json.Decoder.Token", "encoding/json.Decoder.More":
		return true
	}
	return false
}

func (x *extractor) memoryReaderBinding(value *ast.Ident, seen map[types.Object]bool) bool {
	object := x.pkg.TypesInfo.ObjectOf(value)
	binding := x.bindings[object]
	if !x.readerBindingValid(object, seen) {
		return false
	}
	seen[object] = true
	if !x.memoryReader(binding.values[0], seen) {
		return false
	}
	for _, use := range x.fields.uses[object] {
		if !x.readerSynchronous(use) || !x.closedReaderUse(use) {
			return false
		}
	}
	return len(x.fields.uses[object]) != 0
}

func (x *extractor) readerBindingValid(object types.Object, seen map[types.Object]bool) bool {
	return x.localContainerRoot(object) && !seen[object] && !x.bindings[object].unknown && len(x.bindings[object].values) == 1
}

func (x *extractor) readerSynchronous(node ast.Node) bool {
	for parent := x.fields.parents[node]; parent != nil; parent = x.fields.parents[parent] {
		switch parent.(type) {
		case *ast.FuncLit, *ast.GoStmt:
			return false
		}
	}
	return true
}
