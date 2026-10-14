package gopatterns

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"maps"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

type parameterizationSource struct {
	graphValidity map[*pdg.Graph]bool
	snapshot      pdg.SourceSnapshot
	fset          *token.FileSet
	file          *ast.File
	src           []byte
	digest        string
	packageNames  []string
}

func (source *parameterizationSource) verifyCandidate(c patterns.Candidate, other *parameterizationSource, left, right *pdg.Graph) patterns.Candidate {
	spec, reason := source.prove(c.Sites, other, left, right)
	if spec == nil {
		return unverifiedParameterization(c, reason)
	}
	return patterns.Candidate{
		Kind: c.Kind, ScoreMilli: c.ScoreMilli, Breakdown: c.Breakdown, Observation: c.Observation,
		Inference:        "complete source bodies match under substitutions with compatible PDG callable types",
		PossibleRefactor: "extract the verified pair into a shared helper with the differing callables as parameters",
		Sites:            c.Sites, Definitions: c.Definitions, CounterEvidence: c.CounterEvidence, FixSpec: spec,
	}
}

func (s *parameterizationSource) prove(sites []patterns.Site, other *parameterizationSource, left, right *pdg.Graph) (*patterns.FixSpec, string) {
	pair, reason := s.sourcePair(sites, other, left, right)
	if pair == nil {
		return nil, reason
	}
	if reason := parameterizationImportRisk(pair); reason != "" {
		return nil, reason
	}
	normalizeParameterizationImports(pair)
	holes, ok := s.pairHoles(pair)
	if !ok {
		return nil, "full source bodies do not match the supported extraction subset"
	}
	signatures, ok := verifiedHoleTypes(s.fset, holes, pair.left, pair.right)
	if !ok {
		return nil, "PDG callable types are unknown or incompatible"
	}
	if reason := s.rewriteRisk(pair); reason != "" {
		return nil, reason
	}
	return pair.first.proofSpec(pair, signatures)
}

func (s *parameterizationSource) sourcePair(sites []patterns.Site, other *parameterizationSource, left, right *pdg.Graph) (*parameterizationPair, string) {
	if !s.currentGraph(left) || !other.currentGraph(right) {
		return nil, "PDG evidence does not match the current source snapshot"
	}
	a, b := namedFreeFunction(s.file, sites[0].Name), namedFreeFunction(other.file, sites[1].Name)
	if a == nil || b == nil {
		return nil, "source functions are unavailable or have receivers"
	}
	return orderedParameterizationPair(&parameterizationPair{a: a, b: b, left: left, right: right, first: s, second: other}), ""
}

func (s *parameterizationSource) pairHoles(pair *parameterizationPair) ([]hole, bool) {
	if !safeExtractionBodies(pair.a, pair.b) {
		return nil, false
	}
	if !matchingCallSpreads(pair.a, pair.b) {
		return nil, false
	}
	_, holes, ok := diffPair(pair.a, pair.b)
	return holes, ok && holesAreCallTargets(pair.a, pair.b, holes)
}

func (s *parameterizationSource) proofSpec(pair *parameterizationPair, signatures map[string]string) (*patterns.FixSpec, string) {
	encoded, err := json.Marshal(signatures)
	if err != nil {
		return nil, "callable types could not be encoded"
	}
	spec := &patterns.FixSpec{Kind: patterns.Parameterize, File: s.snapshot.Path,
		Line: s.fset.Position(pair.a.Pos()).Line, EndLine: s.fset.Position(pair.b.End()).Line,
		Params: map[string]string{"pair_a": pair.a.Name.Name, "pair_b": pair.b.Name.Name, "callee_types": string(encoded), "source_sha256": s.digest, "substitution_policy": substitutionPolicy, "helper_name": freshHelperName(s, pair)}}
	if pair.first != pair.second {
		spec.EndLine = s.fset.Position(pair.a.End()).Line
		spec.Params["secondary_file"], spec.Params["secondary_sha256"] = pair.second.snapshot.Path, pair.second.digest
	}
	return spec, ""
}

func (s *parameterizationSource) currentGraph(graph *pdg.Graph) bool {
	if valid, ok := s.graphValidity[graph]; ok {
		return valid
	}
	if s.graphValidity == nil {
		s.graphValidity = map[*pdg.Graph]bool{}
	}
	valid := currentGraphSource(graph, s.digest)
	s.graphValidity[graph] = valid
	return valid
}

const substitutionPolicy = "gstyle.parameterization/0.3.0"

type parameterizationPair struct {
	first, second *parameterizationSource
	a, b          *ast.FuncDecl
	left, right   *pdg.Graph
}

type callableProof struct{ canonical, source string }

// VerifyParameterizations uses canonical IR facts, not a second compiler load.
// AST matching checks the complete source; it does not establish type safety.
func VerifyParameterizations(report *patterns.PatternsReport, extraction *Extraction) error {
	if err := CheckFixSources(extraction.Sources); err != nil {
		rejectStaleParameterizations(report)
		return nil
	}
	graphs := make(map[string]*pdg.Graph)
	for i := range extraction.Funcs {
		fp := &extraction.Funcs[i]
		graphs[fp.Name] = &fp.Pdg
	}
	verifier := parameterizationVerifier{graphs: graphs, files: make(map[string]*parameterizationSource), names: maps.Clone(extraction.PackageNames), fset: token.NewFileSet(), claimed: make(map[string]bool), attempts: make(map[[2]string]parameterizationAttempt)}
	var candidates []patterns.Candidate
	for _, candidate := range report.Candidates {
		verified, err := verifier.candidates(candidate)
		if err != nil {
			return err
		}
		candidates = append(candidates, verified...)
	}
	report.Candidates = candidates
	return nil
}

func verifyParameterization(c patterns.Candidate, graphs map[string]*pdg.Graph, files map[string]*parameterizationSource, names map[string][]string, fset *token.FileSet) (patterns.Candidate, error) {
	c.FixSpec = nil
	if len(c.Sites) != 2 {
		return unverifiedParameterization(c, "automatic extraction requires one explicit function pair"), nil
	}
	if functionPackage(c.Sites[0].ID) != functionPackage(c.Sites[1].ID) {
		return unverifiedParameterization(c, "helper extraction requires functions in the same package"), nil
	}
	left, right := graphs[c.Sites[0].ID], graphs[c.Sites[1].ID]
	if !compatibleInterfaces(left, right) {
		return unverifiedParameterization(c, "PDG function interfaces have unknown or incompatible types"), nil
	}
	source, err := parameterizationFile(c.Sites[0].Path, files, names[functionPackage(c.Sites[0].ID)], left.Tables.Sources, fset)
	if err != nil {
		return c, err
	}
	other, err := parameterizationFile(c.Sites[1].Path, files, names[functionPackage(c.Sites[1].ID)], right.Tables.Sources, fset)
	if err != nil {
		return c, err
	}
	return source.verifyCandidate(c, other, left, right), nil
}

func parameterizationFile(path string, files map[string]*parameterizationSource, names []string, sources *pdg.SourceContext, fset *token.FileSet) (*parameterizationSource, error) {
	if source := files[path]; source != nil {
		source.packageNames = names
		return source, nil
	}
	src, err := revisionSource(path, sources)
	if err != nil {
		return nil, err
	}
	absolute := fixSourcePath(sources.Root, path)
	file, err := sourceSyntaxTreeIn(absolute, sources, fset)
	if err != nil {
		return nil, err
	}
	snapshot, _ := sources.Snapshot(sources.Reference(absolute))
	source := &parameterizationSource{snapshot: snapshot, fset: fset, file: file, src: src, digest: fmt.Sprintf("%x", sha256.Sum256(src)),
		packageNames: names}
	files[path] = source
	return source, nil
}

func unverifiedParameterization(c patterns.Candidate, reason string) patterns.Candidate {
	c.CounterEvidence = append(c.CounterEvidence, reason)
	return c
}

func compatibleInterfaces(a, b *pdg.Graph) bool {
	if a == nil || b == nil {
		return false
	}
	if a.Tables == nil || b.Tables == nil {
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

func currentGraphSource(graph *pdg.Graph, digest string) bool {
	// Validate before source/call lookups dereference graph-local records.
	if pdg.Validate(graph) != nil {
		return false
	}
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

func verifiedHoleTypes(fset *token.FileSet, holes []hole, a, b *pdg.Graph) (map[string]string, bool) {
	signatures := make(map[string]string)
	for _, hole := range holes {
		ai, bi := hole.aExpr.(*ast.Ident), hole.bExpr.(*ast.Ident)
		left, right := graphCallable(a, fset.Position(ai.Pos()).Offset), graphCallable(b, fset.Position(bi.Pos()).Offset)
		if !compatibleCallableProofs(left, right) {
			return nil, false
		}
		signatures[ai.Name], signatures[bi.Name] = left.source, left.source
	}
	return signatures, true
}

func graphCallable(graph *pdg.Graph, offset int) callableProof {
	for _, node := range graph.Nodes {
		if !callAtOffset(graph, node, offset) {
			continue
		}
		attrs := graph.NodeAttributes(node)
		if attrs.CallForm != pdg.DirectCall || attrs.Callee.Status != pdg.Known || !strings.Contains(graph.Tables.Text(attrs.Callee.Value), ".") {
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
	return a.canonical != "" && a.source != "" && a.canonical == b.canonical
}

func functionPackage(id string) string {
	if dot := strings.LastIndex(id, "."); dot >= 0 {
		return id[:dot]
	}
	return ""
}

func rejectStaleParameterizations(report *patterns.PatternsReport) {
	for i, candidate := range report.Candidates {
		if candidate.Kind != patterns.Parameterize {
			continue
		}
		candidate.FixSpec = nil
		report.Candidates[i] = unverifiedParameterization(candidate, "PDG evidence does not match the current source snapshot")
	}
}

func orderedParameterizationPair(pair *parameterizationPair) *parameterizationPair {
	if pair.first == pair.second && pair.a.Pos() > pair.b.Pos() {
		pair.a, pair.b = pair.b, pair.a
		pair.left, pair.right = pair.right, pair.left
	}
	return pair
}
