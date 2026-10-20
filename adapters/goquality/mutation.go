package goquality

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/rstyled/gstyle/domain/quality"
)

type referenceBinding struct {
	values       []ast.Expr
	unknown      bool
	selfReturned bool
	closedSelf   bool
}

// Bindings retain all possible assignments. Provenance is conservative across
// branches and loops; it does not depend on the AST walk's execution order.
func (x *extractor) collectBindings(body *ast.BlockStmt) {
	x.collectFieldWrites(body)
	ast.Inspect(body, func(node ast.Node) bool {
		x.addLiteralParameters(node)
		x.addressBinding(node)
		switch node := node.(type) {
		case *ast.AssignStmt:
			x.bindAssignments(node.Lhs, node.Rhs)
		case *ast.ValueSpec:
			places := make([]ast.Expr, len(node.Names))
			for index, name := range node.Names {
				places[index] = name
			}
			x.bindAssignments(places, node.Values)
		case *ast.RangeStmt:
			x.unknownBinding(node.Key)
			x.bindRangeValue(node)
		}
		return true
	})
	x.closeSelfBindings()
}

// Bindings are complete before this derived noescape certificate is cached.
func (x *extractor) closeSelfBindings() {
	for object, binding := range x.bindings {
		if !x.parameters[object] || binding.unknown || !binding.selfReturned || len(binding.values) != 0 {
			continue
		}
		binding.closedSelf = x.containerUses(object)
		x.bindings[object] = binding
	}
}

// Range values from shared containers remain shared; fresh containers can
// still contain borrowed references. Other range forms need separate models.
func (x *extractor) bindRangeValue(node *ast.RangeStmt) {
	id, ok := ast.Unparen(node.Value).(*ast.Ident)
	if !ok {
		return
	}
	object := x.pkg.TypesInfo.ObjectOf(id)
	binding := x.bindings[object]
	switch underlying(x.pkg.TypesInfo.TypeOf(node.X)).(type) {
	case *types.Slice, *types.Map, *types.Array:
		binding.values = append(binding.values, node.X)
	default:
		binding.unknown = true
	}
	x.bindings[object] = binding
}

func (x *extractor) bindAssignments(places, values []ast.Expr) {
	for index, place := range places {
		id, ok := ast.Unparen(place).(*ast.Ident)
		if !ok {
			continue
		}
		object := x.pkg.TypesInfo.ObjectOf(id)
		if object == nil {
			continue
		}
		value := assignmentValue(values, index, len(places))
		if x.selfSliceAssignment(object, value) {
			continue
		}
		binding := x.bindings[object]
		if x.containerSelfAssignment(object, value) {
			binding.selfReturned = true
		} else {
			binding.values = append(binding.values, value)
		}
		x.bindings[object] = binding
	}
}

// A bare self slice is a view of the same storage, with unchanged provenance.
func (x *extractor) selfSliceAssignment(object types.Object, value ast.Expr) bool {
	slice, ok := ast.Unparen(value).(*ast.SliceExpr)
	if !ok {
		return false
	}
	target, ok := ast.Unparen(slice.X).(*ast.Ident)
	return ok && x.pkg.TypesInfo.ObjectOf(target) == object
}

// A self assignment from a modeled container operation keeps local ownership:
// these calls return their selected input, a view of it, or fresh storage.
func (x *extractor) containerSelfAssignment(object types.Object, value ast.Expr) bool {
	call, ok := ast.Unparen(value).(*ast.CallExpr)
	if !ok {
		return false
	}
	target, ok := ast.Unparen(x.selfContainerInput(call)).(*ast.Ident)
	return ok && x.pkg.TypesInfo.ObjectOf(target) == object
}

func (x *extractor) selfContainerInput(call *ast.CallExpr) ast.Expr {
	switch object := x.calledObject(unwrappedCallee(call.Fun)).(type) {
	case *types.Builtin:
		if object.Name() == "append" {
			return call.Args[0]
		}
	case *types.Func:
		if result, ok := x.factories[object]; ok {
			return x.factoryArgument(call, result)
		}
		if quality.ContainerMutation(resolvedName(object)) {
			return x.containerArgs(call)[0]
		}
	}
	return nil
}

func (x *extractor) addressBinding(value ast.Node) {
	// Explicit and implicit addresses permit writes outside direct bindings.
	switch value := value.(type) {
	case *ast.UnaryExpr:
		if value.Op == token.AND {
			x.unknownBinding(value.X)
		}
	case *ast.SelectorExpr:
		// A pointer method can replace a reference alias's map or slice header.
		switch underlying(x.pkg.TypesInfo.TypeOf(value.X)).(type) {
		case *types.Slice, *types.Map:
			if implicitReceiverAddress(x.pkg.TypesInfo, value) {
				x.unknownBinding(value.X)
			}
		}
	}
}

func (x *extractor) unknownBinding(place ast.Expr) {
	id, ok := ast.Unparen(place).(*ast.Ident)
	if !ok {
		return
	}
	object := x.pkg.TypesInfo.ObjectOf(id)
	if object == nil {
		return
	}
	binding := x.bindings[object]
	binding.unknown = true
	x.bindings[object] = binding
}

func (x *extractor) mutate(place ast.Expr, pos token.Pos, storage bool) {
	if discardedPlace(place) {
		return
	}
	object, fields := x.placeRoot(place)
	if object == nil {
		x.temporaryMutation(place, pos, storage, fields)
		return
	}
	if packageVariable(object) {
		x.effect(quality.Global, resolvedVariable(object), pos)
		return
	}
	provenance := x.mutationProvenance(place, storage)
	x.fact.Mutations = append(x.fact.Mutations, quality.Mutation{
		Parameter: x.observableParameter(place, storage, provenance), Root: object.Name(), RootID: x.objectID(object), FieldPath: strings.Join(fields, "."), Line: x.line(pos), Provenance: provenance,
	})
}

func (x *extractor) placeRoot(place ast.Expr) (types.Object, []string) {
	switch place := place.(type) {
	case *ast.Ident:
		return x.pkg.TypesInfo.ObjectOf(place), nil
	case *ast.SelectorExpr:
		return x.selectorRoot(place)
	default:
		base := projectedBase(place)
		if base == nil {
			return nil, nil
		}
		return x.placeRoot(base)
	}
}

func (x *extractor) temporaryMutation(place ast.Expr, pos token.Pos, storage bool, fields []string) {
	provenance := x.mutationProvenance(place, storage)
	position := physicalPosition(x.pkg.Fset, pos)
	x.fact.Mutations = append(x.fact.Mutations, quality.Mutation{Parameter: x.observableParameter(place, storage, provenance), Root: "<temporary>", RootID: fmt.Sprintf("%d:%d", position.Line, position.Column), FieldPath: strings.Join(fields, "."), Line: position.Line, Provenance: provenance})
}

func (x *extractor) mutationProvenance(place ast.Expr, storage bool) quality.Provenance {
	if storage {
		return x.origin(place, map[types.Object]bool{})
	}
	return x.placeProvenance(place)
}

func (x *extractor) selectorRoot(place *ast.SelectorExpr) (types.Object, []string) {
	object := x.pkg.TypesInfo.ObjectOf(place.Sel)
	if object != nil && packageVariable(object) {
		return object, nil
	}
	root, fields := x.placeRoot(place.X)
	return root, append(fields, place.Sel.Name)
}

func (x *extractor) placeProvenance(place ast.Expr) quality.Provenance {
	switch place := place.(type) {
	case *ast.Ident:
		return quality.Local
	case *ast.ParenExpr:
		return x.placeProvenance(place.X)
	case *ast.StarExpr:
		return x.origin(place.X, map[types.Object]bool{})
	case *ast.SelectorExpr:
		return x.selectorProvenance(place)
	case *ast.IndexExpr:
		return x.indexProvenance(place)
	default:
		return quality.Unknown
	}
}

func (x *extractor) selectorProvenance(place *ast.SelectorExpr) quality.Provenance {
	if embeddedPointer(x.pkg.TypesInfo.Selections[place]) {
		return x.fieldOrigin(place, map[types.Object]bool{})
	}
	if x.localReceiverField(place) {
		return quality.Local
	}
	if _, ok := underlying(x.pkg.TypesInfo.TypeOf(place.X)).(*types.Pointer); ok {
		return x.origin(place.X, map[types.Object]bool{})
	}
	return x.placeProvenance(place.X)
}

func (x *extractor) indexProvenance(place *ast.IndexExpr) quality.Provenance {
	if valueArray(x.pkg.TypesInfo.TypeOf(place.X)) {
		return x.placeProvenance(place.X)
	}
	return x.origin(place.X, map[types.Object]bool{})
}

func (x *extractor) origin(value ast.Expr, seen map[types.Object]bool) quality.Provenance {
	switch value := value.(type) {
	case nil:
		return quality.Local // Zero-initialized reference contains no shared state.
	case *ast.Ident:
		if x.pkg.TypesInfo.Types[value].IsNil() {
			return quality.Local
		}
		return x.objectOrigin(x.pkg.TypesInfo.ObjectOf(value), seen)
	case *ast.ParenExpr:
		return x.origin(value.X, seen)
	case *ast.SliceExpr:
		return x.sliceOrigin(value, seen)
	default:
		return x.constructedOrigin(value, seen)
	}
}

func (x *extractor) constructedOrigin(value ast.Expr, seen map[types.Object]bool) quality.Provenance {
	switch value := value.(type) {
	case *ast.UnaryExpr:
		return x.addressOrigin(value)
	case *ast.CompositeLit:
		return quality.Local
	case *ast.CallExpr:
		return x.allocationOrigin(value, seen)
	case *ast.SelectorExpr:
		return x.fieldOrigin(value, seen)
	case *ast.IndexExpr:
		return referencedOrigin(x.origin(value.X, seen))
	default:
		return quality.Unknown
	}
}

// Referenced values from shared storage remain shared. Fresh containers can
// contain borrowed elements, so local storage does not prove local elements.
func referencedOrigin(input quality.Provenance) quality.Provenance {
	if input == quality.External {
		return quality.External
	}
	return quality.Unknown
}

func (x *extractor) addressOrigin(value *ast.UnaryExpr) quality.Provenance {
	if value.Op == token.AND {
		if _, literal := ast.Unparen(value.X).(*ast.CompositeLit); literal {
			return quality.Local
		}
		return x.addressedPlaceOrigin(value.X)
	}
	return quality.Unknown
}

func (x *extractor) addressedPlaceOrigin(place ast.Expr) quality.Provenance {
	object, _ := x.placeRoot(place)
	if object != nil && packageVariable(object) {
		return quality.External
	}
	return x.placeProvenance(place)
}

func (x *extractor) fieldOrigin(value *ast.SelectorExpr, seen map[types.Object]bool) quality.Provenance {
	object, _ := x.placeRoot(value)
	if object == nil {
		return quality.Unknown
	}
	if x.localReceiverStorage(value) {
		return quality.Local
	}
	if x.externalFieldRoot(object, seen) {
		return quality.External
	}
	// Prove each field separately: fresh outer storage can hold borrowed data.
	return x.ownedField(value, seen)
}

func (x *extractor) externalFieldRoot(object types.Object, seen map[types.Object]bool) bool {
	if packageVariable(object) || x.parameters[object] {
		return true
	}
	_, pointer := underlying(object.Type()).(*types.Pointer)
	return pointer && x.objectOrigin(object, seen) == quality.External
}

func (x *extractor) sliceOrigin(value *ast.SliceExpr, seen map[types.Object]bool) quality.Provenance {
	if valueArray(x.pkg.TypesInfo.TypeOf(value.X)) {
		return x.placeProvenance(value.X)
	}
	return x.origin(value.X, seen)
}

func (x *extractor) allocationOrigin(call *ast.CallExpr, seen map[types.Object]bool) quality.Provenance {
	if x.pkg.TypesInfo.Types[call.Fun].IsType() {
		return x.conversionOrigin(call, seen)
	}
	switch object := x.calledObject(unwrappedCallee(call.Fun)).(type) {
	case *types.Builtin:
		return x.builtinAllocationOrigin(call, object.Name(), seen)
	case *types.Func:
		if result, ok := x.factories[object]; ok {
			return x.factoryOrigin(call, result, seen)
		}
		if quality.ContainerClone(resolvedName(object)) {
			return quality.Local
		}
	}
	return quality.Unknown
}

func (x *extractor) factoryOrigin(call *ast.CallExpr, result factoryResult, seen map[types.Object]bool) quality.Provenance {
	if result.parameter == nil {
		return quality.Local
	}
	argument := x.factoryArgument(call, result)
	if argument == nil {
		return quality.Unknown
	}
	return appendOrigin(x.origin(argument, seen))
}

func (x *extractor) factoryArgument(call *ast.CallExpr, result factoryResult) ast.Expr {
	arguments := x.containerArgs(call)
	if result.parameter == nil || *result.parameter >= len(arguments) || call.Ellipsis.IsValid() {
		return nil
	}
	return arguments[*result.parameter]
}

func (x *extractor) builtinAllocationOrigin(call *ast.CallExpr, name string, seen map[types.Object]bool) quality.Provenance {
	switch name {
	case "make", "new":
		return quality.Local
	case "append":
		if x.freshAppend(call) {
			return quality.Local
		}
		return appendOrigin(x.origin(call.Args[0], seen))
	default:
		return quality.Unknown
	}
}

func (x *extractor) objectOrigin(object types.Object, seen map[types.Object]bool) quality.Provenance {
	if object == nil {
		return quality.Unknown
	}
	if packageVariable(object) {
		return quality.External
	}
	if x.parameters[object] {
		return quality.External
	}
	if seen[object] {
		return quality.Unknown
	}
	binding, ok := x.bindings[object]
	if !ok {
		return quality.Unknown
	}
	seen[object] = true
	defer delete(seen, object)
	return x.bindingOrigin(binding, seen)
}

func (x *extractor) bindingOrigin(binding referenceBinding, seen map[types.Object]bool) quality.Provenance {
	if binding.unknown {
		return quality.Unknown
	}
	if len(binding.values) == 0 {
		return quality.Local
	}
	result := x.bindingValueOrigin(binding.values[0], seen)
	for _, value := range binding.values[1:] {
		if result != x.bindingValueOrigin(value, seen) {
			return quality.Unknown
		}
	}
	if binding.selfReturned {
		return appendOrigin(result)
	}
	return result
}

// A range binding receives an element, rather than the container expression.
func (x *extractor) bindingValueOrigin(value ast.Expr, seen map[types.Object]bool) quality.Provenance {
	origin := x.origin(value, seen)
	if parent, ok := x.fields.parents[value].(*ast.RangeStmt); ok && parent.X == value {
		return referencedOrigin(origin)
	}
	return origin
}

// A container update can reuse its input or allocate. Only local input proves
// local output; shared or unknown input leaves the result's ownership unknown.
func appendOrigin(input quality.Provenance) quality.Provenance {
	if input == quality.Local {
		return quality.Local
	}
	return quality.Unknown
}

func assignmentValue(values []ast.Expr, index, placeCount int) ast.Expr {
	if len(values) == 0 {
		return nil
	}
	if len(values) == placeCount {
		return values[index]
	}
	return values[0]
}

func discardedPlace(place ast.Expr) bool {
	id, ok := ast.Unparen(place).(*ast.Ident)
	return ok && id.Name == "_"
}

func projectedBase(place ast.Expr) ast.Expr {
	switch place := place.(type) {
	case *ast.IndexExpr:
		return place.X
	case *ast.SliceExpr:
		return place.X
	case *ast.StarExpr:
		return place.X
	case *ast.ParenExpr:
		return place.X
	default:
		return nil
	}
}

func underlying(typ types.Type) types.Type {
	if typ == nil {
		return nil
	}
	return typ.Underlying()
}

func embeddedPointer(selection *types.Selection) bool {
	if selection == nil {
		return false
	}
	typ := underlying(selection.Recv())
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = underlying(pointer.Elem())
	}
	// The compiler supplies struct fields for every step before the final field.
	for _, index := range selection.Index()[:len(selection.Index())-1] {
		typ = underlying(typ.(*types.Struct).Field(index).Type())
		if _, ok := typ.(*types.Pointer); ok {
			return true
		}
	}
	return false
}

// String-to-slice conversions copy bytes or runes. Other slice conversions
// preserve their input storage; type checking establishes legal element types.
func (x *extractor) conversionOrigin(call *ast.CallExpr, seen map[types.Object]bool) quality.Provenance {
	input := call.Args[0]
	if x.pkg.TypesInfo.Types[input].IsNil() {
		return quality.Local
	}
	if _, slice := underlying(x.pkg.TypesInfo.TypeOf(call)).(*types.Slice); !slice {
		return quality.Unknown
	}
	basic, ok := underlying(x.pkg.TypesInfo.TypeOf(input)).(*types.Basic)
	if ok && basic.Kind() == types.String {
		return quality.Local
	}
	return x.origin(input, seen)
}
