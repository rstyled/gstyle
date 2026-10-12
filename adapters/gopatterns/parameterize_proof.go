package gopatterns

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

// VerifyParameterizations uses canonical IR facts, not a second compiler load.
// AST matching checks the complete source; it does not establish type safety.
func VerifyParameterizations(report *patterns.PatternsReport, extraction *Extraction) error {
	graphs := make(map[string]*pdg.Graph)
	for i := range extraction.Funcs {
		fp := &extraction.Funcs[i]
		graphs[fp.Name] = &fp.Pdg
	}
	files := make(map[string]*parameterizationSource)
	for i, candidate := range report.Candidates {
		if candidate.Kind != patterns.Parameterize {
			continue
		}
		verified, err := verifyParameterization(candidate, graphs, files)
		if err != nil {
			return err
		}
		report.Candidates[i] = verified
	}
	return nil
}

type parameterizationSource struct {
	fset         *token.FileSet
	file         *ast.File
	src          []byte
	digest       string
	dependencies map[*pdg.Graph]patterns.RefactorDependencies
}

func verifyParameterization(c patterns.Candidate, graphs map[string]*pdg.Graph, files map[string]*parameterizationSource) (patterns.Candidate, error) {
	c.FixSpec = nil
	if len(c.Sites) != 2 || c.Sites[0].Path != c.Sites[1].Path {
		return unverifiedParameterization(c, "automatic extraction requires exactly two functions in one file"), nil
	}
	left, right := graphs[c.Sites[0].ID], graphs[c.Sites[1].ID]
	if !compatibleInterfaces(left, right) {
		return unverifiedParameterization(c, "PDG function interfaces have unknown or incompatible types"), nil
	}
	source, err := parameterizationFile(c.Sites[0].Path, files)
	if err != nil {
		return c, err
	}
	return source.verifyCandidate(c, left, right), nil
}

func (source *parameterizationSource) verifyCandidate(c patterns.Candidate, left, right *pdg.Graph) patterns.Candidate {
	spec, reason := source.prove(c.Sites, left, right)
	if spec == nil {
		return unverifiedParameterization(c, reason)
	}
	c.DependencySafety = source.dependencySafety(left, right)
	if !completeDependencySafety(c.DependencySafety) {
		return unverifiedParameterization(c, "PDG dependency safety is incomplete; review binding, memory, call effects and execution coverage")
	}
	if !patterns.RefactorDependencyShapeMatches(left, right) {
		return unverifiedParameterization(c, "PDG bindings, definitions or directed dependency constraints differ between the source bodies")
	}
	return patterns.Candidate{
		Kind: c.Kind, ScoreMilli: c.ScoreMilli, Breakdown: c.Breakdown, Observation: c.Observation,
		Inference:        "complete source bodies match under substitutions with compatible PDG callable types",
		PossibleRefactor: "extract the verified pair into a shared helper with the differing callables as parameters",
		Sites:            c.Sites, Definitions: c.Definitions, CounterEvidence: c.CounterEvidence, FixSpec: dependencyAuthorizedSpec(spec),
		DependencySafety: c.DependencySafety,
	}
}

const dependencyPolicy = "gstyle.whole-body/0.1.0"

func dependencyAuthorizedSpec(spec *patterns.FixSpec) *patterns.FixSpec {
	params := maps.Clone(spec.Params)
	params["dependency_policy"] = dependencyPolicy
	return &patterns.FixSpec{Kind: spec.Kind, File: spec.File, Line: spec.Line, EndLine: spec.EndLine,
		Params: params, Definitions: spec.Definitions}
}

func parameterizationFile(path string, files map[string]*parameterizationSource) (*parameterizationSource, error) {
	if source := files[path]; source != nil {
		return source, nil
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	source := &parameterizationSource{fset: fset, file: file, src: src, digest: fmt.Sprintf("%x", sha256.Sum256(src)),
		dependencies: make(map[*pdg.Graph]patterns.RefactorDependencies)}
	files[path] = source
	return source, nil
}

func (s *parameterizationSource) dependencySafety(a, b *pdg.Graph) []patterns.RefactorDependencies {
	return []patterns.RefactorDependencies{s.dependenciesFor(a), s.dependenciesFor(b)}
}

func (s *parameterizationSource) dependenciesFor(graph *pdg.Graph) patterns.RefactorDependencies {
	if evidence, ok := s.dependencies[graph]; ok {
		return evidence
	}
	evidence := patterns.CheckRefactorDependencies(graph)
	s.dependencies[graph] = evidence
	return evidence
}

func completeDependencySafety(evidence []patterns.RefactorDependencies) bool {
	for _, function := range evidence {
		if function.Status != "complete" {
			return false
		}
	}
	return len(evidence) == 2
}

func unverifiedParameterization(c patterns.Candidate, reason string) patterns.Candidate {
	c.CounterEvidence = append(c.CounterEvidence, reason)
	return c
}

func compatibleInterfaces(a, b *pdg.Graph) bool {
	if a == nil || b == nil {
		return false
	}
	return compatibleParameters(a, a.Function.Inputs, b, b.Function.Inputs) && compatibleParameters(a, a.Function.Outputs, b, b.Function.Outputs)
}

func compatibleParameters(a *pdg.Graph, ap []pdg.Parameter, b *pdg.Graph, bp []pdg.Parameter) bool {
	if len(ap) != len(bp) {
		return false
	}
	for i, param := range ap {
		if !compatibleParameter(a, param, b, bp[i]) {
			return false
		}
	}
	return true
}

func compatibleParameter(a *pdg.Graph, ap pdg.Parameter, b *pdg.Graph, bp pdg.Parameter) bool {
	if !sameTypeFact(a, ap.Type, b, bp.Type) {
		return false
	}
	if ap.Role != bp.Role {
		return false
	}
	return ap.Variadic.Status == pdg.Known && bp.Variadic.Status == pdg.Known && ap.Variadic.Value == bp.Variadic.Value
}

func sameTypeFact(a *pdg.Graph, ap pdg.Fact, b *pdg.Graph, bp pdg.Fact) bool {
	return ap.Status == pdg.Known && bp.Status == pdg.Known && a.Tables.Text(ap.Value) == b.Tables.Text(bp.Value)
}

type parameterizationPair struct {
	a, b        *ast.FuncDecl
	left, right *pdg.Graph
}

func (s *parameterizationSource) prove(sites []patterns.Site, left, right *pdg.Graph) (*patterns.FixSpec, string) {
	pair, reason := s.sourcePair(sites, left, right)
	if pair == nil {
		return nil, reason
	}
	holes, ok := s.pairHoles(pair)
	if !ok {
		return nil, "full source bodies or placement do not match the supported extraction subset"
	}
	signatures, ok := verifiedHoleTypes(s.fset, holes, pair.left, pair.right)
	if !ok {
		return nil, "PDG callable types are unknown or incompatible"
	}
	return s.proofSpec(sites[0].Path, pair, signatures)
}

func (s *parameterizationSource) sourcePair(sites []patterns.Site, left, right *pdg.Graph) (*parameterizationPair, string) {
	if !currentGraphSource(left, s.digest) || !currentGraphSource(right, s.digest) {
		return nil, "PDG evidence does not match the current source snapshot"
	}
	a, b := namedFreeFunction(s.file, sites[0].Name), namedFreeFunction(s.file, sites[1].Name)
	if a == nil || b == nil {
		return nil, "source functions are unavailable or have receivers"
	}
	if a.Pos() > b.Pos() {
		return &parameterizationPair{b, a, right, left}, ""
	}
	return &parameterizationPair{a, b, left, right}, ""
}

func (s *parameterizationSource) pairHoles(pair *parameterizationPair) ([]hole, bool) {
	if !safeToExtract(s.fset, pair.a, pair.b) || !adjacent(s.fset, pair.a, pair.b, s.src) {
		return nil, false
	}
	_, holes, ok := diffPair(pair.a, pair.b)
	return holes, ok
}

func (s *parameterizationSource) proofSpec(path string, pair *parameterizationPair, signatures map[string]string) (*patterns.FixSpec, string) {
	encoded, err := json.Marshal(signatures)
	if err != nil {
		return nil, "callable types could not be encoded"
	}
	return &patterns.FixSpec{Kind: patterns.Parameterize, File: path,
		Line: s.fset.Position(pair.a.Pos()).Line, EndLine: s.fset.Position(pair.b.End()).Line,
		Params: map[string]string{"pair_a": pair.a.Name.Name, "pair_b": pair.b.Name.Name, "callee_types": string(encoded), "source_sha256": s.digest}}, ""
}

func currentGraphSource(graph *pdg.Graph, digest string) bool {
	if len(graph.Sources) != 1 {
		return false
	}
	identity := graph.Sources[0].ContentIdentity
	return identity.Status == pdg.Known && graph.Tables.Text(identity.Value) == digest
}

func namedFreeFunction(file *ast.File, name string) *ast.FuncDecl {
	name = name[strings.LastIndex(name, ".")+1:]
	for _, fn := range freeFuncs(file) {
		if fn.Name.Name == name {
			return fn
		}
	}
	return nil
}

type callableProof struct{ canonical, source string }

func verifiedHoleTypes(fset *token.FileSet, holes []hole, a, b *pdg.Graph) (map[string]string, bool) {
	signatures := make(map[string]string)
	for _, hole := range holes {
		ai, bi := hole.aExpr.(*ast.Ident), hole.bExpr.(*ast.Ident)
		left, right := graphCallable(a, fset.Position(ai.Pos()).Offset), graphCallable(b, fset.Position(bi.Pos()).Offset)
		if !compatibleCallableProofs(left, right) {
			return nil, false
		}
		signatures[ai.Name], signatures[bi.Name] = left.source, right.source
	}
	return signatures, true
}

func graphCallable(graph *pdg.Graph, offset int) callableProof {
	for _, node := range graph.Nodes {
		if !callAtOffset(graph, node, offset) {
			continue
		}
		attrs := graph.NodeAttributes(node)
		if attrs.CallForm != pdg.DirectCall || attrs.Callee.Status != pdg.Known {
			return callableProof{}
		}
		return callableExtensionValues(graph, attrs.Extensions)
	}
	return callableProof{}
}

func callableExtensionValues(graph *pdg.Graph, extensions []pdg.Extension) callableProof {
	var proof callableProof
	for _, extension := range extensions {
		switch graph.Tables.Text(extension.Name) {
		case "gstyle:callable_type":
			proof.canonical = callableExtensionString(graph, extension)
		case "gstyle:callable_source_type":
			proof.source = callableExtensionString(graph, extension)
		}
	}
	return proof
}

func callAtOffset(graph *pdg.Graph, node pdg.Node, offset int) bool {
	if node.Category != pdg.Call || node.Span == 0 || node.Attributes == 0 {
		return false
	}
	return int(graph.Spans[node.Span-1].Start) == offset
}

func callableExtensionString(graph *pdg.Graph, extension pdg.Extension) string {
	var text string
	if err := json.Unmarshal([]byte(graph.Tables.Text(extension.Value)), &text); err != nil {
		return ""
	}
	return text
}

func compatibleCallableProofs(a, b callableProof) bool {
	return a.canonical != "" && a.source != "" && a.canonical == b.canonical && a.source == b.source
}
