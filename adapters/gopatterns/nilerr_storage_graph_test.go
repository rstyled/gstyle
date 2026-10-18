package gopatterns

import (
	"go/ast"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

func TestNilErrStorageEdgesControlIdentity(t *testing.T) {
	for _, mode := range []string{"base input", "base root", "field index", "field fact", "identity input"} {
		t.Run(mode, func(t *testing.T) {
			body := `_=q;if p.Err!=nil{if p.Err==nil{return p.Err};return nil};return nil`
			if mode == "identity input" {
				body = `_=q;if (*R)(p).Err!=nil{return nil};return nil`
			}
			body = strings.ReplaceAll(body, "if ", "\nif ")
			fn, fs, info := parseNilErrFunc(t, `package p;func f(p,q *R)error{`+body+`};type R struct{Err,Other error}`)
			pool := pdg.NewBuilder()
			graph, err := buildGraph(&packages.Package{Fset: fs, TypesInfo: info}, fn, "test.go", pool)
			if err != nil {
				t.Fatal(err)
			}
			initial := findNilErrGraphHits(fn, fs, info, &graph)
			if len(initial) != 1 {
				t.Fatalf("initial hits=%v", initial)
			}
			query := newNilErrGraph(fn, fs, &graph)
			var fields []pdg.Ref
			var other, conversion pdg.Ref
			for ref, node := range query.source {
				switch node := node.(type) {
				case *ast.SelectorExpr:
					if node.Sel.Name == "Err" {
						fields = append(fields, ref)
					}
				case *ast.Ident:
					if node.Name == "q" {
						other = ref
					}
				case *ast.CallExpr:
					conversion = ref
				}
			}
			sort.Slice(fields, func(i, j int) bool { return query.source[fields[i]].Pos() < query.source[fields[j]].Pos() })
			if len(fields) == 0 {
				t.Fatal("missing field read")
			}
			target := fields[0]
			if mode == "base root" || mode == "field index" {
				target = fields[1]
			}
			if mode == "field index" || mode == "field fact" {
				replaceStorageFact(t, &graph, target, pool, mode == "field fact")
			} else {
				if mode == "identity input" {
					target = conversion
				}
				changed := 0
				var edges []pdg.Edge
				for _, edge := range graph.Edges {
					kind := graph.Tables.Text(edge.Subkind)
					value := edge.Kind == pdg.Data && (kind == "source-value" || kind == "binding-value" || kind == "scalar-value")
					if value && edge.Target == target && edge.Position == 1 {
						changed++
						if mode != "base root" {
							continue
						}
						if other == 0 {
							t.Fatal("missing alternate root")
						}
						edge.Source = other
					}
					edges = append(edges, edge)
				}
				if changed == 0 {
					t.Fatal("no storage input changed")
				}
				graph.Edges = edges
			}
			if err := pdg.Validate(&graph); err != nil {
				t.Fatalf("modified graph must remain structurally valid: %v", err)
			}
			hits := findNilErrGraphHits(fn, fs, info, &graph)
			if mode == "base root" || mode == "field index" {
				// The outer loss is no longer proved. The independent inner error guard
				// now has its own discarded-error path, and must report its own line.
				wantLine := fs.Position(query.source[target].Pos()).Line
				if len(hits) != 1 || hits[0].Line != wantLine || hits[0].Line == initial[0].Line {
					t.Fatalf("changed identity reports %v, want only inner line %d", hits, wantLine)
				}
				return
			}
			if len(hits) != 0 {
				t.Fatalf("source storage syntax overrides %s: %v", mode, hits)
			}
		})
	}
}
func replaceStorageFact(t *testing.T, graph *pdg.Graph, ref pdg.Ref, pool *pdg.Builder, remove bool) {
	t.Helper()
	attrs := make([]pdg.Attributes, len(graph.Nodes))
	for i, node := range graph.Nodes {
		attrs[i] = graph.NodeAttributes(node)
	}
	extensions := slices.Clone(attrs[ref-1].Extensions)
	var kept []pdg.Extension
	changed := false
	for _, extension := range extensions {
		if graph.Tables.Text(extension.Name) == "gstyle.go:field-path" {
			changed = true
			if remove {
				continue
			}
			extension.Value = pool.Text(`[1]`)
		}
		kept = append(kept, extension)
	}
	if !changed {
		t.Fatal("missing compiler field fact")
	}
	attrs[ref-1].Extensions = kept
	graph.Attributes = nil
	graph.AttributeTables = pdg.AttributeTables{}
	for i := range graph.Nodes {
		graph.Nodes[i].Attributes = pdg.Ref(i + 1)
	}
	if err := pdg.PackAttributes(graph, attrs); err != nil {
		t.Fatal(err)
	}
}
func TestNilErrCanonicalStorageAndHandlingCases(t *testing.T) {
	cases := []struct {
		name, body string
		want       int
	}{
		{"identity receiver", `if (*R)(p).Err!=nil{if (*R)(p).Err==nil{return p.Err};return nil};return nil`, 1},
		{"unused closure", `if err!=nil{cb:=func(){};_=cb;return nil};return err`, 1},
		{"uncalled capture", `if err!=nil{cb:=func(){_=err};_=cb;return nil};return err`, 1},
		{"blank error declaration", `if err!=nil{var _=err;return nil};return err`, 1},
		{"unrelated aggregate", `if err!=nil{value:=struct{P *R}{P:p};_=value;return nil};return err`, 1},
		{"aggregate copy", `if err!=nil{values:=[]error{err};_=values;return nil};return err`, 0},
		{"shadowed print", `if err!=nil{println(err);return nil};return err`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			suffix := `;type R struct{Err error}`
			if tc.name == "shadowed print" {
				suffix += `;func println(error){}`
			}
			fn, fs, info := parseNilErrFunc(t, `package p;func f(err error,p *R)error{`+tc.body+`}`+suffix)
			if hits := findNilErrHits(fn, fs, info); len(hits) != tc.want {
				t.Fatalf("hits=%v want %d", hits, tc.want)
			}
		})
	}
}
