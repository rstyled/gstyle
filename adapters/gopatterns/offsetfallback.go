package gopatterns

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

type offsetFallbackSource struct {
	pkg      *packages.Package
	path     string
	src      []byte
	comments []pdg.ByteRange
}

func findOffsetFallbacks(pkg *packages.Package, root string, paths []string, sources *pdg.SourceContext) []patterns.Candidate {
	var out []patterns.Candidate
	for _, file := range pkg.Syntax {
		path := pkg.Fset.PositionFor(file.Pos(), false).Filename
		if !declarationPathSelected(root, path, paths) {
			continue
		}
		out = append(out, offsetFallbacksForFile(pkg, file, root, sources)...)
	}
	return out
}

func offsetFallbacksForFile(pkg *packages.Package, file *ast.File, root string, sources *pdg.SourceContext) []patterns.Candidate {
	path := pkg.Fset.PositionFor(file.Pos(), false).Filename
	src, err := revisionSource(path, sources)
	if err != nil {
		return nil
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil
	}
	source := offsetFallbackSource{pkg: pkg, path: filepath.ToSlash(rel), src: src, comments: sourceCommentRanges(pkg.Fset, file, src)}
	var out []patterns.Candidate
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok {
			out = append(out, offsetFallbackCandidates(fn, source)...)
		}
	}
	return out
}

func offsetFallbackCandidates(fn *ast.FuncDecl, source offsetFallbackSource) []patterns.Candidate {
	var out []patterns.Candidate
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		stmt, ok := node.(*ast.IfStmt)
		if !ok {
			return true
		}
		guard := matchOffsetFallback(stmt, source.pkg.TypesInfo)
		if guard != nil {
			out = append(out, offsetFallbackCandidate(fn, guard, source))
		}
		return true
	})
	return out
}

func offsetFallbackCandidate(fn *ast.FuncDecl, guard *offsetFallback, source offsetFallbackSource) patterns.Candidate {
	fset := source.pkg.Fset
	line := fset.PositionFor(guard.stmt.Pos(), false).Line
	name := fn.Name.Name
	if obj, ok := source.pkg.TypesInfo.Defs[fn.Name].(*types.Func); ok {
		name = patterns.FuncID(obj)
	}
	candidate := patterns.Candidate{
		Kind: patterns.OffsetFallback, ScoreMilli: 800,
		Observation: "an upper-bound guard skips a suffix slice when the offset exceeds the collection length",
		Inference:   "without an empty-result branch, an out-of-range offset retains the original collection; verify the intended fallback",
		Sites:       []patterns.Site{{Path: source.path, Line: line, EndLine: fset.PositionFor(guard.stmt.End(), false).Line, ID: name, Name: name}},
	}
	spec := offsetFallbackSpec(guard, source)
	fixed, err := offsetFallbackRewrite(spec, source.src)
	if err != nil {
		candidate.CounterEvidence = []string{err.Error()}
		return candidate
	}
	before := offsetFallbackText(source.src, fset, guard.stmt)
	start := fset.PositionFor(guard.stmt.Pos(), false).Offset
	end := fset.PositionFor(guard.stmt.End(), false).Offset + len(fixed) - len(source.src)
	after := strings.TrimSpace(string(fixed[start:end]))
	candidate.PossibleRefactor = "review this empty-result proposal:\n" + offsetFallbackDiff(before, after)
	if offsetFallbackAutoSafe(guard, source.pkg) && !guardHeaderHasComments(guard.stmt, fset, pdg.SourceSnapshot{Comments: source.comments}) {
		candidate.FixSpec = spec
		candidate.Inference = "the existing end-slice fallback explicitly selects empty results; clamping preserves that behavior"
	} else {
		candidate.CounterEvidence = []string{"fallback intent or rewrite safety is not established; proposal only, never applied by fix --apply"}
	}
	return candidate
}

func offsetFallbackText(src []byte, fset *token.FileSet, node ast.Node) string {
	return strings.TrimSpace(string(src[fset.PositionFor(node.Pos(), false).Offset:fset.PositionFor(node.End(), false).Offset]))
}

func offsetFallbackDiff(before, after string) string {
	return "- " + strings.ReplaceAll(before, "\n", "\n- ") + "\n+ " + strings.ReplaceAll(after, "\n", "\n+ ")
}

func offsetFallbackSpec(guard *offsetFallback, source offsetFallbackSource) *patterns.FixSpec {
	fset := source.pkg.Fset
	return &patterns.FixSpec{
		Kind: patterns.OffsetFallback, File: source.path,
		Line: fset.PositionFor(guard.stmt.Pos(), false).Line,
		Params: map[string]string{
			"source_sha256":   fmt.Sprintf("%x", sha256.Sum256(source.src)),
			"condition_start": fmt.Sprint(fset.PositionFor(guard.stmt.Cond.Pos(), false).Offset),
			"condition_end":   fmt.Sprint(fset.PositionFor(guard.stmt.Cond.End(), false).Offset),
			"condition":       offsetFallbackText(source.src, fset, guard.lower),
			"index_start":     fmt.Sprint(fset.PositionFor(guard.slice.Low.Pos(), false).Offset),
			"index_end":       fmt.Sprint(fset.PositionFor(guard.slice.Low.End(), false).Offset),
			"index":           "min(" + offsetFallbackText(source.src, fset, guard.slice.Low) + ", len(" + offsetFallbackText(source.src, fset, guard.slice.X) + "))",
		},
	}
}
