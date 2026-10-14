package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

func collectSourceReferences(sources *pdg.SourceContext, pkgs []*packages.Package) {
	seen := make(map[string]bool)
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			path := pkg.Fset.PositionFor(file.Pos(), false).Filename
			if seen[path] {
				continue
			}
			seen[path] = true
			collectFileReferences(sources, pkg, file, sources.Reference(path))
		}
	}
}

func collectFileReferences(sources *pdg.SourceContext, pkg *packages.Package, file *ast.File, ref pdg.Ref) {
	sources.SetImports(ref, sourceImports(pkg, file))
	direct := make(map[*ast.Ident]*ast.CallExpr)
	ast.Inspect(file, func(node ast.Node) bool {
		recordDirectCall(direct, node)
		return true
	})
	ast.Inspect(file, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		_, ok = pkg.TypesInfo.Uses[ident].(*types.Func)
		if !ok {
			return true
		}
		sources.References = append(sources.References, sourceReference(pkg, ref, ident, direct[ident]))
		return true
	})
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		sources.Functions = append(sources.Functions, sourceFunction(pkg, fn, ref))
	}
}

func withinNode(node ast.Node, parent ast.Node) bool {
	return node.Pos() >= parent.Pos() && node.End() <= parent.End()
}

func recordDirectCall(direct map[*ast.Ident]*ast.CallExpr, node ast.Node) {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return
	}
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return
	}
	direct[ident] = call
}

// The caller accepts only compiler-resolved function uses.
func sourceReference(pkg *packages.Package, ref pdg.Ref, ident *ast.Ident, call *ast.CallExpr) pdg.SourceReference {
	fset := pkg.Fset
	function := pkg.TypesInfo.Uses[ident].(*types.Func)
	result := pdg.SourceReference{Target: strings.Clone(patterns.FuncID(function.Origin())), Source: ref, Range: nodeByteRange(fset, ident), Callee: nodeByteRange(fset, ident)}
	if call == nil {
		return result
	}
	return pdg.SourceReference{Target: result.Target, Source: ref, Range: nodeByteRange(fset, call), Callee: result.Callee, Arguments: sourceArgumentRanges(fset, call.Args), Receiver: sourceReceiver(pkg, function, call), Direct: true, Variadic: call.Ellipsis.IsValid()}
}

// Successful typed loading guarantees a function object and signature here.
func sourceFunction(pkg *packages.Package, fn *ast.FuncDecl, ref pdg.Ref) pdg.SourceFunction {
	object := pkg.TypesInfo.Defs[fn.Name].(*types.Func)
	signature := object.Type().(*types.Signature)
	result := pdg.SourceFunction{ID: strings.Clone(patterns.FuncID(object)), Name: strings.Clone(fn.Name.Name), Package: strings.Clone(pkg.PkgPath), Source: ref, Declaration: nodeByteRange(pkg.Fset, fn), Body: nodeByteRange(pkg.Fset, fn.Body), NameRange: nodeByteRange(pkg.Fset, fn.Name), Parameters: nodeByteRange(pkg.Fset, fn.Type.Params), Method: fn.Recv != nil, Generic: signature.TypeParams().Len() > 0, Variadic: signature.Variadic()}
	for _, field := range fn.Type.Params.List {
		for _, name := range field.Names {
			result.ParametersList = append(result.ParametersList, sourceParameter(pkg, fn, field, name))
		}
	}
	bindings := map[string]bool{}
	ast.Inspect(fn, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if ok && pkg.TypesInfo.ObjectOf(ident) != nil && !bindings[ident.Name] {
			bindings[ident.Name] = true
			result.Bindings = append(result.Bindings, strings.Clone(ident.Name))
		}
		return true
	})
	return result
}

func sourceParameter(pkg *packages.Package, fn *ast.FuncDecl, field *ast.Field, name *ast.Ident) pdg.SourceParameter {
	object := pkg.TypesInfo.Defs[name]
	result := pdg.SourceParameter{Captured: parameterCaptured(pkg, fn.Body, object), Name: strings.Clone(name.Name), Type: strings.Clone(types.TypeString(object.Type(), nil)), TypeSyntax: types.ExprString(field.Type)}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if parameterAddressed(pkg, node, object) {
			result.Addressed = true
		}
		ident, ok := node.(*ast.Ident)
		if ok && pkg.TypesInfo.Uses[ident] == object {
			result.Uses = append(result.Uses, nodeByteRange(pkg.Fset, ident))
		}
		return true
	})
	return result
}

func nodeByteRange(fset *token.FileSet, node ast.Node) pdg.ByteRange {
	return pdg.ByteRange{Start: uint32(fset.PositionFor(node.Pos(), false).Offset), End: uint32(fset.PositionFor(node.End(), false).Offset)}
}

func parameterAddressed(pkg *packages.Package, node ast.Node, object types.Object) bool {
	unary, ok := node.(*ast.UnaryExpr)
	if !ok || unary.Op != token.AND {
		return false
	}
	ident, ok := ast.Unparen(unary.X).(*ast.Ident)
	return ok && pkg.TypesInfo.Uses[ident] == object
}

func sourceArgumentRanges(fset *token.FileSet, arguments []ast.Expr) []pdg.ByteRange {
	result := make([]pdg.ByteRange, len(arguments))
	for i, arg := range arguments {
		result[i] = nodeByteRange(fset, arg)
	}
	return result
}

func parameterCaptured(pkg *packages.Package, body *ast.BlockStmt, object types.Object) bool {
	captured := false
	ast.Inspect(body, func(node ast.Node) bool {
		closure, ok := node.(*ast.FuncLit)
		if !ok {
			return true
		}
		ast.Inspect(closure.Body, func(child ast.Node) bool {
			ident, ok := child.(*ast.Ident)
			if ok && pkg.TypesInfo.Uses[ident] == object {
				captured = true
			}
			return true
		})
		return false
	})
	return captured
}
