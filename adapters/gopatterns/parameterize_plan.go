package gopatterns

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

// PlanParameterizationFix renders both wrappers from one immutable source
// revision. It uses the stored source projection and resolved import uses.
func PlanParameterizationFix(spec *patterns.FixSpec) (map[string][]byte, error) {
	if spec.Context == nil {
		return nil, fmt.Errorf("parameterization source context is missing")
	}
	if err := CheckFixSources(spec.Context); err != nil {
		return nil, err
	}
	first, second, signatures, err := parameterizationPlanSources(spec)
	if err != nil {
		return nil, err
	}
	pair, err := parameterizationSourcePair(spec, first, second)
	if err != nil {
		return nil, err
	}
	ctx, err := parameterizationPlanContext(pair, signatures, spec.Params["helper_name"])
	if err != nil {
		return nil, err
	}
	return renderParameterizationPair(ctx, first, second)
}

func parameterizationSourcePair(spec *patterns.FixSpec, first, second *parameterizationSource) (*parameterizationPair, error) {
	a, b := namedFreeFunction(first.file, spec.Params["pair_a"]), namedFreeFunction(second.file, spec.Params["pair_b"])
	if a == nil || b == nil {
		return nil, fmt.Errorf("parameterization source functions are missing")
	}
	if !safeExtractionBodies(a, b) || !matchingCallSpreads(a, b) {
		return nil, fmt.Errorf("parameterization source shape differs from proof")
	}
	return &parameterizationPair{first: first, second: second, a: a, b: b}, nil
}

func parameterizationPlanContext(pair *parameterizationPair, signatures map[string]string, helperName string) (*paramCtx, error) {
	normalizeParameterizationImports(pair)
	renames, holes, ok := diffPair(pair.a, pair.b)
	if !ok {
		return nil, fmt.Errorf("parameterization substitutions differ from proof")
	}
	if err := validateParameterizationImports(pair.first, pair.second, nodeByteRange(pair.first.fset, pair.a.Body), nodeByteRange(pair.first.fset, pair.b.Body)); err != nil {
		return nil, err
	}
	return &paramCtx{fset: pair.first.fset, a: pair.a, b: pair.b, src: pair.first.src, comments: pair.first.file.Comments, r: renames, holes: holes, signatures: signatures, extractedName: helperName}, nil
}

func parameterizationPlanSources(spec *patterns.FixSpec) (*parameterizationSource, *parameterizationSource, map[string]string, error) {
	fset := token.NewFileSet()
	files := map[string]*parameterizationSource{}
	first, err := parameterizationFile(spec.File, files, nil, spec.Context, fset)
	if err != nil {
		return nil, nil, nil, err
	}
	_, _, signatures, err := parameterizationProof(spec, first.src)
	if err != nil {
		return nil, nil, nil, err
	}
	second, err := parameterizationFile(spec.Params["secondary_file"], files, nil, spec.Context, fset)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := checkParameterizationSecondary(first, second, spec); err != nil {
		return nil, nil, nil, err
	}
	return first, second, signatures, nil
}

func renderParameterizationPair(ctx *paramCtx, first, second *parameterizationSource) (map[string][]byte, error) {
	primary, ok := buildPrimaryReplacement(ctx)
	if !ok {
		return nil, fmt.Errorf("parameterization helper cannot be rendered")
	}
	edits, err := parameterizationImportEdits(second, nodeByteRange(ctx.fset, ctx.b.Body))
	if err != nil {
		return nil, err
	}
	other := *ctx
	other.src, other.comments = second.src, second.file.Comments
	var wrapper bytes.Buffer
	writeWrapper(&wrapper, &other, ctx.b, false)
	edits = append(edits, parameterizationDeclarationEdit(&other, ctx.b, wrapper.Bytes()))
	afterSecond, err := renderParameterizationEdits(second.snapshot, edits)
	if err != nil {
		return nil, err
	}
	afterFirst, err := renderParameterizationEdits(first.snapshot, []pdg.SourceEdit{parameterizationDeclarationEdit(ctx, ctx.a, primary)})
	if err != nil {
		return nil, err
	}
	return map[string][]byte{first.snapshot.Path: afterFirst, second.snapshot.Path: afterSecond}, nil
}

func renderParameterizationEdits(snapshot pdg.SourceSnapshot, edits []pdg.SourceEdit) ([]byte, error) {
	text, err := pdg.ApplySourceEdits(snapshot, edits)
	if err != nil {
		return nil, err
	}
	return format.Source([]byte(text))
}

func parameterizationDeclarationEdit(ctx *paramCtx, fn *ast.FuncDecl, text []byte) pdg.SourceEdit {
	return pdg.SourceEdit{Range: pdg.ByteRange{Start: uint32(declarationStart(ctx.fset, fn)), End: uint32(ctx.fset.PositionFor(fn.End(), false).Offset)}, Replacement: string(text)}
}

func checkParameterizationSecondary(first, second *parameterizationSource, spec *patterns.FixSpec) error {
	if first == second || second.digest != spec.Params["secondary_sha256"] {
		return fmt.Errorf("parameterization requires current secondary source proof")
	}
	if sourceHasLineDirectives(first.snapshot) || sourceHasLineDirectives(second.snapshot) {
		return unsupportedFix("source line directives prevent helper extraction")
	}
	return nil
}
