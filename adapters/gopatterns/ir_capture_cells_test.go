package gopatterns

import (
	"encoding/json"
	"fmt"
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"path/filepath"
	"slices"
	"testing"
)

func TestLexicalCaptureCells(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		counts       []int
	}{
		{"parameter", `func f(flag bool){func(){_=flag}()}`, []int{1}},
		{"shadow parameter", `func f(){flag:=true;func(flag bool){_=flag}(false);_=flag}`, []int{0}},
		{"shadow local", `func f(){flag:=true;func(){flag:=false;_=flag}();_=flag}`, []int{0}},
		{"named result", `func f()(flag bool){func(){flag=true}();return}`, []int{1}},
		{"receiver", `type item struct{};func(x *item) f(){func(){_=x}()}`, []int{1}},
		{"globals constants and fields", `var global bool;const fixed=true;func f(){func(){_=global;_=fixed;var x struct{field bool};_=x.field}()}`, []int{0}},
		{"outer field base", `func f(x struct{Field bool}){func(){_=x.Field}()}`, []int{1}},
		{"relay", `func f(flag bool){func(){func(){_=flag}()}()}`, []int{1, 1}},
		{"shadow relay", `func f(x bool){func(){x:=false;func(){_=x}()}()}`, []int{0, 1}},
		{"mixed locals and captures", `func f(x bool){y:=true;func(arg bool){local:=false;_=arg;_=local;_=x;_=y}(false)}`, []int{2}},
		{"indirect write", `func f(){x:=false;func(){p:=&x;*p=true}()}`, []int{1}},
		{"named type", `func f(){type Flag bool;var x Flag;func(){_=x}()}`, []int{1}},
		{"range variable", `func f(xs []bool){for _,x:=range xs{func(){_=x}()}}`, []int{1}},
		{"generic parameter", `func f[T any](x T){func(){_=x}()}`, []int{1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, "package example;"+tc.source)
			graphs := map[string]*pdg.Graph{}
			for _, graph := range extraction.Callables {
				graphs[graph.Tables.Text(graph.Function.ID)] = graph
				for _, parameter := range append(slices.Clone(graph.Function.Inputs), graph.Function.Outputs...) {
					if parameter.Symbol != 0 && graph.Symbols[parameter.Symbol-1].Role == pdg.CaptureRole {
						t.Fatal("own formal parameter becomes capture")
					}
				}
			}
			var counts []int
			for _, body := range extraction.Callables {
				environment, found := testCaptureEnvironment(t, body)
				if !found {
					continue
				}
				links, err := bindingflow.CaptureBindings(body, graphs[environment.Owner])
				if err != nil {
					t.Fatal(err)
				}
				if links == nil {
					t.Fatal("explicit environment returns unknown links")
				}
				counts = append(counts, len(links))
				for inner, outer := range links {
					if body.Symbols[inner-1].Role != pdg.CaptureRole {
						t.Fatal("captured cell is not classified as capture")
					}
					if graphs[environment.Owner].Symbols[outer-1].Role == pdg.GlobalRole {
						t.Fatal("package global becomes captured cell")
					}
					assertCaptureMemoryInput(t, body, inner)
				}
				assertRelayCaptureEdges(t, body)
			}
			if !slices.Equal(counts, tc.counts) {
				t.Fatalf("capture counts=%v want %v", counts, tc.counts)
			}
		})
	}
}

func testCaptureEnvironment(t *testing.T, graph *pdg.Graph) (bindingflow.CaptureEnvironment, bool) {
	t.Helper()
	for _, extension := range graph.Extensions {
		if graph.Tables.Text(extension.Name) != "gstyle.closure:environment" {
			continue
		}
		var environment bindingflow.CaptureEnvironment
		if err := json.Unmarshal([]byte(graph.Tables.Text(extension.Value)), &environment); err != nil {
			t.Fatal(err)
		}
		return environment, true
	}
	return bindingflow.CaptureEnvironment{}, false
}

func assertCaptureMemoryInput(t *testing.T, body *pdg.Graph, symbol pdg.Ref) {
	t.Helper()
	for _, definition := range body.Definitions {
		if definition.Symbol == symbol && body.Tables.Text(body.NodeAttributes(body.Nodes[definition.Node-1]).Construct) == "memory-input" {
			return
		}
	}
	t.Fatal("captured cell lacks an external memory definition")
}

func assertRelayCaptureEdges(t *testing.T, body *pdg.Graph) {
	t.Helper()
	for _, ref := range closureNodes(body) {
		for _, symbol := range body.NodeAttributes(body.Nodes[ref-1]).Symbols {
			found := false
			for _, edge := range body.Edges {
				if edge.Target == ref && edge.Symbol == symbol && edge.Location != 0 && body.Tables.Text(edge.Subkind) == "capture-address" {
					found = true
				}
			}
			if !found {
				t.Fatal("relayed captured cell lacks a construction address edge")
			}
		}
	}
}

func TestCaptureBindingsRejectMalformedOrMismatchedCells(t *testing.T) {
	pkg := loadTestPackage(t, `package example;func f(x bool){if x{x:=false;func(){_=x}()}}`)
	p := packageAnalysis{sources: map[string]string{pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename: "fixture-digest"}, pkg: pkg, root: filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename), pool: pdg.NewBuilder()}
	if err := p.build(); err != nil {
		t.Fatal(err)
	}
	owner, body := p.graphs[0].graph, p.graphs[1].graph
	for _, pair := range [][2]*pdg.Graph{{nil, owner}, {body, nil}, {&pdg.Graph{}, owner}} {
		if _, err := bindingflow.CaptureBindings(pair[0], pair[1]); err == nil {
			t.Fatal("invalid native graph accepted")
		}
	}
	environment, _ := testCaptureEnvironment(t, body)
	encoded, _ := json.Marshal(environment)
	wrongSource := ""
	for _, symbol := range owner.Symbols {
		if symbol.Role == pdg.ParameterRole {
			wrongSource = owner.Tables.Text(symbol.ID)
		}
	}
	mismatched := environment
	mismatched.Cells = []bindingflow.CaptureCell{{Symbol: environment.Cells[0].Symbol, Source: wrongSource}}
	mismatch, _ := json.Marshal(mismatched)
	for _, data := range []string{"null", "{}", "{", `{"owner":"other","cells":[]}`, fmt.Sprintf(`{"owner":%q,"cells":null}`, environment.Owner), fmt.Sprintf(`{"owner":%q,"cells":[]}`, environment.Owner), string(mismatch), fmt.Sprintf(`{"owner":%q,"cells":[{"symbol":"absent","source":"absent"}]}`, environment.Owner)} {
		t.Run(data, func(t *testing.T) {
			copy := *body
			copy.Extensions = p.pool.SingletonExtension("gstyle.closure:environment", data)
			if _, err := bindingflow.CaptureBindings(&copy, owner); err == nil {
				t.Fatal("invalid capture environment accepted")
			}
		})
	}
	duplicate := *body
	duplicate.Extensions = append(p.pool.SingletonExtension("gstyle.closure:environment", string(encoded)), p.pool.SingletonExtension("gstyle.closure:environment", string(encoded))...)
	if _, err := bindingflow.CaptureBindings(&duplicate, owner); err == nil {
		t.Fatal("duplicate environment accepted")
	}
	environment.Cells = append(environment.Cells, environment.Cells[0])
	encoded, _ = json.Marshal(environment)
	duplicate.Extensions = p.pool.SingletonExtension("gstyle.closure:environment", string(encoded))
	if _, err := bindingflow.CaptureBindings(&duplicate, owner); err == nil {
		t.Fatal("duplicate cell accepted")
	}
	changed := *owner
	changed.Sources = slices.Clone(owner.Sources)
	changed.Sources[0].ContentIdentity = p.pool.Known("changed-source")
	if _, err := bindingflow.CaptureBindings(body, &changed); err == nil {
		t.Fatal("mixed source revisions accepted")
	}
	emptyBody, emptyOwner := *body, *owner
	emptyBody.Sources = slices.Clone(body.Sources)
	emptyOwner.Sources = slices.Clone(owner.Sources)
	emptyBody.Sources[0].Snapshot = 0
	emptyOwner.Sources[0].Snapshot = 0
	emptyBody.Sources[0].ContentIdentity = p.pool.Known("")
	emptyOwner.Sources[0].ContentIdentity = p.pool.Known("")
	if err := pdg.Validate(&emptyBody); err != nil {
		t.Fatal(err)
	}
	if err := pdg.Validate(&emptyOwner); err != nil {
		t.Fatal(err)
	}
	if _, err := bindingflow.CaptureBindings(&emptyBody, &emptyOwner); err == nil {
		t.Fatal("empty known revisions certify a capture link")
	}
	changed.Sources[0].ContentIdentity = p.pool.Missing(pdg.Unknown, "no snapshot")
	if _, err := bindingflow.CaptureBindings(body, &changed); err == nil {
		t.Fatal("unknown source revision accepted")
	}
	missing := *body
	missing.Extensions = nil
	if links, err := bindingflow.CaptureBindings(&missing, owner); err != nil || links != nil {
		t.Fatalf("missing evidence=%v %v", links, err)
	}
	links, err := bindingflow.CaptureBindings(body, owner)
	if err != nil {
		t.Fatal(err)
	}
	clear(links)
	again, err := bindingflow.CaptureBindings(body, owner)
	if err != nil || len(again) != 1 {
		t.Fatalf("returned map mutates borrowed environment: %v %v", again, err)
	}
}

func TestCaptureBindingsCompareTextAcrossInternPools(t *testing.T) {
	pkg := loadTestPackage(t, `package example;func f(x bool){func(){_=x}()}`)
	filename := pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename
	var analyses []*packageAnalysis
	for _, seed := range []string{"", "different intern prefix"} {
		pool := pdg.NewBuilder()
		if seed != "" {
			pool.Text(seed)
		}
		analysis := &packageAnalysis{pkg: pkg, root: filepath.Dir(filename), pool: pool, sources: map[string]string{filename: "same-source"}}
		if err := analysis.build(); err != nil {
			t.Fatal(err)
		}
		analyses = append(analyses, analysis)
	}
	owner, body := analyses[0].graphs[0].graph, analyses[1].graphs[1].graph
	if owner.Tables == body.Tables || owner.Symbols[0].Scope == body.Symbols[0].Scope {
		t.Fatal("fixture does not use distinct intern references")
	}
	links, err := bindingflow.CaptureBindings(body, owner)
	if err != nil || len(links) != 1 {
		t.Fatalf("cross-pool binding=%v %v", links, err)
	}
}

func TestClosureInventoryRequiresCompilerScope(t *testing.T) {
	pkg := loadTestPackage(t, `package example;func f(){func(){}()}`)
	copied := *pkg
	info := *pkg.TypesInfo
	info.Scopes = nil
	copied.TypesInfo = &info
	p := packageAnalysis{pkg: &copied}
	var literal *ast.FuncLit
	ast.Inspect(pkg.Syntax[0], func(node ast.Node) bool {
		if value, ok := node.(*ast.FuncLit); ok {
			literal = value
		}
		return true
	})
	if literal == nil {
		t.Fatal("missing fixture closure")
	}
	if _, _, err := p.closurePackage(literal, "testp.f.closure@1"); err == nil {
		t.Fatal("missing compiler scope certifies an empty environment")
	}
}
