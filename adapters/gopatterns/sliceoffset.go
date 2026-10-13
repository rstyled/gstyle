package gopatterns

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
	"golang.org/x/tools/go/packages"
)

type sliceOffsetSource struct {
	pkg  *packages.Package
	path string
	src  []byte
}

func findSliceOffsets(pkg *packages.Package, root string, paths []string, identities map[string]string) []patterns.Candidate {
	var out []patterns.Candidate
	for _, file := range pkg.Syntax {
		path := pkg.Fset.PositionFor(file.Pos(), false).Filename
		if !declarationPathSelected(root, path, paths) {
			continue
		}
		out = append(out, sliceOffsetsForFile(pkg, file, root, identities[path])...)
	}
	return out
}

func sliceOffsetsForFile(pkg *packages.Package, file *ast.File, root, identity string) []patterns.Candidate {
	path := pkg.Fset.PositionFor(file.Pos(), false).Filename
	src, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	if fmt.Sprintf("%x", sha256.Sum256(src)) != identity {
		return nil
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil
	}
	source := sliceOffsetSource{pkg: pkg, path: filepath.ToSlash(rel), src: src}
	var out []patterns.Candidate
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok {
			out = append(out, sliceOffsetCandidates(fn, source)...)
		}
	}
	return out
}

func sliceOffsetCandidates(fn *ast.FuncDecl, source sliceOffsetSource) []patterns.Candidate {
	var out []patterns.Candidate
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		stmt, ok := node.(*ast.IfStmt)
		if !ok {
			return true
		}
		guard := matchSliceOffset(stmt, source.pkg.TypesInfo)
		if guard != nil {
			out = append(out, sliceOffsetCandidate(fn, guard, source))
		}
		return true
	})
	return out
}

func sliceOffsetCandidate(fn *ast.FuncDecl, guard *sliceOffset, source sliceOffsetSource) patterns.Candidate {
	fset := source.pkg.Fset
	line := fset.PositionFor(guard.stmt.Pos(), false).Line
	name := fn.Name.Name
	if obj, ok := source.pkg.TypesInfo.Defs[fn.Name].(*types.Func); ok {
		name = patterns.FuncID(obj)
	}
	candidate := patterns.Candidate{
		Kind: patterns.SliceOffsetFallback, ScoreMilli: 800,
		Observation: "an upper-bound guard skips a suffix slice when the offset exceeds the collection length",
		Inference:   "without an empty-result branch, an out-of-range offset retains the original collection; verify the intended fallback",
		Sites:       []patterns.Site{{Path: source.path, Line: line, EndLine: fset.PositionFor(guard.stmt.End(), false).Line, ID: name, Name: name}},
	}
	spec := sliceOffsetSpec(guard, source)
	fixed, err := sliceOffsetRewrite(spec, source.src)
	if err != nil {
		candidate.CounterEvidence = []string{err.Error()}
		return candidate
	}
	before := sliceOffsetText(source.src, fset, guard.stmt)
	start := fset.PositionFor(guard.stmt.Pos(), false).Offset
	end := fset.PositionFor(guard.stmt.End(), false).Offset + len(fixed) - len(source.src)
	after := strings.TrimSpace(string(fixed[start:end]))
	candidate.PossibleRefactor = "review this empty-result proposal:\n" + sliceOffsetDiff(before, after)
	if sliceOffsetAutoSafe(guard, source.pkg) && !guardHeaderHasComments(guard.stmt, fset, source.src) {
		candidate.FixSpec = spec
		candidate.Inference = "the existing end-slice fallback explicitly selects empty results; clamping preserves that behavior"
	} else {
		candidate.CounterEvidence = []string{"fallback intent or rewrite safety is not established; proposal only, never applied by fix --apply"}
	}
	return candidate
}

func sliceOffsetText(src []byte, fset *token.FileSet, node ast.Node) string {
	return strings.TrimSpace(string(src[fset.PositionFor(node.Pos(), false).Offset:fset.PositionFor(node.End(), false).Offset]))
}

func sliceOffsetDiff(before, after string) string {
	return "- " + strings.ReplaceAll(before, "\n", "\n- ") + "\n+ " + strings.ReplaceAll(after, "\n", "\n+ ")
}

func sliceOffsetSpec(guard *sliceOffset, source sliceOffsetSource) *patterns.FixSpec {
	fset := source.pkg.Fset
	return &patterns.FixSpec{
		Kind: patterns.SliceOffsetFallback, File: source.path,
		Line: fset.PositionFor(guard.stmt.Pos(), false).Line,
		Params: map[string]string{
			"source_sha256":   fmt.Sprintf("%x", sha256.Sum256(source.src)),
			"condition_start": fmt.Sprint(fset.PositionFor(guard.stmt.Cond.Pos(), false).Offset),
			"condition_end":   fmt.Sprint(fset.PositionFor(guard.stmt.Cond.End(), false).Offset),
			"condition":       sliceOffsetText(source.src, fset, guard.lower),
			"index_start":     fmt.Sprint(fset.PositionFor(guard.slice.Low.Pos(), false).Offset),
			"index_end":       fmt.Sprint(fset.PositionFor(guard.slice.Low.End(), false).Offset),
			"index":           "min(" + sliceOffsetText(source.src, fset, guard.slice.Low) + ", len(" + sliceOffsetText(source.src, fset, guard.slice.X) + "))",
		},
	}
}
