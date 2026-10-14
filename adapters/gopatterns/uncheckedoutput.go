package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/patterns/outputcheck"
)

// Restrict this rule to typed stdlib formatted writes with a syntactically
// discarded error. It does not prove later uses of assigned error bindings.
func (w *errorCheckWalker) discardsOutput(call *ast.CallExpr) bool {
	input := w.outputDecision()
	input.Formatted = formattedOutputCall(call, w.info)
	if !input.Formatted {
		return outputcheck.Discards(input)
	}
	input.Memory = memoryOutput(w.info.TypeOf(call.Args[0]))
	return outputcheck.Discards(input)
}

func (w *errorCheckWalker) outputDecision() outputcheck.Decision {
	switch parent := w.callParent().(type) {
	case *ast.ExprStmt:
		return outputcheck.Decision{Standalone: true}
	case *ast.AssignStmt:
		return outputcheck.Decision{Values: len(parent.Rhs), Results: len(parent.Lhs), ErrorBlank: blankOutputTarget(parent.Lhs)}
	case *ast.ValueSpec:
		return outputcheck.Decision{Values: len(parent.Values), Results: len(parent.Names), ErrorBlank: len(parent.Names) == 2 && parent.Names[1].Name == "_"}
	default:
		return outputcheck.Decision{}
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
	return outputcheck.Formatted(function.Pkg().Path(), function.Name())
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
	return outputcheck.Memory(named.Obj().Pkg().Path(), named.Obj().Name())
}

func blankOutputTarget(targets []ast.Expr) bool {
	if len(targets) != 2 {
		return false
	}
	id, ok := specUnwrapParens(targets[1]).(*ast.Ident)
	return ok && id.Name == "_"
}
