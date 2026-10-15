package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/patterns/outputcheck"
)

func (w *errorCheckWalker) discardsOutput(call *ast.CallExpr) bool {
	callee := outputCalleeIdentity(call, w.info)
	var writer outputcheck.Identity
	if len(call.Args) > 0 {
		writer = outputWriterIdentity(w.info.TypeOf(call.Args[0]))
	}
	return outputcheck.Classify(callee, writer, w.outputConsumption())
}

func (w *errorCheckWalker) outputConsumption() outputcheck.Consumption {
	switch parent := w.callParent().(type) {
	case *ast.ExprStmt:
		return outputcheck.Consumption{Standalone: true}
	case *ast.AssignStmt:
		return outputcheck.Consumption{Values: len(parent.Rhs), Results: len(parent.Lhs), ErrorName: outputErrorTarget(parent.Lhs)}
	case *ast.ValueSpec:
		var name string
		if len(parent.Names) == 2 {
			name = parent.Names[1].Name
		}
		return outputcheck.Consumption{Values: len(parent.Values), Results: len(parent.Names), ErrorName: name}
	default:
		return outputcheck.Consumption{}
	}
}

func outputCalleeIdentity(call *ast.CallExpr, info *types.Info) outputcheck.Identity {
	var object types.Object
	switch fun := specUnwrapParens(call.Fun).(type) {
	case *ast.Ident:
		object = info.ObjectOf(fun)
	case *ast.SelectorExpr:
		object = info.ObjectOf(fun.Sel)
	}
	function, ok := object.(*types.Func)
	if !ok || function.Pkg() == nil {
		return outputcheck.Identity{}
	}
	return outputcheck.Identity{Package: function.Pkg().Path(), Name: function.Name()}
}

func outputWriterIdentity(typ types.Type) outputcheck.Identity {
	pointer, ok := types.Unalias(typ).(*types.Pointer)
	if !ok {
		return outputcheck.Identity{}
	}
	named, ok := types.Unalias(pointer.Elem()).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return outputcheck.Identity{}
	}
	return outputcheck.Identity{Package: named.Obj().Pkg().Path(), Name: named.Obj().Name()}
}

func outputErrorTarget(targets []ast.Expr) string {
	if len(targets) != 2 {
		return ""
	}
	id, ok := specUnwrapParens(targets[1]).(*ast.Ident)
	if !ok {
		return ""
	}
	return id.Name
}
