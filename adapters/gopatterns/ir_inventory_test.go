package gopatterns

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/rstyled/gstyle/adapters/govalidation"
	"github.com/rstyled/gstyle/domain/pdg"
)

func sourceInventory(t *testing.T, graph *pdg.Graph, source string) error {
	fn, fset, info := independentSource(t, source)
	return govalidation.Inventory(graph, fn, fset, info)
}

func independentSource(t *testing.T, source string) (*ast.FuncDecl, *token.FileSet, *types.Info) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	if _, err := (&types.Config{}).Check("example", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	for _, declaration := range file.Decls {
		if fn, ok := declaration.(*ast.FuncDecl); ok {
			return fn, fset, info
		}
	}
	t.Fatal("no function")
	return nil, nil, nil
}

func TestIndependentValidatorsRejectMissingDependencies(t *testing.T) {
	source := `package example; func f(x int) int { return x }`
	fn, fset, info := independentSource(t, source)
	for _, kind := range []string{"reaching-definition", "scalar-value"} {
		graph := parseIR(t, source, pdg.NewBuilder())
		for i, edge := range graph.Edges {
			if graph.Tables.Text(edge.Subkind) == kind {
				graph.Edges = append(graph.Edges[:i], graph.Edges[i+1:]...)
				break
			}
		}
		if err := pdg.Validate(&graph); err != nil {
			t.Fatal("mutation must remain structurally valid", err)
		}
		if kind == "reaching-definition" && govalidation.Reaching(&graph) == nil {
			t.Fatal("missing definition edge is accepted")
		}
		if kind == "scalar-value" && govalidation.Values(&graph, fn, fset, info) == nil {
			t.Fatal("missing operand edge is accepted")
		}
	}
}

func TestIndependentCFGRejectsSwappedOutcomes(t *testing.T) {
	source := `package example; func f(c bool) int { if c { return 1 }; return 2 }`
	graph := parseIR(t, source, pdg.NewBuilder())
	for i, block := range graph.NormalFlow.Blocks {
		if len(block.Successors) == 2 {
			graph.NormalFlow.Blocks[i].Successors[0], graph.NormalFlow.Blocks[i].Successors[1] = block.Successors[1], block.Successors[0]
			break
		}
	}
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal("mutation must remain structurally valid", err)
	}
	fn, fset, _ := independentSource(t, source)
	if govalidation.CFG(&graph, fn, fset) == nil {
		t.Fatal("swapped true/false outcomes are accepted")
	}
}

func TestIndependentBindingsRejectWrongRead(t *testing.T) {
	source := `package example; func f(x,y int) int { return x }`
	graph := parseIR(t, source, pdg.NewBuilder())
	for _, node := range graph.Nodes {
		if node.Category != pdg.Read {
			continue
		}
		set := graph.Attributes[node.Attributes-1]
		graph.AttributeTables.Accesses[set.Reads-1][0].Symbol = graph.Function.Inputs[1].Symbol
	}
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal("mutation must remain structurally valid", err)
	}
	fn, fset, info := independentSource(t, source)
	if govalidation.Bindings(&graph, fn, fset, info) == nil {
		t.Fatal("wrong binding is accepted")
	}
}

func TestIndependentOrderingRejectsInventedDependency(t *testing.T) {
	source := `package example; func f(x int) int { return x }`
	pool := pdg.NewBuilder()
	graph := parseIR(t, source, pool)
	for _, edge := range graph.Edges {
		if edge.Kind != pdg.Execution {
			continue
		}
		edge.ID = pool.Text("invented-execution")
		edge.Source, edge.Target = edge.Target, edge.Source
		graph.Edges = append(graph.Edges, edge)
		break
	}
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal("mutation must remain structurally valid", err)
	}
	if govalidation.Ordering(&graph) == nil {
		t.Fatal("invented execution dependency is accepted")
	}
}

func TestIndependentScalarInventory(t *testing.T) {
	functions := []string{
		`type Scalar int; func (x Scalar) f() (y Scalar) { y=x; return }`,
		`func f(x int) int { return x }`,
		`func f(x int,c bool) int { if c { x=1 }; return x }`,
		`func f(x,y bool) bool { return x && !y }`,
		`func f(x,y bool) bool { return (x || y) && x }`,
		`func f(x int) int { var y int; y=x; y++; return y }`,
		`func f(x int) int { x+=1; return x }`,
		`func f(x int) int { (x)++; return x }`,
		`func f(x int) int { (x)+=1; return x }`,
		`func f() (x int) { return }`,
	}
	for _, function := range functions {
		t.Run(function, func(t *testing.T) {
			source := "package example; " + function
			graph := parseIR(t, source, pdg.NewBuilder())
			if err := sourceInventory(t, &graph, source); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestIndependentInventoryRejectsProducerCategoryClaim(t *testing.T) {
	source := `package example; func f(x int) int { return x }`
	graph := parseIR(t, source, pdg.NewBuilder())
	for i, node := range graph.Nodes {
		if node.Category == pdg.Return {
			graph.Nodes[i].Category = pdg.Computation
		}
	}
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal("mutation must remain structurally valid", err)
	}
	if err := sourceInventory(t, &graph, source); err == nil {
		t.Fatal("producer category claim is trusted")
	}
}
