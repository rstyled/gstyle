package goquality

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
)

// A nonempty append to a full slice with len == cap allocates new outer
// storage. Referenced elements retain their borrowed provenance.
func (x *extractor) freshAppend(call *ast.CallExpr) bool {
	if !x.appendAddsElements(call) {
		return false
	}
	return x.fullSlice(call.Args[0])
}

func (x *extractor) appendAddsElements(call *ast.CallExpr) bool {
	if len(call.Args) < 2 {
		return false
	}
	if !call.Ellipsis.IsValid() {
		return true
	}
	value := ast.Unparen(call.Args[1])
	if literal, ok := value.(*ast.CompositeLit); ok {
		return len(literal.Elts) > 0
	}
	literal := x.pkg.TypesInfo.Types[value].Value
	return literal != nil && literal.Kind() == constant.String && constant.StringVal(literal) != ""
}

func (x *extractor) fullSlice(value ast.Expr) bool {
	switch value := ast.Unparen(value).(type) {
	case *ast.SliceExpr:
		return value.Slice3 && x.sameCapacityBound(value.High, value.Max)
	case *ast.CallExpr:
		function, ok := x.calledObject(unwrappedCallee(value.Fun)).(*types.Func)
		return ok && x.cappedReturns[function.Origin()]
	case *ast.Ident:
		return x.fullSlice(x.localSliceValue(value))
	default:
		return false
	}
}

func (x *extractor) sameCapacityBound(left, right ast.Expr) bool {
	if left == nil || right == nil {
		return false
	}
	a, b := x.pkg.TypesInfo.Types[left].Value, x.pkg.TypesInfo.Types[right].Value
	if a != nil && b != nil {
		return constant.Compare(a, token.EQL, b)
	}
	object := x.capacityBoundObject(left)
	return object == x.capacityBoundObject(right) && x.stableCapacityObject(object)
}

// Only stable scalar identifiers and builtin len of an identifier match.
// Repeated calls or field reads do not establish equal runtime bounds.
func (x *extractor) capacityBoundObject(value ast.Expr) types.Object {
	if id, ok := ast.Unparen(value).(*ast.Ident); ok {
		return x.pkg.TypesInfo.ObjectOf(id)
	}
	call, ok := ast.Unparen(value).(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return nil
	}
	builtin, ok := x.calledObject(unwrappedCallee(call.Fun)).(*types.Builtin)
	if !ok || builtin.Name() != "len" {
		return nil
	}
	return x.capacityLengthObject(call.Args[0])
}

func (x *extractor) capacityLengthObject(value ast.Expr) types.Object {
	id, ok := ast.Unparen(value).(*ast.Ident)
	if !ok {
		return nil
	}
	switch underlying(x.pkg.TypesInfo.TypeOf(id)).(type) {
	case *types.Slice, *types.Array, *types.Pointer, *types.Basic:
		return x.pkg.TypesInfo.ObjectOf(id)
	default:
		return nil // Map and channel lengths can change through aliases.
	}
}

func (x *extractor) stableCapacityObject(object types.Object) bool {
	if object == nil || packageVariable(object) || x.bindings[object].unknown {
		return false
	}
	for _, id := range x.fields.uses[object] {
		if !x.stableCapacityUse(id) {
			return false
		}
	}
	return true
}

func (x *extractor) stableCapacityUse(id *ast.Ident) bool {
	proof := receiverProof{parents: x.fields.parents}
	if proof.function(id) != nil {
		return false // A closure can change the bound during operand evaluation.
	}
	_, parent := proof.parent(id)
	selector, ok := parent.(*ast.SelectorExpr)
	return !ok || !implicitReceiverAddress(x.pkg.TypesInfo, selector)
}
