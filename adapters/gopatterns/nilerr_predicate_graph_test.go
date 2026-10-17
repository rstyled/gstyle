package gopatterns

import (
	"go/ast"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

func TestNilErrPredicateEdgesControlFindings(t *testing.T) {
	for _, mode := range []string{"guard input", "comparison input", "comparison value", "parentheses input", "identity conversion", "nil conversion", "negation input", "guard operator", "subject parentheses", "subject conversion"} {
		t.Run(mode, func(t *testing.T) {
			body := `if err!=nil{return nil};return err`
			switch mode {
			case "comparison input", "comparison value":
				body = `if err!=nil{n:=1;_=2;if n==1{return nil};return err};return err`
			case "parentheses input":
				body = `if err!=nil{if (b){return nil};return err};return err`
			case "identity conversion":
				body = `if err!=nil{alias:=bool(b);if alias{return nil};return err};return err`
			case "nil conversion":
				body = `empty:=error(nil);if err!=nil{return empty};return err`
			case "subject parentheses":
				body = `if (err)!=nil{return nil};return err`
			case "subject conversion":
				body = `if error(err)!=nil{return nil};return err`
			case "negation input":
				body = `if !(err==nil){return nil};return err`
			}
			fn, fs, info := parseNilErrFunc(t, `package p;func f(err error,b bool)error{`+body+`}`)
			pool := pdg.NewBuilder()
			graph, err := buildGraph(&packages.Package{Fset: fs, TypesInfo: info}, fn, "test.go", pool)
			if err != nil {
				t.Fatal(err)
			}
			if hits := findNilErrGraphHits(fn, fs, info, &graph); len(hits) != 1 {
				t.Fatalf("initial hits=%v", hits)
			}
			q := newNilErrGraph(fn, fs, &graph)
			var target, other pdg.Ref
			position := pdg.Position(1)
			for ref, node := range q.source {
				switch node := node.(type) {
				case *ast.BinaryExpr:
					if (mode == "guard input" || mode == "guard operator") && node.Op.String() == "!=" {
						target = ref
						position = 2
					}
					if (mode == "comparison input" || mode == "comparison value") && node.Op.String() == "==" {
						target = ref
					}
				case *ast.BasicLit:
					if node.Value == "2" {
						other = ref
					}
				case *ast.ParenExpr:
					if mode == "parentheses input" || mode == "subject parentheses" {
						target = ref
					}
				case *ast.CallExpr:
					if mode == "identity conversion" || mode == "nil conversion" || mode == "subject conversion" {
						target = ref
					}
				case *ast.UnaryExpr:
					if mode == "negation input" {
						target = ref
					}
				}
			}
			if target == 0 {
				t.Fatal("missing predicate operation")
			}
			changed := 0
			if mode == "guard operator" {
				replacePredicateOperator(t, &graph, target, pool.Known("=="))
				changed++
			} else {
				if mode == "comparison value" {
					position = 2
					if other == 0 {
						t.Fatal("missing replacement constant")
					}
				}
				var edges []pdg.Edge
				for _, edge := range graph.Edges {
					kind := graph.Tables.Text(edge.Subkind)
					value := edge.Kind == pdg.Data && (kind == "source-value" || kind == "binding-value" || kind == "scalar-value")
					if value && edge.Target == target && edge.Position == position {
						changed++
						if mode != "comparison value" {
							continue
						}
						edge.Source = other
					}
					edges = append(edges, edge)
				}
				graph.Edges = edges
			}
			if changed == 0 {
				t.Fatal("no canonical dependency changed")
			}
			if err := pdg.Validate(&graph); err != nil {
				t.Fatalf("modified graph must remain structurally valid: %v", err)
			}
			if hits := findNilErrGraphHits(fn, fs, info, &graph); len(hits) != 0 {
				t.Fatalf("source predicate overrides %s: %v", mode, hits)
			}
		})
	}
}
func replacePredicateOperator(t *testing.T, graph *pdg.Graph, ref pdg.Ref, operator pdg.Fact) {
	t.Helper()
	attrs := make([]pdg.Attributes, len(graph.Nodes))
	for i, node := range graph.Nodes {
		attrs[i] = graph.NodeAttributes(node)
	}
	attrs[ref-1].Operator = operator
	graph.Attributes = nil
	graph.AttributeTables = pdg.AttributeTables{}
	for i := range graph.Nodes {
		graph.Nodes[i].Attributes = pdg.Ref(i + 1)
	}
	if err := pdg.PackAttributes(graph, attrs); err != nil {
		t.Fatal(err)
	}
}
