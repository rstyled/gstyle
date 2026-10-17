package gopatterns

import (
	"encoding/json"
	"go/ast"
	"go/types"
	"strconv"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

// Callee values are dependencies, not argument zero. Static declaration identity
// remains in Callee; dynamic values retain their own canonical producer path.
func (b *irBuilder) callTargetFacts(body *ast.BlockStmt) {
	ast.Inspect(body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		b.callTargetValue(call)
		b.methodDeclarationFact(call)
		b.callEntryFact(call)
		b.callShapeFact(call)
		return true
	})
}

// An interface method declaration is a contract identity, not a resolved
// implementation target. Callee remains unknown for dynamic dispatch.
func (b *irBuilder) methodDeclarationFact(call *ast.CallExpr) {
	selector, ok := unparen(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return
	}
	selection := b.info.Selections[selector]
	if selection == nil || selection.Kind() == types.FieldVal {
		return
	}
	data, _ := json.Marshal(patterns.FuncID(selection.Obj().(*types.Func)))
	extensions := b.pool.SingletonExtension("gstyle.go:method-declaration", string(data))
	if selection.Kind() == types.MethodExpr {
		extensions = append(extensions, b.pool.SingletonExtension("gstyle.go:method-expression", "true")...)
	}
	attrs := &b.attributes[b.graph.Nodes[b.nodes[call]-1].Attributes-1]
	attrs.Extensions = append(attrs.Extensions, extensions...)
}

func (b *irBuilder) callTargetValue(call *ast.CallExpr) {
	source, target := b.nodes[call.Fun], b.nodes[call]
	if source == 0 || target == 0 || b.graph.Nodes[target-1].Category != pdg.Call {
		return
	}
	b.graph.Edges = append(b.graph.Edges, pdg.Edge{ID: b.pool.Text("target-e" + strconv.Itoa(len(b.graph.Edges)+1)), Source: source, Target: target, Kind: pdg.Data, Subkind: b.pool.Text("call-target"), Evidence: b.pool.Evidence(pdg.Supported, "The evaluated function value selects the invoked target; it is not an argument."), Provenance: b.sourceOrigin})
}

// Entry kind describes the invocation boundary, not the callee body's effects.
func (b *irBuilder) callEntryFact(call *ast.CallExpr) {
	ref := b.nodes[call]
	if ref == 0 || b.graph.Nodes[ref-1].Category != pdg.Call {
		return
	}
	kindData, _ := json.Marshal(callEntryKind(b.callee(call.Fun)))
	inputsData, _ := json.Marshal(b.closeArgumentInputs(call))
	attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	attrs.Extensions = append(attrs.Extensions,
		pdg.Extension{Name: b.pool.Text("gstyle.go:call-entry"), Value: b.pool.Text(string(kindData))},
		pdg.Extension{Name: b.pool.Text("gstyle.go:resource-inputs"), Value: b.pool.Text(string(inputsData))})

}

// Only a declared Close-capable formal argument is an ownership boundary.
// Passing a handle to formatting, readers or interface{} is not that contract.
func (b *irBuilder) closeArgumentInputs(call *ast.CallExpr) []pdg.Position {
	positions := []pdg.Position{}
	typ := b.info.TypeOf(call.Fun)
	if typ == nil {
		return positions
	}
	signature, ok := typ.Underlying().(*types.Signature)
	if !ok {
		return positions
	}
	offset := 0
	if b.implicitReceiver(call.Fun) {
		offset = 1
	}
	for i := range call.Args {
		if closeCapableType(callParameterType(signature, i, call.Ellipsis.IsValid())) {
			positions = append(positions, pdg.Position(i+offset+1))
		}
	}
	return positions
}

func callEntryKind(object types.Object) string {
	switch object := object.(type) {
	case *types.Builtin:
		return "builtin"
	case *types.Func:
		if object.Type().(*types.Signature).Recv() != nil {
			return "method"
		}
		return "function"
	default:
		return "unknown"
	}
}

func callParameterType(signature *types.Signature, index int, spread bool) types.Type {
	count := signature.Params().Len()
	if count == 0 {
		return nil
	}
	if signature.Variadic() && index >= count-1 {
		typ := signature.Params().At(count - 1).Type()
		if spread {
			return typ
		}
		return typ.(*types.Slice).Elem()
	}
	if index >= count {
		return nil
	}
	return signature.Params().At(index).Type()
}

func closeCapableType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	selection := types.NewMethodSet(typ).Lookup(nil, "Close")
	if selection == nil {
		return false
	}
	signature := selection.Obj().Type().(*types.Signature)
	return signature.Params().Len() == 0
}
