package gopatterns

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/callflow"
	"go/ast"
	"path/filepath"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
	"golang.org/x/tools/go/packages"
)

// functionGraph owns one lowering of a compiler declaration. Summaries and
// reports borrow its graph; neither consumer builds a replacement graph.
type functionGraph struct {
	closure     bool
	inventory   *irBuilder
	declaration *ast.FuncDecl
	path        string
	graph       *pdg.Graph
}
type packageAnalysis struct {
	closurePackageView *packages.Package
	program            *callflow.Program
	exits              callflow.ExitSummaries
	pkg                *packages.Package
	root               string
	pool               *pdg.Builder
	sources            map[string]string
	graphs             []functionGraph
	context            extractCtx
	result             []FuncPdg
}

func (p *packageAnalysis) build() error {
	for _, file := range p.pkg.Syntax {
		graphs, err := p.fileGraphs(file)
		if err != nil {
			return err
		}
		p.graphs = append(p.graphs, graphs...)
	}
	return p.finalizeGraphs()
}
func (p *packageAnalysis) fileGraphs(file *ast.File) ([]functionGraph, error) {
	filename := p.pkg.Fset.PositionFor(file.Pos(), false).Filename
	rel, err := filepath.Rel(p.root, filename)
	if err != nil {
		return nil, fmt.Errorf("PDG source path: %w", err)
	}
	var graphs []functionGraph
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		sources, err := p.declarationGraphs(fn, filepath.ToSlash(rel), filename)
		if err != nil {
			return nil, err
		}
		graphs = append(graphs, sources...)
	}
	return graphs, nil
}
func (p *packageAnalysis) declarationGraphs(fn *ast.FuncDecl, path, filename string) ([]functionGraph, error) {
	source, err := p.typedFunctionGraph(p.pkg, fn, path, filename)
	if err != nil {
		return nil, err
	}
	closures, err := p.closureGraphs(source, filename)
	if err != nil {
		return nil, err
	}
	return append([]functionGraph{source}, closures...), nil
}

func (p *packageAnalysis) typedFunctionGraph(pkg *packages.Package, fn *ast.FuncDecl, path, filename string) (functionGraph, error) {
	inventory, err := prepareGraph(pkg, fn, path, p.pool)
	if err != nil {
		return functionGraph{}, err
	}
	graph, err := inventory.summaryInventory().finishGraph(fn)
	if err != nil {
		return functionGraph{}, err
	}
	graph.Sources[0].ContentIdentity = pdgSourceIdentity(p.pool, p.sources[filename])
	graph.Sources[0].Snapshot = sourceSnapshotReference(p.pool.Tables.Sources, filename)
	return functionGraph{inventory: inventory, declaration: fn, path: path, graph: &graph}, nil
}

func (p *packageAnalysis) summarize() error {
	functions, err := resourceFunctions(p.pkg, p.graphs)
	if err != nil {
		return err
	}
	facts, err := summarizeResourcesWithExits(functions, p.exits)
	if err != nil {
		return err
	}
	p.context = newExtractCtx(p.pkg, p.root, facts)
	p.context.resourceQueries = make(map[*pdg.Graph]*resourceflow.Query, len(functions))
	for _, function := range functions {
		p.context.resourceQueries[function.graph] = function.query
	}
	return nil
}
func (p *packageAnalysis) analyze() error {
	for _, source := range p.graphs {
		if source.closure {
			continue
		}
		result, err := extractFunc(p.pkg, source, p.context)
		if err != nil {
			return err
		}
		p.result = append(p.result, result)
	}
	return nil
}

// Package summaries use provisional dependency solves over one source inventory.
// Consumers receive only the final canonical graphs.
func (p *packageAnalysis) finalizeGraphs() error {
	graphs := map[string]*pdg.Graph{}
	for _, source := range p.graphs {
		graphs[source.graph.Tables.Text(source.graph.Function.ID)] = source.graph
	}
	if p.program == nil {
		p.program = callflow.NewProgram()
	}
	exits, err := p.program.ExitEffects(graphs)
	if err != nil {
		return err
	}
	p.exits = exits
	return p.publishGraphs(exits)
}

func (p *packageAnalysis) publishGraphs(exits callflow.ExitSummaries) error {
	for i := range p.graphs {
		if err := p.graphs[i].finalize(exits); err != nil {
			return err
		}
		if err := p.program.Add(p.graphs[i].graph); err != nil {
			return err
		}
	}
	p.closurePackageView = nil
	return nil
}

func (source *functionGraph) finalize(exits callflow.ExitSummaries) error {
	if pdg.ExitBodyFlow(source.graph) == nil {
		source.inventory = nil
		return nil
	}
	queries, err := callflow.NewQueries(source.graph)
	if err != nil {
		return err
	}
	b := source.inventory
	b.callExits = map[pdg.Ref]callflow.Outcomes{}
	observations := callflow.OrdinaryObservations(queries, exits)
	b.conditionValues = observations.Conditions
	for node, ref := range b.nodes {
		b.recordCallExit(node, ref, observations.Calls[ref])
	}
	if len(b.callExits)+len(b.conditionValues) == 0 {
		source.inventory = nil
		return nil
	}
	graph, err := b.finishGraph(source.declaration)
	if err != nil {
		return err
	}
	graph.Sources = source.graph.Sources
	graph.ValueFlow.ExitContext = pdg.ExitBodyFlow(source.graph)
	*source.graph = graph
	source.inventory = nil
	return nil
}

func (b *irBuilder) recordCallExit(node ast.Node, ref pdg.Ref, outcome callflow.Outcomes) {
	call, ok := node.(*ast.CallExpr)
	if !ok || b.deferredInvocation(call) || !b.mayReturn(call) {
		return
	}
	if outcome == 0 || outcome&(callflow.Returns|callflow.UnknownExit) != 0 {
		return
	}
	b.callExits[ref] = outcome
}
