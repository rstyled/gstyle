package gopatterns

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/typeutil"
)

// Bounded source summaries cover direct and unmodified local file returns. Cycles without a
// known acquisition never become sources; dynamic user interfaces stay unknown.
func returnedFileSources(pkg *packages.Package) map[types.Object]bool {
	sources := map[types.Object]bool{}
	for range 8 {
		changed := false
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				if addFileSource(pkg, decl, sources) {
					changed = true
				}
			}
		}
		if !changed {
			break
		}
	}
	return sources
}

func addFileSource(pkg *packages.Package, decl ast.Decl, sources map[types.Object]bool) bool {
	fn, ok := decl.(*ast.FuncDecl)
	if !ok || fn.Body == nil {
		return false
	}
	object := pkg.TypesInfo.Defs[fn.Name]
	if sources[object] || !forwardsFile(pkg.TypesInfo, fn, sources) {
		return false
	}
	sources[object] = true
	return true
}

func forwardsFile(info *types.Info, fn *ast.FuncDecl, sources map[types.Object]bool) bool {
	locals := fileSourceLocals(info, fn, sources)
	valid, acquires := true, false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		if ret, ok := node.(*ast.ReturnStmt); ok {
			kind := fileReturnKind(info, ret, sources, locals)
			valid = valid && kind != 0
			acquires = acquires || kind == 2
		}
		return valid
	})
	return valid && acquires
}

// Return kinds: 0 unknown, 1 nil resource, 2 acquired resource.
func fileReturnKind(info *types.Info, ret *ast.ReturnStmt, sources map[types.Object]bool, locals map[types.Object]bool) uint8 {
	if len(ret.Results) == 2 {
		if returnedLocalFile(info, ret.Results[0], locals) {
			return 2
		}
		if typedNil(info, ret.Results[0]) {
			return 1
		}
		return 0
	}
	if len(ret.Results) != 1 {
		return 0
	}
	return directFileReturn(info, ret.Results[0], sources)
}

func directFileReturn(info *types.Info, expr ast.Expr, sources map[types.Object]bool) uint8 {
	call, ok := ast.Unparen(expr).(*ast.CallExpr)
	if ok && fileSourceCall(info, call, sources) {
		return 2
	}
	return 0
}

func fileSourceCall(info *types.Info, call *ast.CallExpr, sources map[types.Object]bool) bool {
	if osFileCall(info, call) {
		return true
	}
	function := fileCallObject(info, call)
	if function == nil {
		return false
	}
	return sources[function] || function.FullName() == "(io/fs.FS).Open"
}

func fileCallObject(info *types.Info, call *ast.CallExpr) *types.Func {
	function, _ := typeutil.Callee(info, call).(*types.Func)
	return function
}

func returnedLocalFile(info *types.Info, expr ast.Expr, locals map[types.Object]bool) bool {
	name, ok := ast.Unparen(expr).(*ast.Ident)
	return ok && locals[info.ObjectOf(name)]
}

func fileSourceLocals(info *types.Info, fn *ast.FuncDecl, sources map[types.Object]bool) map[types.Object]bool {
	locals := map[types.Object]bool{}
	writes := nilErrWrites(fn, info)
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if ok && sourceAssignment(info, assign, sources) != nil {
			object := info.Defs[assign.Lhs[0].(*ast.Ident)]
			locals[object] = writes[object] == 1
		}
		return true
	})
	// Only return-slot uses may forward ownership. Closing, aliasing or
	// capturing the handle prevents a fresh-file summary.
	if len(locals) == 0 {
		return locals
	}
	allowed := returnedFileNames(fn)
	for name, object := range info.Uses {
		if !allowed[name] {
			delete(locals, object)
		}
	}
	return locals
}

func sourceAssignment(info *types.Info, assign *ast.AssignStmt, sources map[types.Object]bool) *ast.CallExpr {
	if len(assign.Lhs) != 2 || len(assign.Rhs) != 1 || !newResourceBindings(info, assign.Lhs) {
		return nil
	}
	call, ok := ast.Unparen(assign.Rhs[0]).(*ast.CallExpr)
	if !ok || !fileSourceCall(info, call, sources) {
		return nil
	}
	return call
}

func returnedFileNames(fn *ast.FuncDecl) map[*ast.Ident]bool {
	names := map[*ast.Ident]bool{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		ret, ok := node.(*ast.ReturnStmt)
		if ok && len(ret.Results) > 0 {
			name, _ := ast.Unparen(ret.Results[0]).(*ast.Ident)
			names[name] = true
		}
		return true
	})
	return names
}
