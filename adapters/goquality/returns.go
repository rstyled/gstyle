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
func localContainerReturn(pkg *packages.Package, fn *ast.FuncDecl) ast.Expr {
	signature := pkg.TypesInfo.Defs[fn.Name].Type().(*types.Signature)
	if fn.Body == nil || signature.Results().Len() != 1 || signature.Results().At(0).Name() != "" {
		return nil
	}
	switch underlying(signature.Results().At(0).Type()).(type) {
	case *types.Map, *types.Slice, *types.Array:
	default:
		return nil
	}
	x := extractor{pkg: pkg, bindings: map[types.Object]referenceBinding{}, parameters: map[types.Object]bool{}}
	x.addParameters(fn.Type.Params)
	x.addParameters(fn.Recv)
	x.collectBindings(fn.Body)
	return x.ownedContainerReturn(fn.Body)
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

func (x *extractor) returnedContainer(value ast.Expr) bool {
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
	assignment, ok := parent.(*ast.AssignStmt)
	if !ok || len(assignment.Lhs) != 1 {
		return false
	}
	target, ok := ast.Unparen(assignment.Lhs[0]).(*ast.Ident)
	return ok && x.pkg.TypesInfo.ObjectOf(target) == x.pkg.TypesInfo.ObjectOf(id)
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
	if !ok || !quality.ContainerMutation(resolvedName(function)) {
		return false
	}
	if len(call.Args) == 0 || ast.Unparen(call.Args[0]) != id {
		return resolvedName(function) == "maps.Copy"
	}
	signature := function.Type().(*types.Signature)
	return signature.Results().Len() == 0 || x.containerResult(id, call)
}

func (x *extractor) containerResult(id *ast.Ident, call *ast.CallExpr) bool {
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
