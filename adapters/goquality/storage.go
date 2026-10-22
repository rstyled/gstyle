package goquality

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/quality"
)

func (x *extractor) formalStorage(value ast.Expr) *int {
	return x.formalStorageSeen(value, map[types.Object]bool{})
}

func (x *extractor) formalStorageSeen(value ast.Expr, seen map[types.Object]bool) *int {
	if call, ok := ast.Unparen(value).(*ast.CallExpr); ok {
		return x.formalStorageSeen(x.pointerReturnArgument(call), seen)
	}
	id, ok := ast.Unparen(value).(*ast.Ident)
	if !ok {
		return nil
	}
	object := x.pkg.TypesInfo.ObjectOf(id)
	if binding, rebound := x.bindings[object]; rebound {
		return x.aliasParameter(object, binding, seen)
	}
	return x.parameterStorage(object)
}

func (x *extractor) parameterStorage(object types.Object) *int {
	if object == x.receiver && object != nil {
		index := -1
		return &index
	}
	if index, ok := x.paramIndex[object]; ok {
		return &index
	}
	return nil
}

// Stable aliases and closed self updates retain a formal's outer storage.
// Arbitrary reassignment and range elements cannot establish that identity.
func (x *extractor) aliasParameter(object types.Object, binding referenceBinding, seen map[types.Object]bool) *int {
	if x.parameters[object] {
		if !binding.closedSelf {
			return nil
		}
		return x.parameterStorage(object)
	}
	if seen[object] {
		return nil
	}
	seen[object] = true
	return x.bindingParameter(binding, seen)
}

func (x *extractor) bindingParameter(binding referenceBinding, seen map[types.Object]bool) *int {
	if binding.unknown || binding.selfReturned || len(binding.values) != 1 {
		return nil
	}
	if _, rangeValue := x.fields.parents[binding.values[0]].(*ast.RangeStmt); rangeValue {
		return nil
	}
	return x.formalStorageSeen(binding.values[0], seen)
}

// Only the formal's storage is bound. Reference fields and indirect pointees
// can alias independently, even when the outer object is freshly allocated.
func (x *extractor) referenceStorage(value ast.Expr) *int {
	value = ast.Unparen(value)
	if slice, ok := value.(*ast.SliceExpr); ok {
		if valueArray(x.pkg.TypesInfo.TypeOf(slice.X)) {
			return nil
		}
		return x.referenceStorage(slice.X)
	}
	return x.formalStorage(value)
}

func (x *extractor) mutationParameter(place ast.Expr, storage bool) *int {
	if storage {
		return x.referenceStorage(place)
	}
	switch place := ast.Unparen(place).(type) {
	case *ast.StarExpr:
		return x.referenceStorage(place.X)
	case *ast.SelectorExpr:
		if embeddedPointer(x.pkg.TypesInfo.Selections[place]) {
			return nil
		}
		return x.valueStorage(place.X)
	case *ast.IndexExpr:
		return x.referenceStorage(place.X)
	default:
		return nil
	}
}

func (x *extractor) valueStorage(value ast.Expr) *int {
	switch value := ast.Unparen(value).(type) {
	case *ast.Ident:
		if _, ok := underlying(x.pkg.TypesInfo.TypeOf(value)).(*types.Pointer); ok {
			return x.formalStorage(value)
		}
	case *ast.SelectorExpr:
		if _, ok := underlying(x.pkg.TypesInfo.TypeOf(value)).(*types.Struct); ok && !embeddedPointer(x.pkg.TypesInfo.Selections[value]) {
			return x.valueStorage(value.X)
		}
	}
	return nil
}

func (x *extractor) argumentStorage(value ast.Expr) quality.Storage {
	plain := plainJSONType(x.pkg.TypesInfo.TypeOf(value), map[types.Type]bool{}, "Marshal")
	if address, ok := ast.Unparen(value).(*ast.UnaryExpr); ok && address.Op == token.AND {
		return quality.Storage{PlainJSON: plain, Provenance: x.addressOrigin(address), Parameter: x.mutationParameter(address.X, false)}
	}
	return quality.Storage{PlainJSON: plain, Provenance: x.mutationProvenance(value, true), Parameter: x.referenceStorage(value)}
}

func (x *extractor) callStorage(call *ast.CallExpr) []quality.Storage {
	result := make([]quality.Storage, len(call.Args))
	for i, arg := range call.Args {
		result[i] = x.argumentStorage(arg)
	}
	return result
}

func (x *extractor) bindCallStorage(call *ast.CallExpr, fact quality.Call) quality.Call {
	fact.LocalJSONDecode = x.localJSONDecode(call, fact.Callee)
	fact.StorageArgs = x.callStorage(call)
	selector, ok := unwrappedCallee(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return fact
	}
	selection := x.pkg.TypesInfo.Selections[selector]
	if selection == nil || selection.Kind() == types.FieldVal {
		return fact
	}
	return x.methodStorage(selection, selector.X, fact)
}

func (x *extractor) methodStorage(selection *types.Selection, receiver ast.Expr, fact quality.Call) quality.Call {
	if selection.Kind() == types.MethodExpr {
		if len(fact.StorageArgs) > 0 {
			storage := fact.StorageArgs[0]
			if len(selection.Index()) != 1 {
				storage = quality.Storage{Provenance: quality.Unknown}
			}
			fact.Receiver = &storage
			fact.StorageArgs = fact.StorageArgs[1:]
		}
		return fact
	}
	storage := quality.Storage{Provenance: quality.Unknown}
	if len(selection.Index()) == 1 {
		storage = x.methodReceiverStorage(selection.Obj().Type().(*types.Signature), receiver)
	}
	fact.Receiver = &storage
	return fact
}

func (x *extractor) methodReceiverStorage(signature *types.Signature, receiver ast.Expr) quality.Storage {
	_, pointerMethod := underlying(signature.Recv().Type()).(*types.Pointer)
	_, pointerValue := underlying(x.pkg.TypesInfo.TypeOf(receiver)).(*types.Pointer)
	if pointerMethod && !pointerValue {
		return quality.Storage{Provenance: x.placeProvenance(receiver), Parameter: x.mutationParameter(receiver, false)}
	}
	return x.argumentStorage(receiver)
}

func (x *extractor) observableParameter(place ast.Expr, storage bool, provenance quality.Provenance) *int {
	if provenance != quality.External {
		return nil
	}
	return x.mutationParameter(place, storage)
}
