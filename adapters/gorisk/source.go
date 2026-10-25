package gorisk

import (
	"context"
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/changerisk"
	"github.com/rstyled/gstyle/domain/pdg"
)

func Extract(ctx context.Context, root string, paths []string) ([]changerisk.Function, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	ext, err := gopatterns.Extract(ctx, root, paths)
	if err != nil {
		return nil, err
	}
	if err := gopatterns.CheckFixSources(ext.Sources); err != nil {
		return nil, err
	}
	return projectExtraction(ext, paths)
}

func projectExtraction(ext *gopatterns.Extraction, paths []string) ([]changerisk.Function, error) {
	var result []changerisk.Function
	identities := map[string]int{}
	for _, function := range ext.Funcs {
		if !slices.Contains(paths, filepath.ToSlash(function.Path)) {
			continue
		}
		item, err := projectFunction(function, ext.Callables, functionIdentity(function, identities))
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func functionIdentity(function gopatterns.FuncPdg, counts map[string]int) string {
	if !strings.HasSuffix(function.Name, ".init") && !strings.HasSuffix(function.Name, "._") {
		return function.Name
	}
	key := function.Path + ":" + function.Name
	counts[key]++
	return fmt.Sprintf("%s:%d", key, counts[key])
}

func projectFunction(source gopatterns.FuncPdg, callables []*pdg.Graph, identity string) (changerisk.Function, error) {
	graph, err := changerisk.Project(&source.Pdg)
	if err != nil {
		return changerisk.Function{}, err
	}
	body, err := functionTokens(&source.Pdg)
	if err != nil {
		return changerisk.Function{}, err
	}
	combined, err := projectWithClosures(&source.Pdg, graph, callables)
	if err != nil {
		return changerisk.Function{}, err
	}
	return changerisk.Function{ID: identity, Name: source.Name, Path: source.Path, Line: source.Line, Body: body, Graph: combined}, nil
}

func enclosed(parent, child *pdg.Graph) bool {
	a, b := parent.Function.Span, child.Function.Span
	if graphPath(parent) != graphPath(child) || a.Start == b.Start {
		return false
	}
	return b.Start >= a.Start && b.End <= a.End
}

func graphPath(graph *pdg.Graph) string {
	return graph.Tables.Text(graph.Sources[graph.Function.Span.Source-1].Path)
}

func projectWithClosures(source *pdg.Graph, graph changerisk.Graph, callables []*pdg.Graph) (changerisk.Graph, error) {
	shapes := []string{graph.Shape}
	nodes := slices.Clone(graph.Nodes)
	for _, child := range callables {
		if !enclosed(source, child) {
			continue
		}
		projected, err := changerisk.Project(child)
		if err != nil {
			return changerisk.Graph{}, err
		}
		shapes = append(shapes, projected.Shape)
		nodes = append(nodes, projected.Nodes...)
	}
	if len(shapes) == 1 {
		return graph, nil
	}
	slices.Sort(shapes)
	return changerisk.Graph{Shape: strings.Join(shapes, ":"), Nodes: nodes, Edges: graph.Edges, Ambiguous: true}, nil
}

func functionTokens(graph *pdg.Graph) (string, error) {
	span := graph.Function.Span
	_, content, ok := graph.Tables.Sources.Source(graph.Sources[span.Source-1].Snapshot)
	if !ok || uint64(span.End) > uint64(len(content)) || span.Start > span.End {
		return "", fmt.Errorf("PDG function source is unavailable")
	}
	return tokens([]byte(content[span.Start:span.End]))
}

func tokens(source []byte) (string, error) {
	file := token.NewFileSet().AddFile("function.go", -1, len(source))
	var scan scanner.Scanner
	var scanErr error
	scan.Init(file, source, func(pos token.Position, message string) { scanErr = fmt.Errorf("%s: %s", pos, message) }, 0)
	var items []string
	for {
		_, kind, literal := scan.Scan()
		if kind == token.EOF {
			return strings.Join(items, "\x00"), scanErr
		}
		if kind == token.SEMICOLON {
			literal = ";"
		}
		items = append(items, kind.String()+":"+literal)
	}
}

func ExistingPaths(root string, changed []string) ([]string, string, error) {
	var paths []string
	excluded := 0
	for _, name := range changed {
		class, err := classifyPath(root, name)
		if err != nil {
			return nil, "", err
		}
		switch class {
		case includedPath:
			paths = append(paths, name)
		case excludedPath:
			excluded++
		}
	}
	message := ""
	if excluded > 0 {
		message = fmt.Sprintf("%d changed test or nested-module Go files are outside the PDG comparison", excluded)
	}
	return paths, message, nil
}

type pathClass int

const (
	missingPath pathClass = iota
	includedPath
	excludedPath
)

func classifyPath(root, name string) (pathClass, error) {
	if !strings.HasSuffix(name, ".go") {
		return missingPath, nil
	}
	_, err := os.Stat(filepath.Join(root, name))
	if os.IsNotExist(err) {
		return missingPath, nil
	}
	if err != nil {
		return missingPath, err
	}
	if strings.HasSuffix(name, "_test.go") || nestedModule(root, name) {
		return excludedPath, nil
	}
	return includedPath, nil
}

func nestedModule(root, name string) bool {
	for dir := filepath.Dir(name); dir != "."; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(root, dir, "go.mod")); err == nil {
			return true
		}
	}
	return false
}
