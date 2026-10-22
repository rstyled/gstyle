package goquality

import (
	"go/ast"
	"go/constant"
	"go/types"

	"golang.org/x/tools/go/packages"
)

const lipglossPackage = "charm.land/lipgloss/v2"

func styleValueType(typ types.Type) bool {
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok {
		return false
	}
	object := named.Obj()
	return object.Pkg() != nil && object.Pkg().Path() == lipglossPackage && object.Name() == "Style"
}

// In v2.0.6, this exact chain fills a zero Style with a value color.
// Other Style values can hold mutable references or callbacks. Calls on the
// proven value retain their effects; only the global read is stable.
func stableStyle(pkg *packages.Package, value ast.Expr) bool {
	if !auditedStyleModule(pkg) {
		return false
	}
	foreground := styleCall(value, "Style.Foreground", 1, pkg.TypesInfo)
	if foreground == nil {
		return false
	}
	selector := ast.Unparen(foreground.Fun).(*ast.SelectorExpr)
	if styleCall(selector.X, "NewStyle", 0, pkg.TypesInfo) == nil {
		return false
	}
	return constantStyleColor(foreground.Args[0], pkg.TypesInfo) || privateStyleColor(pkg, foreground.Args[0])
}

func constantStyleColor(value ast.Expr, info *types.Info) bool {
	color := styleCall(value, "Color", 1, info)
	if color == nil {
		return false
	}
	constantColor := info.Types[color.Args[0]].Value
	return constantColor != nil && constantColor.Kind() == constant.String
}

// A private color consumed only by this initializer cannot be rebound or have
// its slot exposed. Keep aliases and other uses outside this bounded proof.
func privateStyleColor(pkg *packages.Package, value ast.Expr) bool {
	id, _ := ast.Unparen(value).(*ast.Ident)
	variable, ok := pkg.TypesInfo.Uses[id].(*types.Var)
	if !ok || variable.Exported() || !packageVariable(variable) || variable.Pkg() != pkg.Types {
		return false
	}
	return soleColorUse(pkg.TypesInfo, variable, id) && colorDeclaration(pkg, variable)
}

func soleColorUse(info *types.Info, variable *types.Var, current *ast.Ident) bool {
	for use, object := range info.Uses {
		if object == variable && use != current {
			return false
		}
	}
	return true
}

func colorDeclaration(pkg *packages.Package, variable *types.Var) bool {
	found := false
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			spec, ok := node.(*ast.ValueSpec)
			if ok && len(spec.Names) == 1 && len(spec.Values) == 1 && pkg.TypesInfo.Defs[spec.Names[0]] == variable {
				found = constantStyleColor(spec.Values[0], pkg.TypesInfo)
			}
			return !found
		})
	}
	return found
}

func auditedStyleModule(pkg *packages.Package) bool {
	dependency := pkg.Imports[lipglossPackage]
	if dependency == nil || dependency.Module == nil {
		return false
	}
	module := dependency.Module
	return module.Path == lipglossPackage && module.Version == "v2.0.6" && module.Replace == nil
}

func styleCall(value ast.Expr, name string, arity int, info *types.Info) *ast.CallExpr {
	call, ok := ast.Unparen(value).(*ast.CallExpr)
	if !ok || len(call.Args) != arity {
		return nil
	}
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	function, ok := info.ObjectOf(selector.Sel).(*types.Func)
	if !ok || resolvedName(function) != lipglossPackage+"."+name {
		return nil
	}
	return call
}
