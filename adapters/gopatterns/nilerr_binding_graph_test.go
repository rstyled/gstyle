package gopatterns

import (
	"go/ast"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

func TestNilErrCanonicalBindingCopies(t *testing.T) {
	cases := []struct {
		name, body string
		want       int
	}{
		{"blank slot", `a,_,b:=1,99,0;_=a;if err!=nil{return b,nil};return 0,err`, 1},
		{"parallel snapshot", `a,b:=0,1;a,b=b,a;_=a;if err!=nil{return b,nil};return 0,err`, 1},
		{"parallel nonzero", `a,b:=0,1;a,b=b,a;_=b;if err!=nil{return a,nil};return 0,err`, 0},
		{"compound changes zero", `result:=0;result+=1;if err!=nil{return result,nil};return 0,err`, 0},
		{"overwrite zero", `result:=1;result=0;if err!=nil{return result,nil};return 0,err`, 1},
		{"weak alias is not definite", `x:=1;y:=1;p:=&y;if false{p=&x};*p=0;if err!=nil{return x,nil};return 0,err`, 0},
		{"overwrite nonzero", `result:=0;result=1;if err!=nil{return result,nil};return 0,err`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn, fs, info := parseNilErrFunc(t, `package p;func f(err error)(int,error){`+tc.body+`}`)
			if hits := findNilErrHits(fn, fs, info); len(hits) != tc.want {
				t.Fatalf("hits=%v want %d", hits, tc.want)
			}
		})
	}
}
func TestNilErrValueEdgesControlFindings(t *testing.T) {
	for _, mode := range []string{"missing copy", "nonzero copy", "missing return"} {
		t.Run(mode, func(t *testing.T) {
			fn, fs, info := parseNilErrFunc(t, `package p;func f(err error)(int,error){result:=0;_=1;if err!=nil{return result,nil};return 0,err}`)
			graph, err := buildGraph(&packages.Package{Fset: fs, TypesInfo: info}, fn, "test.go", pdg.NewBuilder())
			if err != nil {
				t.Fatal(err)
			}
			if hits := findNilErrGraphHits(fn, fs, info, &graph); len(hits) != 1 {
				t.Fatalf("initial hits=%v", hits)
			}
			q := newNilErrGraph(fn, fs, &graph)
			var assignment, nonzero, returned pdg.Ref
			for ref, node := range q.source {
				switch node := node.(type) {
				case *ast.AssignStmt:
					if len(graph.NodeAttributes(graph.Nodes[ref-1]).Writes) > 0 {
						assignment = ref
					}
				case *ast.BasicLit:
					if node.Value == "1" {
						nonzero = ref
					}
				case *ast.ReturnStmt:
					if len(node.Results) == 2 && info.Types[node.Results[1]].IsNil() {
						returned = ref
					}
				}
			}
			if assignment == 0 || nonzero == 0 || returned == 0 {
				t.Fatal("missing source sites")
			}
			changed := 0
			var edges []pdg.Edge
			for _, edge := range graph.Edges {
				kind := graph.Tables.Text(edge.Subkind)
				value := edge.Kind == pdg.Data && (kind == "source-value" || kind == "binding-value" || kind == "scalar-value")
				copyEdge := value && edge.Target == assignment && edge.Position == 1
				returnEdge := value && edge.Target == returned && edge.Position == 2
				if mode == "missing copy" && copyEdge {
					changed++
					continue
				}
				if mode == "nonzero copy" && copyEdge {
					edge.Source = nonzero
					changed++
				}
				if mode == "missing return" && returnEdge {
					changed++
					continue
				}
				edges = append(edges, edge)
			}
			graph.Edges = edges
			if changed == 0 {
				t.Fatal("no canonical operand changed")
			}
			if err := pdg.Validate(&graph); err != nil {
				t.Fatalf("mutation must preserve structural validity: %v", err)
			}
			if hits := findNilErrGraphHits(fn, fs, info, &graph); len(hits) != 0 {
				t.Fatalf("source list overrides %s: %v", mode, hits)
			}
		})
	}
}
