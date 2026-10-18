package gopatterns

import (
	"fmt"
	"go/ast"
	"path/filepath"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
	"golang.org/x/tools/go/packages"
)

// functionGraph owns one lowering of a compiler declaration. Summaries and
// reports borrow its graph; neither consumer builds a replacement graph.
type functionGraph struct {
	declaration *ast.FuncDecl
	path        string
	graph       *pdg.Graph
}
type packageAnalysis struct {
	pkg     *packages.Package
	root    string
	pool    *pdg.Builder
	sources map[string]string
	graphs  []functionGraph
	context extractCtx
	result  []FuncPdg
}

func (p *packageAnalysis) build() error {
	for _, file := range p.pkg.Syntax {
		graphs, err := p.fileGraphs(file)
		if err != nil {
			return err
		}
		p.graphs = append(p.graphs, graphs...)
	}
	return nil
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
		graph, err := buildGraph(p.pkg, fn, filepath.ToSlash(rel), p.pool)
		if err != nil {
			return nil, err
		}
		graph.Sources[0].ContentIdentity = pdgSourceIdentity(p.pool, p.sources[filename])
		graph.Sources[0].Snapshot = sourceSnapshotReference(p.pool.Tables.Sources, filename)
		graphs = append(graphs, functionGraph{declaration: fn, path: filepath.ToSlash(rel), graph: &graph})
	}
	return graphs, nil
}
func (p *packageAnalysis) summarize() error {
	functions, err := resourceFunctions(p.pkg, p.graphs)
	if err != nil {
		return err
	}
	facts, err := summarizeResources(functions)
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
		result, err := extractFunc(p.pkg, source, p.context)
		if err != nil {
			return err
		}
		p.result = append(p.result, result)
	}
	return nil
}
