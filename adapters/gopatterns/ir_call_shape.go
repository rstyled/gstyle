package gopatterns

import (
	"encoding/json"
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (b *irBuilder) callSignature(call *ast.CallExpr) *types.Signature {
	if b.implicitReceiver(call.Fun) {
		return nil
	}
	typ := b.info.TypeOf(call.Fun)
	if typ == nil {
		return nil
	}
	signature, _ := typ.Underlying().(*types.Signature)
	return signature
}
func (b *irBuilder) callTupleArity(call *ast.CallExpr) int {
	if len(call.Args) != 1 {
		return 0
	}
	tuple, ok := b.info.TypeOf(call.Args[0]).(*types.Tuple)
	if !ok {
		return 0
	}
	return tuple.Len()
}
func (b *irBuilder) callArgumentCount(call *ast.CallExpr) int {
	if arity := b.callTupleArity(call); arity != 0 {
		return arity
	}
	return len(call.Args)
}
func (b *irBuilder) callShapeFact(call *ast.CallExpr) {
	signature := b.callSignature(call)
	ref := b.nodes[call]
	if signature == nil || ref == 0 {
		return
	}
	if b.graph.Nodes[ref-1].Category != pdg.Call {
		return
	}
	fixed := signature.Params().Len()
	if signature.Variadic() {
		fixed--
	}
	shape := bindingflow.CallShape{Fixed: uint32(fixed), Variadic: signature.Variadic(), Spread: call.Ellipsis.IsValid(), Tuple: uint32(b.callTupleArity(call))}
	data, _ := json.Marshal(shape)
	attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle.go:call-shape", string(data))...)
}
func (b *irBuilder) tupleArityFact(expression ast.Expr, attrs *pdg.Attributes) {
	tuple, ok := b.info.TypeOf(expression).(*types.Tuple)
	if !ok || tuple.Len() < 2 {
		return
	}
	data, _ := json.Marshal(tuple.Len())
	attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle.go:tuple-arity", string(data))...)
}
