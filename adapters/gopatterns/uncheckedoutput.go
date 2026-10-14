package gopatterns

import (
	"go/ast"
	"go/types"
	"slices"
)

// Restrict this rule to typed stdlib formatted writes with a syntactically
// discarded error. It does not prove later uses of assigned error bindings.
func (w *errorCheckWalker) discardsOutput(call *ast.CallExpr) bool {
	input := outputDecision{Formatted: formattedOutputCall(call, w.info)}
	if !input.Formatted {
		return discardsOutputDecision(input)
	}
	input.Memory = memoryOutput(w.info.TypeOf(call.Args[0]))
	return discardsOutputDecision(w.outputDecision(input))
}

// This adapter remains outside the formal proof. It supplies syntactic counts
// and blank-target identity to the verified, import-free decision function.
func (w *errorCheckWalker) outputDecision(input outputDecision) outputDecision {
	switch parent := w.callParent().(type) {
	case *ast.ExprStmt:
		input.Standalone = true
	case *ast.AssignStmt:
		input.Values, input.Results = len(parent.Rhs), len(parent.Lhs)
		input.ErrorBlank = blankOutputTarget(parent.Lhs)
	case *ast.ValueSpec:
		input.Values, input.Results = len(parent.Values), len(parent.Names)
		if len(parent.Names) == 2 {
			input.ErrorBlank = parent.Names[1].Name == "_"
		}
	}
	return input
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

func blankOutputTarget(targets []ast.Expr) bool {
	if len(targets) != 2 {
		return false
	}
	id, ok := targets[1].(*ast.Ident)
	return ok && id.Name == "_"
}
