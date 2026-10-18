package gopatterns

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/callflow"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/cfg"
	"golang.org/x/tools/go/packages"
)

type irBuilder struct {
	conditionValues                      map[pdg.Ref]pdg.BooleanFact
	callExits                            map[pdg.Ref]callflow.Outcomes
	typeQualifier                        types.Qualifier
	sourceFilename                       string
	attributes                           []pdg.Attributes
	outputs                              []pdg.Ref
	pool                                 *pdg.Builder
	graph                                pdg.Graph
	info                                 *types.Info
	sizes                                types.Sizes
	fset                                 *token.FileSet
	symbols                              map[types.Object]pdg.Ref
	nodes                                map[ast.Node]pdg.Ref
	logicalPredicates                    map[*ast.BinaryExpr]pdg.Ref
	deferredCalls                        map[*ast.CallExpr]pdg.Ref
	rangeBounds                          map[pdg.Ref]pdg.Ref
	writeTargets                         map[ast.Node]bool
	stack                                []ast.Node
	policy, sourceEvidence, sourceOrigin pdg.Ref
	executionEvidence, executionOrigin   pdg.Ref
	bindingOperations                    bool
	runtimeOperations                    bool
	scalarDefinitions                    bool
	scalarOperations                     bool
	extendedScalar                       bool
	generalBindings                      bool
	closureValues                        bool
	weakDefinitions                      map[pdg.Ref]bool
	controlFlow                          *cfg.CFG
	controlOutcomes                      map[int][]pdg.Position
	implicitReads                        map[ast.Node][]pdg.Ref
	semanticControl                      bool
	normalDependencies                   bool
	scalarOrderKnown                     bool
}

func newIRBuilder(pkg *packages.Package, fn *ast.FuncDecl, path string, pool *pdg.Builder) *irBuilder {
	b := &irBuilder{
		pool: pool, info: pkg.TypesInfo, sizes: pkg.TypesSizes, fset: pkg.Fset, typeQualifier: callableQualifier(pkg, fn),
		sourceFilename: pkg.Fset.PositionFor(fn.Pos(), false).Filename,
		symbols:        map[types.Object]pdg.Ref{}, nodes: map[ast.Node]pdg.Ref{},
		writeTargets: map[ast.Node]bool{}, logicalPredicates: map[*ast.BinaryExpr]pdg.Ref{}, deferredCalls: map[*ast.CallExpr]pdg.Ref{},
	}
	b.policy = pool.Policy("gstyle.abstraction", "0.4.0")
	b.sourceEvidence = pool.Evidence(pdg.Approximated, "Operation identified in typed source; dependency completeness is separate.")
	b.sourceOrigin = pool.Origin(pdg.SourceOrigin, "Go source operation; abstraction lowering is approximate.")
	b.header(pkg, fn, path)
	b.buildEvidence(pkg)
	return b
}

func (b *irBuilder) header(pkg *packages.Package, fn *ast.FuncDecl, path string) {
	p := b.pool
	qualified := b.functionName(fn)
	identity := p.Text(pkg.PkgPath + "." + fn.Name.Name)
	if qualified.Status == pdg.Known {
		identity = qualified.Value
	}
	b.graph = pdg.Graph{
		Tables: p.Tables, Profile: b.policy,
		Producer: pdg.Producer{
			Name: p.Text("gstyle"), Version: p.Text("pdg-ir-0.4.0"), Language: p.Text("go"),
			LanguageVersion: p.Missing(pdg.Unknown, "Compiler language version is not recorded."), Configuration: p.Text("{}"),
		},
		Sources: []pdg.Source{{ID: p.Text(path), Path: p.Text(path), ContentIdentity: p.Missing(pdg.Unknown, "Source content digest is not computed.")}},
		Function: pdg.Function{
			ID: identity, Name: p.Known(fn.Name.Name), QualifiedName: qualified, Span: b.span(fn),
			Interface:      p.Evidence(pdg.Supported, "Complete typed function interface."),
			ReferenceCount: pdg.CountFact{Status: pdg.Unknown, Reason: p.Text("Reference-count policy is not selected.")},
		},
	}
}

func (b *irBuilder) functionName(fn *ast.FuncDecl) pdg.Fact {
	object, ok := b.info.Defs[fn.Name].(*types.Func)
	if !ok {
		return b.pool.Missing(pdg.Unknown, "Function identity is unresolved.")
	}
	name := patterns.FuncID(object)
	return b.pool.Known(name)
}

func (b *irBuilder) span(n ast.Node) pdg.Span {
	if n == nil {
		return b.graph.Function.Span
	}
	start := b.fset.PositionFor(n.Pos(), false).Offset
	end := b.fset.PositionFor(n.End(), false).Offset
	return pdg.Span{Source: 1, Start: uint32(start), End: uint32(end)}
}

func (b *irBuilder) addNode(n ast.Node, category pdg.Category, attrs pdg.Attributes) pdg.Ref {
	if ref := b.nodes[n]; ref != 0 {
		node := &b.graph.Nodes[ref-1]
		node.Category = category
		b.attributes[node.Attributes-1] = attrs
		return ref
	}
	return b.appendNode(n, category, attrs)
}

// appendNode publishes the node, span, attributes and AST lookup together.
// Their references must stay aligned during construction.
func (b *irBuilder) appendNode(n ast.Node, category pdg.Category, attrs pdg.Attributes) pdg.Ref {
	id := pdg.Ref(len(b.graph.Nodes) + 1)
	b.graph.Spans = append(b.graph.Spans, b.span(n))
	b.attributes = append(b.attributes, attrs)
	b.graph.Nodes = append(b.graph.Nodes, pdg.Node{
		ID: b.pool.Text("n" + strconv.Itoa(int(id))), Category: category, Line: b.sourceLine(n),
		OriginalKind: b.pool.Known(fmt.Sprintf("%T", n)), Span: pdg.Ref(len(b.graph.Spans)),
		Attributes: pdg.Ref(len(b.attributes)), Evidence: b.sourceEvidence, Provenance: b.sourceOrigin,
	})
	if n != nil {
		b.nodes[n] = id
	}
	return id
}

func (b *irBuilder) symbol(id *ast.Ident, role pdg.Role) pdg.Ref {
	obj := b.info.ObjectOf(id)
	if obj == nil || id.Name == "_" {
		return 0
	}
	if ref, ok := b.symbols[obj]; ok {
		return ref
	}
	if obj.Pkg() != nil && obj.Parent() == obj.Pkg().Scope() {
		role = pdg.GlobalRole
	}
	ref := pdg.Ref(len(b.graph.Symbols) + 1)
	b.graph.Symbols = append(b.graph.Symbols, pdg.Symbol{
		ID: b.pool.Text("s" + strconv.Itoa(int(ref))), Scope: b.scope(obj),
		Name: b.pool.Known(obj.Name()), Type: b.pool.Known(irTypeString(obj.Type())), Role: role, Extensions: b.resolvedType(obj.Type()),
	})
	b.symbols[obj] = ref
	return ref
}

// definition updates both the definition table and its owner backreference.
func (b *irBuilder) definition(node, symbol pdg.Ref) {
	if symbol == 0 {
		return
	}
	b.storageDefinition(node, symbol, 0)
}

func (b *irBuilder) storageDefinition(node, symbol, location pdg.Ref) pdg.Ref {
	id := pdg.Ref(len(b.graph.Definitions) + 1)
	b.graph.Definitions = append(b.graph.Definitions, pdg.Definition{ID: b.pool.Text("d" + strconv.Itoa(int(id))), Node: node, Symbol: symbol, Location: location})
	attrs := &b.attributes[b.graph.Nodes[node-1].Attributes-1]
	attrs.Defines = append(attrs.Defines, id)
	return id
}

func (b *irBuilder) sourceLine(node ast.Node) uint32 {
	if node == nil {
		return 0
	}
	return uint32(b.fset.PositionFor(node.Pos(), false).Line)
}

func (b *irBuilder) selectScalarLowering(fn *ast.FuncDecl) {
	b.scalarOperations = b.selectScalarSource(fn)
	b.bindingOperations = !b.scalarOperations && b.interfaceBindingSubset(fn)
	b.generalBindings = !b.scalarOperations && !b.bindingOperations && b.exitOperationSubset(fn.Body)
	b.runtimeOperations = b.generalBindings || runtimeRelevant(fn, b.info)
	b.scalarOrderKnown = scalarOrderKnown(fn.Body) && len(b.graph.Function.Outputs) <= 1
}

func (b *irBuilder) selectScalarSource(fn *ast.FuncDecl) bool {
	if !b.scalarInterface(fn) {
		return false
	}
	if b.reachingSubset(fn.Body) {
		return true
	}
	b.extendedScalar = b.extendedReachingSubset(fn.Body)
	return b.extendedScalar
}

// buildGraph is the sole production graph construction path.
func buildGraph(pkg *packages.Package, fn *ast.FuncDecl, path string, pool *pdg.Builder) (pdg.Graph, error) {
	b, err := prepareGraph(pkg, fn, path, pool)
	if err != nil {
		return pdg.Graph{}, err
	}
	return b.finishGraph(fn)
}

func prepareGraph(pkg *packages.Package, fn *ast.FuncDecl, path string, pool *pdg.Builder) (*irBuilder, error) {
	file := pkg.Fset.File(fn.Pos())
	if file == nil || uint64(file.Size()) > uint64(^uint32(0)) {
		return nil, fmt.Errorf("PDG source exceeds 32-bit offset range")
	}
	b := &builder{info: pkg.TypesInfo, fset: pkg.Fset, binds: map[types.Object]int{}, ir: newIRBuilder(pkg, fn, path, pool)}
	b.ir.interfaces(fn)
	b.params(fn)
	b.stmt(fn.Body)
	b.ir.selectScalarLowering(fn)
	b.ir.implicitReads = map[ast.Node][]pdg.Ref{}
	b.ir.enrich(fn.Body)
	b.ir.closureSourceFacts(fn)
	b.ir.functionValueFacts(fn.Body)
	b.ir.callTargetFacts(fn.Body)
	b.ir.bindingSourceFacts(fn)
	b.ir.interfaceValueFacts()
	b.ir.runtimeValueFacts(fn)
	b.ir.memorySourceFacts(fn)
	b.ir.representationFacts(fn)
	b.ir.deferredSourceFacts(fn.Body)
	b.ir.operationEffectFacts(fn.Body)
	b.ir.evaluationRegionFacts(fn.Body)
	b.ir.returnValues(fn.Body)
	b.ir.references()
	return b.ir, nil
}

func (b *irBuilder) finishGraph(fn *ast.FuncDecl) (pdg.Graph, error) {
	if err := b.dependencies(fn.Body); err != nil {
		return pdg.Graph{}, err
	}

	b.canonicalValueFacts(fn)
	b.capabilities()
	if err := b.pack(); err != nil {
		return pdg.Graph{}, err
	}
	return b.graph, nil
}
