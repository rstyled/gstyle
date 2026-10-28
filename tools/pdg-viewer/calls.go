package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/typeutil"
)

const typedLoadMode = packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo

// Call kinds. A resolved call to an indexed declaration has a Target instead.
const (
	callExternal = "external" // static callee outside the repository
	callDynamic  = "dynamic"  // interface method or function value
)

type typedFile struct {
	root      string
	positions *token.FileSet
	info      *types.Info
	symbols   map[string]int
}

// resolveTypedCalls records only type-checked static callees.
// Builtins and conversions are not calls. Interface and function-value calls stay dynamic.
// gstyle-allow(side_effect_density): Type checking runs the Go toolchain to load the repository's packages.
func resolveTypedCalls(index *repoIndex) error {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pkgs, err := packages.Load(&packages.Config{Context: ctx, Mode: typedLoadMode, Dir: index.Root}, "./...")
	if err != nil {
		return err
	}
	applyTypedPackages(index, pkgs)
	index.CallResolution = "typed"
	for _, s := range index.Symbols {
		if s.Resolution != "typed" {
			index.CallResolution = "partial"
			return fmt.Errorf("some declarations are outside the active build or have type errors; their calls are unavailable")
		}
	}
	return nil
}

// gstyle-allow(side_effect_density): Applies type-checked package data to the scan-owned index; packages with errors keep unavailable calls.
func applyTypedPackages(index *repoIndex, pkgs []*packages.Package) {
	symbols := make(map[string]int, len(index.Symbols))
	for i, s := range index.Symbols {
		symbols[s.ID] = i
	}
	for _, p := range pkgs {
		if len(p.Errors) != 0 || p.IllTyped {
			continue
		}
		file := typedFile{index.Root, p.Fset, p.TypesInfo, symbols}
		for _, syntax := range p.Syntax {
			file.resolve(index, syntax)
		}
	}
}

// gstyle-allow(side_effect_density): Replaces each indexed declaration's calls with its typed calls.
func (f typedFile) resolve(index *repoIndex, syntax *ast.File) {
	if f.info == nil {
		return
	}
	for _, decl := range syntax.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if i, ok := f.symbols[f.symbolID(fn.Name.Pos())]; ok {
			index.Symbols[i].Calls = f.calls(fn.Body)
			index.Symbols[i].Resolution = "typed"
		}
	}
}

func (f typedFile) calls(body *ast.BlockStmt) []repoCall {
	calls := []repoCall{}
	ast.Inspect(body, func(n ast.Node) bool {
		expr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		call, keep := f.call(expr)
		if keep {
			calls = append(calls, call)
		}
		return true
	})
	return calls
}

func (f typedFile) call(call *ast.CallExpr) (repoCall, bool) {
	if tv, ok := f.info.Types[call.Fun]; ok && tv.IsType() {
		return repoCall{}, false
	}
	spelling := types.ExprString(call.Fun)
	switch callee := typeutil.Callee(f.info, call).(type) {
	case *types.Builtin:
		return repoCall{}, false
	case *types.Func:
		return f.funcCall(spelling, callee, typeutil.StaticCallee(f.info, call) != nil), true
	default:
		return repoCall{Spelling: spelling, Kind: callDynamic}, true
	}
}

func (f typedFile) funcCall(spelling string, callee *types.Func, static bool) repoCall {
	if !static {
		return repoCall{Spelling: spelling, Kind: callDynamic}
	}
	origin := callee.Origin()
	if id := f.symbolID(origin.Pos()); f.indexed(id) {
		return repoCall{Spelling: spelling, Target: id}
	}
	call := repoCall{Spelling: spelling, Kind: callExternal}
	if origin.Pkg() != nil {
		call.Package = origin.Pkg().Path()
	}
	return call
}

func (f typedFile) indexed(id string) bool {
	_, ok := f.symbols[id]
	return ok
}

func (f typedFile) symbolID(pos token.Pos) string {
	position := f.positions.Position(pos)
	rel, err := filepath.Rel(f.root, position.Filename)
	if err != nil || !filepath.IsLocal(rel) {
		return ""
	}
	return filepath.ToSlash(rel) + ":" + strconv.Itoa(position.Offset)
}
