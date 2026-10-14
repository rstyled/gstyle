package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

// findNilGuards flags functions where a nil guard checks one pointer while a
// different pointer is dereferenced without a guard of its own. Detection-only:
// which variable the author meant to guard is intent, not syntax.
func findNilGuards(pkg *packages.Package, root string, paths []string, sources *pdg.SourceContext) []patterns.Candidate {
	var out []patterns.Candidate
	for _, file := range pkg.Syntax {
		out = append(out, nilGuardsForFile(pkg, file, root, paths)...)
	}
	return out
}

func nilGuardsForFile(pkg *packages.Package, file *ast.File, root string, paths []string) []patterns.Candidate {
	path := pkg.Fset.PositionFor(file.Pos(), false).Filename
	if !declarationPathSelected(root, path, paths) {
		return nil
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil
	}
	rel = filepath.ToSlash(rel)
	var out []patterns.Candidate
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		out = append(out, nilGuardCandidates(pkg, fn, rel)...)
	}
	return out
}

func nilGuardCandidates(pkg *packages.Package, fn *ast.FuncDecl, path string) []patterns.Candidate {
	guarded := collectNilGuarded(fn, pkg.TypesInfo)
	if len(guarded) == 0 {
		return nil
	}
	var out []patterns.Candidate
	for _, deref := range collectUnguardedDerefs(fn, pkg.TypesInfo, guarded) {
		out = append(out, buildNilGuardCandidate(pkg, fn, path, guardedNames(guarded), deref))
	}
	return out
}

// collectNilGuarded returns the pointer-typed objects guarded by a returning
// `x == nil` check in the function body.
func collectNilGuarded(fn *ast.FuncDecl, info *types.Info) map[types.Object]bool {
	guarded := map[types.Object]bool{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		stmt, ok := node.(*ast.IfStmt)
		if !ok {
			return true
		}
		if id := nilGuardTarget(stmt.Cond); id != nil {
			if obj := info.ObjectOf(id); isPointerVar(obj) && branchReturns(stmt.Body) {
				guarded[obj] = true
			}
		}
		return true
	})
	return guarded
}

type nilDeref struct {
	obj  types.Object
	name string
	pos  token.Pos
}

// collectUnguardedDerefs returns the first dereference site per pointer object
// that has no guard in the function.
func collectUnguardedDerefs(fn *ast.FuncDecl, info *types.Info, guarded map[types.Object]bool) []nilDeref {
	seen := map[types.Object]bool{}
	var out []nilDeref
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		if deref, ok := unguardedDeref(node, info, guarded, seen); ok {
			seen[deref.obj] = true
			out = append(out, deref)
		}
		return true
	})
	return out
}

// unguardedDeref reports node as an unguarded pointer dereference when it is
// a first-seen `*x` over a pointer object with no guard.
func unguardedDeref(node ast.Node, info *types.Info, guarded, seen map[types.Object]bool) (nilDeref, bool) {
	star, ok := node.(*ast.StarExpr)
	if !ok {
		return nilDeref{}, false
	}
	id, ok := star.X.(*ast.Ident)
	if !ok {
		return nilDeref{}, false
	}
	obj := info.ObjectOf(id)
	if !isPointerVar(obj) || guarded[obj] || seen[obj] {
		return nilDeref{}, false
	}
	return nilDeref{obj: obj, name: id.Name, pos: star.Pos()}, true
}

func buildNilGuardCandidate(pkg *packages.Package, fn *ast.FuncDecl, path, guards string, deref nilDeref) patterns.Candidate {
	name := fn.Name.Name
	if fnObj, ok := pkg.TypesInfo.Defs[fn.Name].(*types.Func); ok {
		name = patterns.FuncID(fnObj)
	}
	line := pkg.Fset.PositionFor(deref.pos, false).Line
	return patterns.Candidate{
		Kind:       patterns.NilGuard,
		ScoreMilli: 800,
		Observation: "nil guard checks " + guards + " while " + deref.name +
			" is dereferenced without a guard",
		Inference: "verify the guard targets the dereferenced pointer; the checked variable may be the wrong one",
		CounterEvidence: []string{
			"intent unknown; no automatic fix — verify against the contract",
		},
		Sites: []patterns.Site{{
			Path: path, Line: line, EndLine: line,
			ID: name, Name: name,
		}},
	}
}

// nilGuardTarget returns the identifier compared against nil in `x == nil`
// or `nil == x`, or nil.
func nilGuardTarget(cond ast.Expr) *ast.Ident {
	bin, ok := cond.(*ast.BinaryExpr)
	if !ok || bin.Op != token.EQL {
		return nil
	}
	if other := nilCounterpart(bin.X, bin.Y); other != nil {
		return other
	}
	return nilCounterpart(bin.Y, bin.X)
}

// nilCounterpart returns side as an identifier when other is the nil
// identifier, or nil.
func nilCounterpart(side, other ast.Expr) *ast.Ident {
	id, ok := other.(*ast.Ident)
	if !ok || !isNilIdent(id) {
		return nil
	}
	otherID, ok := side.(*ast.Ident)
	if !ok {
		return nil
	}
	return otherID
}

func isPointerVar(obj types.Object) bool {
	v, ok := obj.(*types.Var)
	if !ok {
		return false
	}
	_, ok = v.Type().Underlying().(*types.Pointer)
	return ok
}

func branchReturns(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		if _, ok := n.(*ast.ReturnStmt); ok {
			found = true
			return false
		}
		return true
	})
	return found
}

func guardedNames(guarded map[types.Object]bool) string {
	names := make([]string, 0, len(guarded))
	for obj := range guarded {
		names = append(names, obj.Name())
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
