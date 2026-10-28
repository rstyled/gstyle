package gopatterns

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"

	"github.com/rstyled/gstyle/adapters/govalidation"
	"github.com/rstyled/gstyle/domain/ccgraph"
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

const NativeExtractionVersion = "0.2.0"

type NativeEvidence struct {
	Key, Source, ContentIdentity string
	PointerWidth                 int
	Projection                   ccgraph.Projection
}

type NativeMatching struct {
	root                    string
	Build                   NativeBuild
	ExtractionProfile       string
	ExtractionVersion       string
	PDGSchema, CFGSchema    string
	MatchingProfile         string
	ContractCommit          string
	AdmissionContractCommit string
	QualityStatus           string
	RegisteredProfiles      []NativePolicy
	SourceSnapshots         map[string]string
	Functions               []ccgraph.StageFunction
	Ineligible              []ccgraph.Diagnostic
	Evidence                []NativeEvidence
	Graphs                  []pdg.Graph `json:"-"`
}

func newNativeMatching(root string, build NativeBuild) *NativeMatching {
	return &NativeMatching{root: root, Build: build, ExtractionProfile: ScalarNormalProfileID, ExtractionVersion: NativeExtractionVersion, PDGSchema: pdg.SchemaVersion, CFGSchema: "0.2.0", MatchingProfile: ccgraph.Profile, ContractCommit: ccgraph.ContractCommit, AdmissionContractCommit: "56edd6bb804b10edaf9bbc2343d69e5e298d8c1a", RegisteredProfiles: []NativePolicy{nativePolicy()}, QualityStatus: "not_evaluated: bounded independently labeled development/held-out study remains required before replacing miner defaults"}
}

type nativeLoader struct {
	result  *NativeMatching
	pkg     *packages.Package
	root    string
	sources map[string]string
	pool    *pdg.Builder
}

func (n *nativeLoader) packageGraphs() error {
	for _, file := range n.pkg.Syntax {
		if err := n.file(file); err != nil {
			return err
		}
	}
	return nil
}

func (n *nativeLoader) file(file *ast.File) error {
	filename := n.pkg.Fset.PositionFor(file.Pos(), false).Filename
	rel, err := filepath.Rel(n.root, filename)
	if err != nil {
		return err
	}
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if err := n.function(fn, filepath.ToSlash(rel), n.sources[filename]); err != nil {
			return err
		}
	}
	return nil
}

func (n *nativeLoader) function(fn *ast.FuncDecl, path, digest string) error {
	graph, err := buildGraph(n.pkg, fn, path, n.pool)
	if err != nil {
		return err
	}
	key := graph.Tables.Text(graph.Function.ID)
	graph.Sources[0].ContentIdentity = pdgSourceIdentity(n.pool, digest)
	stage, projection, err := nativeStage(&graph, n.pkg, fn)
	if err != nil {
		n.result.Ineligible = append(n.result.Ineligible, ccgraph.Diagnostic{Function: key, Code: "ineligible-native-source", Message: err.Error()})
		return nil
	}
	if err := certifyNative(&graph, n.pool, n.result.Build, stage.PointerWidth); err != nil {
		return err
	}
	n.result.Functions = append(n.result.Functions, stage)
	n.result.Graphs = append(n.result.Graphs, graph)
	n.result.Evidence = append(n.result.Evidence, NativeEvidence{Key: key, Source: path, ContentIdentity: digest, PointerWidth: stage.PointerWidth, Projection: projection})
	return nil
}

// ExtractMatching is an explicit conformance path. Routine advisory mining is
// unchanged; native JSON producer claims are never accepted as validation proof.
func ExtractMatching(ctx context.Context, root string, paths []string) (*NativeMatching, error) {
	abs, err := absRoot(root)
	if err != nil {
		return nil, err
	}
	query, err := patternsQuery(abs, paths)
	if err != nil {
		return nil, err
	}
	build, err := nativeBuild(ctx, abs)
	if err != nil {
		return nil, err
	}
	return loadNativeMatching(ctx, abs, query, build)
}

func loadNativeMatching(ctx context.Context, root string, query []string, build NativeBuild) (*NativeMatching, error) {
	pkgs, sources, err := loadPatternSources(ctx, root, query)
	if err != nil {
		return nil, err
	}
	result := newNativeMatching(root, build)
	result.SourceSnapshots, err = nativeSourceSnapshots(root, sources)
	if err != nil {
		return nil, err
	}
	pool := pdg.NewBuilder()
	for _, pkg := range pkgs {
		loader := nativeLoader{result: result, pkg: pkg, root: root, sources: sources, pool: pool}
		if err := loader.packageGraphs(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func nativeSourceSnapshots(root string, sources map[string]string) (map[string]string, error) {
	result := map[string]string{}
	for path, digest := range sources {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil, err
		}
		result[filepath.ToSlash(rel)] = digest
	}
	return result, nil
}

func nativeStage(graph *pdg.Graph, pkg *packages.Package, fn *ast.FuncDecl) (ccgraph.StageFunction, ccgraph.Projection, error) {
	if graph.NormalFlow == nil {
		return ccgraph.StageFunction{}, ccgraph.Projection{}, fmt.Errorf("source contains constructs outside the scalar normal-flow profile")
	}
	if !scalarOrderKnown(fn.Body) || len(graph.Function.Outputs) > 1 {
		return ccgraph.StageFunction{}, ccgraph.Projection{}, fmt.Errorf("unspecified operand/result evaluation order; all-orders support is not implemented")
	}

	if err := validateNativeSource(graph, pkg, fn); err != nil {
		return ccgraph.StageFunction{}, ccgraph.Projection{}, err
	}
	return nativeInterfaceStage(graph, pkg, fn)
}

func validateNativeSource(graph *pdg.Graph, pkg *packages.Package, fn *ast.FuncDecl) error {
	if pkg.TypesSizes == nil || graph.Sources[0].ContentIdentity.Status != pdg.Known {
		return fmt.Errorf("target width or source identity is unknown")
	}
	if err := govalidation.Inventory(graph, fn, pkg.Fset, pkg.TypesInfo); err != nil {
		return err
	}
	if err := govalidation.Bindings(graph, fn, pkg.Fset, pkg.TypesInfo); err != nil {
		return err
	}
	return validateNativeFlow(graph, pkg, fn)
}

func validateNativeFlow(graph *pdg.Graph, pkg *packages.Package, fn *ast.FuncDecl) error {
	if err := govalidation.CFG(graph, fn, pkg.Fset); err != nil {
		return err
	}
	if err := govalidation.Ordering(graph); err != nil {
		return err
	}
	if err := govalidation.Reaching(graph); err != nil {
		return err
	}
	return govalidation.Values(graph, fn, pkg.Fset, pkg.TypesInfo)
}

func nativeInterfaceStage(graph *pdg.Graph, pkg *packages.Package, fn *ast.FuncDecl) (ccgraph.StageFunction, ccgraph.Projection, error) {
	inputs, outputs, err := govalidation.Interfaces(graph, fn, pkg.TypesInfo)
	if err != nil {
		return ccgraph.StageFunction{}, ccgraph.Projection{}, err
	}
	projection, err := govalidation.Project(graph)
	if err != nil {
		return ccgraph.StageFunction{}, ccgraph.Projection{}, err
	}
	references := projection.Features[8]
	stage := ccgraph.StageFunction{Key: graph.Tables.Text(graph.Function.ID), Name: fn.Name.Name, Graph: projection.Graph, References: &references, Inputs: &inputs, Outputs: &outputs, PointerWidth: int(pkg.TypesSizes.Sizeof(types.Typ[types.Uintptr]) * 8), UsesNativeWidth: nativeWidthOperations(fn, pkg.TypesInfo)}
	return stage, projection, nil
}

func nativeWidthOperations(fn *ast.FuncDecl, info *types.Info) bool {
	native := false
	ast.Inspect(fn, func(node ast.Node) bool {
		expression, ok := node.(ast.Expr)
		if !ok {
			return true
		}
		primitive := govalidation.Primitive(info.TypeOf(expression))
		if primitive == "isize" || primitive == "usize" {
			native = true
		}
		return true
	})
	return native
}

func certifyNative(graph *pdg.Graph, pool *pdg.Builder, build NativeBuild, width int) error {
	configuration, err := json.Marshal(struct {
		Target       NativeBuild `json:"target"`
		PointerWidth int         `json:"pointer_width"`
	}{build, width})
	if err != nil {
		return err
	}
	graph.Producer.Configuration = pool.Text(string(configuration))
	if graph.Producer.LanguageVersion.Status != pdg.Known {
		graph.Producer.LanguageVersion = pool.Known(build.GOVERSION)
	}
	return certifyScalarExtraction(graph, pool)
}

func certifyScalarExtraction(graph *pdg.Graph, pool *pdg.Builder) error {
	graph.Profile = pool.Policy(ScalarNormalProfileID, NativeExtractionVersion)
	graph.Conformance = pdg.Conformance{Status: pdg.Conformant}
	for i := range graph.Nodes {
		graph.Nodes[i].Evidence = pool.Evidence(pdg.Supported, "Source inventory and value/binding sites independently checked under the scalar normal-flow profile.")
	}
	for i, capability := range graph.Capabilities {
		if pool.Tables.Text(capability.Name) == "node_extraction" {
			graph.Capabilities[i].Evidence = pool.Evidence(pdg.Supported, "Operation inventory independently checked against typed source, including unreachable sites.")
		}
	}
	graph.Diagnostics = nil
	return pdg.Validate(graph)
}
