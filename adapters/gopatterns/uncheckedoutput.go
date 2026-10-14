package gopatterns

import (
	"go/ast"
	"go/types"
	"slices"
)

// Restrict this rule to typed stdlib formatted writes with a syntactically
// discarded error. It does not prove later uses of assigned error bindings.
func (w *errorCheckWalker) discardsOutput(call *ast.CallExpr) bool {
	if !formattedOutputCall(call, w.info) {
		return false
	}
	if memoryOutput(w.info.TypeOf(call.Args[0])) {
		return false
	}
	switch parent := w.callParent().(type) {
	case *ast.ExprStmt:
		return true
	case *ast.AssignStmt:
		return discardedOutputAssignment(parent)
	case *ast.ValueSpec:
		return discardedOutputDeclaration(parent)
	default:
		return false
	}
}

func formattedOutputCall(call *ast.CallExpr, info *types.Info) bool {
	var object types.Object
	switch fun := specUnwrapParens(call.Fun).(type) {
	case *ast.Ident:
		object = info.ObjectOf(fun)
	case *ast.SelectorExpr:
		object = info.ObjectOf(fun.Sel)
	}
	function, ok := object.(*types.Func)
	if !ok || function.Pkg() == nil {
		return false
	}
	return function.Pkg().Path() == "fmt" && slices.Contains([]string{"Fprint", "Fprintf", "Fprintln"}, function.Name())
}

func memoryOutput(typ types.Type) bool {
	pointer, ok := types.Unalias(typ).(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := types.Unalias(pointer.Elem()).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}
	key := named.Obj().Pkg().Path() + "." + named.Obj().Name()
	return slices.Contains([]string{"bytes.Buffer", "strings.Builder"}, key)
}

func discardedOutputAssignment(assign *ast.AssignStmt) bool {
	if len(assign.Rhs) != 1 || len(assign.Lhs) != 2 {
		return false
	}
	id, ok := assign.Lhs[1].(*ast.Ident)
	return ok && id.Name == "_"
}

func discardedOutputDeclaration(declaration *ast.ValueSpec) bool {
	if len(declaration.Values) != 1 || len(declaration.Names) != 2 {
		return false
	}
	return declaration.Names[1].Name == "_"
}
