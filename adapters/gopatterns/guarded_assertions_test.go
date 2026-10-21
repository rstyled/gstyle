package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

func TestGuardedAssertions(t *testing.T) {
	for _, test := range []struct {
		name, body string
		want       int
	}{
		{"type case", `switch v.(type){case string:_=v.(string)}`, 0},
		{"bytes case loop", `switch v.(type){case []byte:for range v.([]byte){_ = len(v.([]byte))}}`, 0},
		{"guard return", `if _,ok:=v.(string);!ok{return};_=v.(string)`, 0},
		{"successful branch", `if _,ok:=v.(string);ok{_=v.(string)}`, 0},
		{"else branch", `if _,ok:=v.(string);!ok{}else{_=v.(string)}`, 0},
		{"unrelated call", `if _,ok:=v.(string);!ok{return};touch();_=v.(string)`, 0},
		{"shadowed write", `if _,ok:=v.(string);!ok{return};{v:=7;_=v};_=v.(string)`, 0},
		{"alias type", `switch v.(type){case Alias:_=v.(string)}`, 0},
		{"range binding", `for _,v:=range []any{v}{if _,ok:=v.(string);ok{_=v.(string)}}`, 0},
		{"unprotected", `_=v.(string)`, 1},
		{"wrong type", `switch v.(type){case int:_=v.(string)}`, 1},
		{"case overwrite", `switch v.(type){case string:v=7;_=v.(string)}`, 1},
		{"guard overwrite", `if _,ok:=v.(string);!ok{return};v=7;_=v.(string)`, 1},
		{"branch overwrite", `if _,ok:=v.(string);!ok{return};if flag{v=7};_=v.(string)`, 1},
		{"wrong branch", `if _,ok:=v.(string);!ok{_=v.(string)}`, 1},
		{"wrong variable", `if _,ok:=other.(string);!ok{return};_=v.(string)`, 1},
		{"shadowed assertion", `if _,ok:=v.(string);!ok{return};{var v any=7;_=v.(string)}`, 1},
		{"address escape", `if _,ok:=v.(string);!ok{return};replace(&v);_=v.(string)`, 1},
		{"closure write", `if _,ok:=v.(string);!ok{return};func(){v=7}();_=v.(string)`, 1},
		{"closure assertion", `if _,ok:=v.(string);!ok{return};defer func(){_=v.(string)}();v=7`, 1},
		{"goto bypass", `if flag{goto use};if _,ok:=v.(string);!ok{return};use:_=v.(string)`, 1},
		{"multi case", `switch v.(type){case string,int:_=v.(string)}`, 1},
		{"global", `if _,ok:=global.(string);!ok{return};touch();_=global.(string)`, 1},
		{"shared field", `if _,ok:=box.value.(string);!ok{return};touch();_=box.value.(string)`, 1},
		{"stale boolean", `_,ok:=v.(string);ok=true;if ok{_=v.(string)}`, 1},
		{"reused boolean", `var ok bool;if _,ok=v.(string);ok{_=v.(string)}`, 1},
		{"shared boolean", `if _,globalOK=v.(string);globalOK{_=v.(string)}`, 1},
		{"nested shadowed boolean", `if _,ok:=v.(string);ok{if _,ok:=other.(int);ok{_=v.(int)}}`, 1},
		{"loop invalidation", `if _,ok:=v.(string);!ok{return};for flag{_=v.(string);v=7}`, 1},
		{"typed nil pointer case", `switch v.(type){case *int:_=v.(*int)}`, 0},
		{"empty interface case", `switch v.(type){case any:_=v.(any)}`, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			fn, fs, info := parseNilErrFunc(t, `package p
type Alias = string
var global any
var globalOK bool
func check(v,other any,flag bool,box *struct{value any}){`+test.body+`}
func touch(){}
func replace(v *any){*v=7}
`)
			graph, err := buildGraph(&packages.Package{Fset: fs, TypesInfo: info}, fn, "test.go", pdg.NewBuilder())
			if err != nil {
				t.Fatal(err)
			}
			proven := guardedAssertionsForTest(fn, fs, info, &graph)
			if hits := findForceTypeAssertHits(fn, fs, proven); len(hits) != test.want {
				t.Fatalf("got %+v, want %d warnings", hits, test.want)
			}
		})
	}
}

// The assertion proof must follow canonical edges, not lexical nesting.
func TestGuardedAssertionUsesCanonicalFlow(t *testing.T) {
	fn, fs, info := parseNilErrFunc(t, `package p;func check(v any){if _,ok:=v.(string);!ok{return};_=v.(string)}`)
	graph, err := buildGraph(&packages.Package{Fset: fs, TypesInfo: info}, fn, "test.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	if len(guardedAssertionsForTest(fn, fs, info, &graph)) != 1 {
		t.Fatal("missing guard proof")
	}
	flow := pdg.BodyFlow(&graph)
	for i, block := range flow.Blocks {
		if len(block.Successors) == 2 {
			flow.Blocks[i].Successors[0] = block.Successors[1]
		}
	}
	flow.RefreshReachability()
	if proven := guardedAssertionsForTest(fn, fs, info, &graph); len(proven) != 0 {
		t.Fatalf("bypassed guard still proves assertions: %v", proven)
	}
	graph.NormalFlow, graph.ValueFlow, graph.Extensions = nil, nil, nil
	if proven := guardedAssertionsForTest(fn, fs, info, &graph); len(proven) != 0 {
		t.Fatalf("missing flow still proves assertions: %v", proven)
	}
}

func TestGuardedAssertionNeedsCanonicalOperands(t *testing.T) {
	for _, test := range []struct{ name, body, kind string }{
		{"case input", `switch v.(type){case string:_=v.(string)}`, "control-value"},
		{"assertion input", `if _,ok:=v.(string);!ok{return};_=v.(string)`, "source-value"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fn, fs, info := parseNilErrFunc(t, `package p;func check(v any){`+test.body+`}`)
			graph, err := buildGraph(&packages.Package{Fset: fs, TypesInfo: info}, fn, "test.go", pdg.NewBuilder())
			if err != nil {
				t.Fatal(err)
			}
			if len(guardedAssertionsForTest(fn, fs, info, &graph)) != 1 {
				t.Fatal("missing original proof")
			}
			kept := graph.Edges[:0]
			removed := 0
			for _, edge := range graph.Edges {
				if graph.Tables.Text(edge.Subkind) == test.kind {
					removed++
					continue
				}
				kept = append(kept, edge)
			}
			graph.Edges = kept
			if removed == 0 {
				t.Fatal("fixture lacks the operand edges under test")
			}
			if proven := guardedAssertionsForTest(fn, fs, info, &graph); len(proven) != 0 {
				t.Fatalf("missing operands still prove assertions: %v", proven)
			}
		})
	}
}

func guardedAssertionsForTest(fn *ast.FuncDecl, fset *token.FileSet, info *types.Info, graph *pdg.Graph) map[*ast.TypeAssertExpr]bool {
	return provenTypeAssertions(&packages.Package{Fset: fset, TypesInfo: info}, functionGraph{declaration: fn, graph: graph}, nil)
}
