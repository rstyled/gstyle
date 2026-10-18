package goquality

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/quality"
)

func (x *extractor) formalStorage(value ast.Expr) *int {
	id, ok := ast.Unparen(value).(*ast.Ident)
	if !ok {
		return nil
	}
	object := x.pkg.TypesInfo.ObjectOf(id)
	if _, rebound := x.bindings[object]; rebound {
		return nil
	}
	if object == x.receiver && object != nil {
		index := -1
		return &index
	}
	if index, ok := x.paramIndex[object]; ok {
		return &index
	}
	return nil
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
	if address, ok := ast.Unparen(value).(*ast.UnaryExpr); ok && address.Op == token.AND {
		return quality.Storage{Provenance: x.addressOrigin(address), Parameter: x.mutationParameter(address.X, false)}
	}
	return quality.Storage{Provenance: x.mutationProvenance(value, true), Parameter: x.referenceStorage(value)}
}

func (x *extractor) callStorage(call *ast.CallExpr) []quality.Storage {
	result := make([]quality.Storage, len(call.Args))
	for i, arg := range call.Args {
		result[i] = x.argumentStorage(arg)
	}
	return result
}

func (x *extractor) bindCallStorage(call *ast.CallExpr, fact quality.Call) quality.Call {
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
