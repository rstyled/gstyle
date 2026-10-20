package goquality

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/quality"
	"golang.org/x/tools/go/packages"
)

// Prove the returned backing storage, not purity of the function or ownership
// of referenced elements. Unrelated calls and effects keep their normal facts.
func containerFactory(pkg *packages.Package, fn *ast.FuncDecl) factoryResult {
	if value := localContainerReturn(pkg, fn, nil); value != nil {
		return factoryResult{value: value}
	}
	signature := pkg.TypesInfo.Defs[fn.Name].Type().(*types.Signature)
	for index := range signature.Params().Len() {
		if signature.Variadic() && index == signature.Params().Len()-1 {
			continue
		}
		if value := localContainerReturn(pkg, fn, &index); value != nil {
			return factoryResult{value: value, parameter: &index}
		}
	}
	return factoryResult{}
}

func localContainerReturn(pkg *packages.Package, fn *ast.FuncDecl, parameter *int) ast.Expr {
	function := pkg.TypesInfo.Defs[fn.Name].(*types.Func)
	signature := function.Type().(*types.Signature)
	if fn.Body == nil || signature.Results().Len() != 1 || signature.Results().At(0).Name() != "" {
		return nil
	}
	switch underlying(signature.Results().At(0).Type()).(type) {
	case *types.Map, *types.Slice, *types.Array:
	default:
		return nil
	}
	x := extractor{pkg: pkg, factories: map[*types.Func]factoryResult{function: {value: fn.Name, parameter: parameter}}, bindings: map[types.Object]referenceBinding{}, parameters: map[types.Object]bool{}}
	x.addParameters(fn.Type.Params)
	x.addParameters(fn.Recv)
	x.collectBindings(fn.Body)
	if !x.seedContainerParameter(signature.Params(), parameter) {
		return nil
	}
	return x.ownedContainerReturn(fn.Body)
}

// A relative certificate substitutes a fresh input only for its selected formal.
// Other formals stay borrowed, including children referenced by container elements.
func (x *extractor) seedContainerParameter(parameters *types.Tuple, index *int) bool {
	if index == nil {
		return true
	}
	parameter := parameters.At(*index)
	switch underlying(parameter.Type()).(type) {
	case *types.Map, *types.Slice:
	default:
		return false
	}
	for _, use := range x.fields.uses[parameter] {
		_, parent := (receiverProof{parents: x.fields.parents}).parent(use)
		if address, ok := parent.(*ast.UnaryExpr); ok && address.Op == token.AND {
			return false
		}
	}
	delete(x.parameters, parameter)
	binding := x.bindings[parameter]
	binding.closedSelf = false
	binding.values = append(binding.values, nil)
	x.bindings[parameter] = binding
	return x.containerUses(parameter)
}

func (x *extractor) ownedContainerReturn(body *ast.BlockStmt) ast.Expr {
	var result ast.Expr
	valid := true
	ast.Inspect(body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncLit:
			return false
		case *ast.ReturnStmt:
			value := assignmentValue(node.Results, 0, 1)
			if !x.returnedContainer(value) {
				valid = false
				return false
			}
			result = value
		}
		return valid
	})
	if !valid {
		return nil
	}
	return result
}

// Self recursion is an induction hypothesis: every terminating return must
// independently construct fresh outer storage. Borrowed branches reject it.
func (x *extractor) returnedContainer(value ast.Expr) bool {
	switch value := ast.Unparen(value).(type) {
	case *ast.CompositeLit:
		return true
	case *ast.CallExpr:
		return x.returnedContainerCall(value)
	default:
		return x.returnedContainerRoot(value)
	}
}

func (x *extractor) returnedContainerCall(call *ast.CallExpr) bool {
	if x.pkg.TypesInfo.Types[call.Fun].IsType() {
		return false
	}
	if builtin, ok := x.calledObject(unwrappedCallee(call.Fun)).(*types.Builtin); ok && builtin.Name() == "append" {
		return len(call.Args) > 0 && x.returnedContainer(call.Args[0])
	}
	return x.allocationOrigin(call, map[types.Object]bool{}) == quality.Local
}

func (x *extractor) returnedContainerRoot(value ast.Expr) bool {
	id, ok := ast.Unparen(value).(*ast.Ident)
	if !ok {
		return false
	}
	object := x.pkg.TypesInfo.ObjectOf(id)
	if x.pkg.TypesInfo.Types[id].IsNil() {
		return true
	}
	binding, found := x.bindings[object]
	if !found || !x.localContainerRoot(object) || !x.containerUses(object) {
		return false
	}
	// The use proof permits only a closed JSON decode to take its address.
	binding.unknown = false
	return x.bindingOrigin(binding, map[types.Object]bool{}) == quality.Local
}

func (x *extractor) localContainerRoot(object types.Object) bool {
	variable, ok := object.(*types.Var)
	return ok && isLocalVar(variable) && !x.parameters[object]
}

func (x *extractor) containerUses(object types.Object) bool {
	proof := receiverProof{parents: x.fields.parents}
	for _, id := range x.fields.uses[object] {
		if proof.function(id) != nil || !x.containerUse(id) {
			return false
		}
	}
	return true
}

func (x *extractor) containerUse(id *ast.Ident) bool {
	proof := receiverProof{parents: x.fields.parents}
	node, parent := proof.parent(id)
	switch parent := parent.(type) {
	case *ast.CallExpr:
		return x.containerCall(id, parent)
	case *ast.UnaryExpr:
		return parent.Op == token.AND && x.containerDecode(parent)
	case *ast.SliceExpr:
		return x.containerSelfView(parent)
	default:
		return x.containerValueUse(node, parent)
	}
}

func (x *extractor) containerValueUse(node, parent ast.Node) bool {
	switch parent := parent.(type) {
	case *ast.IndexExpr:
		return parent.X == node && (receiverFields{receiverProof: receiverProof{pkg: x.pkg, parents: x.fields.parents}}).elementUse(parent)
	case *ast.RangeStmt:
		return parent.X == node
	case *ast.ReturnStmt:
		return true
	case *ast.AssignStmt:
		return assignmentPlace(node, parent)
	default:
		return false
	}
}

func (x *extractor) containerSelfView(view *ast.SliceExpr) bool {
	id, ok := ast.Unparen(view.X).(*ast.Ident)
	return ok && x.containerResult(id, view)
}

func (x *extractor) containerDecode(address *ast.UnaryExpr) bool {
	call, ok := x.fields.parents[address].(*ast.CallExpr)
	if !ok {
		return false
	}
	function, ok := x.calledObject(unwrappedCallee(call.Fun)).(*types.Func)
	return ok && resolvedName(function) == "encoding/json.Unmarshal" && x.localJSONDecode(call, "encoding/json.Unmarshal")
}

func (x *extractor) containerCall(id *ast.Ident, call *ast.CallExpr) bool {
	if !x.synchronousContainerCall(call) {
		return false
	}
	object := x.calledObject(unwrappedCallee(call.Fun))
	builtin, ok := object.(*types.Builtin)
	if !ok {
		return x.containerOperation(id, call, object)
	}
	switch builtin.Name() {
	case "len", "cap", "delete", "clear", "copy":
		return true
	case "append":
		return x.returnedAppend(id, call)
	default:
		return false
	}
}

func (x *extractor) returnedAppend(id *ast.Ident, call *ast.CallExpr) bool {
	if len(call.Args) == 0 || ast.Unparen(call.Args[0]) != id {
		return false
	}
	proof := receiverProof{parents: x.fields.parents}
	_, parent := proof.parent(call)
	if _, returned := parent.(*ast.ReturnStmt); returned {
		return true
	}
	return x.containerResult(id, call)
}

func (x *extractor) synchronousContainerCall(node ast.Node) bool {
	for ; node != nil; node = x.fields.parents[node] {
		switch node.(type) {
		case *ast.FuncLit, *ast.GoStmt, *ast.DeferStmt:
			return false
		}
	}
	return true
}

func (x *extractor) containerOperation(id *ast.Ident, call *ast.CallExpr, object types.Object) bool {
	function, ok := object.(*types.Func)
	if !ok {
		return false
	}
	if result, exists := x.factories[function]; exists {
		return x.relativeContainerCall(id, call, result)
	}
	return x.modeledContainerCall(id, call, function)
}

func (x *extractor) modeledContainerCall(id *ast.Ident, call *ast.CallExpr, function *types.Func) bool {
	if !quality.ContainerMutation(resolvedName(function)) {
		return false
	}
	if len(call.Args) == 0 || ast.Unparen(call.Args[0]) != id {
		return resolvedName(function) == "maps.Copy"
	}
	signature := function.Type().(*types.Signature)
	return signature.Results().Len() == 0 || x.containerResult(id, call)
}

func (x *extractor) relativeContainerCall(id *ast.Ident, call *ast.CallExpr, result factoryResult) bool {
	if ast.Unparen(x.factoryArgument(call, result)) != id {
		return false
	}
	_, parent := (receiverProof{parents: x.fields.parents}).parent(call)
	if _, returned := parent.(*ast.ReturnStmt); returned {
		return true
	}
	return x.containerResult(id, call)
}

func (x *extractor) containerResult(id *ast.Ident, call ast.Node) bool {
	proof := receiverProof{parents: x.fields.parents}
	_, parent := proof.parent(call)
	if _, discarded := parent.(*ast.ExprStmt); discarded {
		return true
	}
	assignment, ok := parent.(*ast.AssignStmt)
	if !ok || len(assignment.Lhs) != 1 {
		return false
	}
	target, ok := ast.Unparen(assignment.Lhs[0]).(*ast.Ident)
	return ok && (target.Name == "_" || x.pkg.TypesInfo.ObjectOf(target) == x.pkg.TypesInfo.ObjectOf(id))
}
