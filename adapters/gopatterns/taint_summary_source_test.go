package gopatterns

import (
	"context"
	"fmt"
	"github.com/rstyled/gstyle/domain/patterns"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sourceTaintFlows(t *testing.T, body string) []patterns.TaintFlow {
	t.Helper()
	files := map[string]string{"go.mod": "module example\n\ngo 1.26.0\n", "p.go": `package example
import("database/sql";"net/http")
var db *sql.DB
` + body}
	return moduleTaintFlows(t, files)
}
func moduleTaintFlows(t *testing.T, files map[string]string) []patterns.TaintFlow {
	t.Helper()
	root := t.TempDir()
	for name, source := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	paths := []string{"."}
	if _, ok := files["helper/helper.go"]; ok {
		paths = append(paths, "helper")
	}
	extraction, err := ExtractForFix(context.Background(), root, paths)
	if err != nil {
		t.Fatal(err)
	}
	var facts []*patterns.MiningFacts
	for _, fn := range extraction.Funcs {
		graph := fn.Pdg
		facts = append(facts, &patterns.MiningFacts{ID: graph.Tables.Text(graph.Function.ID), Name: fn.Name, Path: fn.Path, Canonical: &graph})
	}
	flows, err := patterns.QueryTaintFlows(facts)
	if err != nil {
		t.Fatal(err)
	}
	return flows
}
func TestTaintSummariesSourceBoundaries(t *testing.T) {
	cases := []struct {
		name, source string
		want         int
	}{
		{"argument return", `func copy(v string)string{return v};func handler(r *http.Request){db.Exec(copy(r.FormValue("q")))}`, 1},
		{"variadic tail", `func pick(v ...string)string{return v[1]};func handler(r *http.Request){db.Exec(pick("SELECT 1",r.FormValue("q")))}`, 1},
		{"constant return", `func copy(v string)string{return "SELECT 1"};func handler(r *http.Request){db.Exec(copy(r.FormValue("q")))}`, 0},
		{"source wrapper", `func input(r *http.Request)string{return r.FormValue("q")};func handler(r *http.Request){db.Exec(input(r))}`, 1},
		{"sink wrapper", `func execute(v string){db.Exec(v)};func handler(r *http.Request){execute(r.FormValue("q"))}`, 1},
		{"result slots", `func pair(v string)(string,string){return v,"SELECT 1"};func handler(r *http.Request){_,clean:=pair(r.FormValue("q"));db.Exec(clean)}`, 0},
		{"recursive helper", `func copy(v string,b bool)string{if b{return v};return copy(v,true)};func handler(r *http.Request){db.Exec(copy(r.FormValue("q"),false))}`, 1},
		{"factory builder", `type Builder struct{q string};func New()*Builder{return &Builder{}};func(b *Builder)Set(v string){b.q=v};func(b *Builder)Query()string{return b.q};func handler(r *http.Request){b:=New();b.Set(r.FormValue("q"));db.Exec(b.Query())}`, 1},
		{"builder field", `type Builder struct{q string};func(b *Builder)Set(v string){b.q=v};func(b *Builder)Query()string{return b.q};func handler(r *http.Request){b:=&Builder{};b.Set(r.FormValue("q"));db.Exec(b.Query())}`, 1},
		{"separate builders", `type Builder struct{q string};func(b *Builder)Set(v string){b.q=v};func(b *Builder)Query()string{return b.q};func handler(r *http.Request){a,b:=&Builder{},&Builder{};a.Set(r.FormValue("q"));db.Exec(b.Query())}`, 0},
		{"safe overwrite", `type Builder struct{q string};func(b *Builder)Set(v string){b.q=v};func(b *Builder)Query()string{return b.q};func handler(r *http.Request){b:=&Builder{};b.Set(r.FormValue("q"));b.Set("SELECT 1");db.Exec(b.Query())}`, 0},
		{"conditional write retains old", `type Builder struct{q string};func(b *Builder)Set(v string,yes bool){if yes{b.q=v}};func(b *Builder)Query()string{return b.q};func handler(r *http.Request,choose bool){b:=&Builder{};b.Set(r.FormValue("q"),true);b.Set("SELECT 1",choose);db.Exec(b.Query())}`, 1},
		{"both branches overwrite", `type Builder struct{q string};func(b *Builder)Set(v string){b.q=v};func(b *Builder)Clear(yes bool){if yes{b.q="SELECT 1"}else{b.q="SELECT 2"}};func(b *Builder)Query()string{return b.q};func handler(r *http.Request,choose bool){b:=&Builder{};b.Set(r.FormValue("q"));b.Clear(choose);db.Exec(b.Query())}`, 0},
		{"builder alias", `type Builder struct{q string};func(b *Builder)Set(v string){b.q=v};func(b *Builder)Query()string{return b.q};func handler(r *http.Request){b:=&Builder{};alias:=b;alias.Set(r.FormValue("q"));db.Exec(b.Query())}`, 1},
		{"distinct factory calls", `type Builder struct{q string};func New()*Builder{return &Builder{}};func(b *Builder)Set(v string){b.q=v};func(b *Builder)Query()string{return b.q};func handler(r *http.Request){a,b:=New(),New();a.Set(r.FormValue("q"));db.Exec(b.Query())}`, 0},
		{"field swap", `type Builder struct{q,other string};func(b *Builder)Set(v string){b.q=v};func(b *Builder)Swap(){b.q,b.other=b.other,b.q};func(b *Builder)Query()string{return b.other};func handler(r *http.Request){b:=&Builder{};b.Set(r.FormValue("q"));b.Swap();db.Exec(b.Query())}`, 1},
		{"other field stays clean", `type Builder struct{q,other string};func(b *Builder)Set(v string){b.q=v};func(b *Builder)Query()string{return b.other};func handler(r *http.Request){b:=&Builder{};b.Set(r.FormValue("q"));db.Exec(b.Query())}`, 0},
		{"repeated allocation retains old object", `type Builder struct{q string};func(b *Builder)Set(v string){b.q=v};func(b *Builder)Query()string{return b.q};func handler(r *http.Request){var old *Builder;for i:=0;i<2;i++{b:=&Builder{};if i==0{old=b;b.Set(r.FormValue("q"))}else{b.Set("SELECT 1")}};db.Exec(old.Query())}`, 1},
		{"nested receiver", `type Builder struct{q string;child *Builder};func(b *Builder)Set(v string){b.q=v};func(b *Builder)Child()*Builder{return b.child};func(b *Builder)Query()string{return b.child.q};func handler(r *http.Request){b:=&Builder{};b.child=&Builder{};b.Child().Set(r.FormValue("q"));db.Exec(b.Query())}`, 1},
		{"fluent builder", `type Builder struct{q string};func(b *Builder)Set(v string)*Builder{b.q=v;return b};func(b *Builder)Query()string{return b.q};func handler(r *http.Request){b:=&Builder{};db.Exec(b.Set(r.FormValue("q")).Query())}`, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flows := sourceTaintFlows(t, tc.source)
			if len(flows) != tc.want {
				t.Fatalf("got %+v, want %d", flows, tc.want)
			}
		})
	}
}

func TestTaintSummariesDoNotMixCallers(t *testing.T) {
	flows := sourceTaintFlows(t, `func query(v string)string{return v};func unsafe(r *http.Request){db.Exec(query(r.FormValue("q")))};func safe(r *http.Request){db.Exec(query("SELECT 1"))}`)
	if len(flows) != 1 || flows[0].FuncID != "example.unsafe" {
		t.Fatalf("caller contamination: %+v", flows)
	}
}
func TestTaintSummariesAcrossPackages(t *testing.T) {
	flows := moduleTaintFlows(t, map[string]string{
		"go.mod":           "module example\n\ngo 1.26.0\n",
		"helper/helper.go": `package helper;import "net/http";func Input(r *http.Request)string{return r.FormValue("q")};func Forward(v string)string{return v}`,
		"p.go":             `package example;import("database/sql";"net/http";"example/helper");func handler(r *http.Request,db *sql.DB){db.Exec(helper.Forward(helper.Input(r)))}`,
	})
	if len(flows) != 1 || flows[0].SourceFuncID != "example/helper.Input" || flows[0].FuncID != "example.handler" {
		t.Fatalf("cross-package flow: %+v", flows)
	}
}
func TestTaintBuilderAgainstTwoObjectOracle(t *testing.T) {
	var source strings.Builder
	source.WriteString(`type Builder struct{q string};func(b *Builder)Set(v string){b.q=v};func(b *Builder)Query()string{return b.q};`)
	expected := map[string]bool{}
	for sequence := 0; sequence < 256; sequence++ {
		for target := 0; target < 2; target++ {
			name := fmt.Sprintf("case%d_%d", sequence, target)
			fmt.Fprintf(&source, "func %s(r *http.Request){a,b:=&Builder{},&Builder{};_=a;_=b;", name)
			memory := [2]bool{}
			remaining := sequence
			for step := 0; step < 4; step++ {
				action := remaining % 4
				remaining /= 4
				object := action / 2
				tainted := action%2 == 1
				memory[object] = tainted
				expression := `"SELECT 1"`
				if tainted {
					expression = `r.FormValue("q")`
				}
				fmt.Fprintf(&source, "%s.Set(%s);", []string{"a", "b"}[object], expression)
			}
			fmt.Fprintf(&source, "db.Exec(%s.Query())};", []string{"a", "b"}[target])
			expected["example."+name] = memory[target]
		}
	}
	observed := map[string]bool{}
	for _, flow := range sourceTaintFlows(t, source.String()) {
		observed[flow.FuncID] = true
	}
	for name, want := range expected {
		if observed[name] != want {
			t.Errorf("%s: got %v want %v", name, observed[name], want)
		}
	}
}

func TestTaintRecursiveFieldWideningRetainsDeepSource(t *testing.T) {
	var source strings.Builder
	source.WriteString(`type Builder struct{q string;next *Builder};func walk(b *Builder,stop bool)string{if stop{return b.q};return walk(b.next,stop)};func handler(r *http.Request,stop bool){root:=&Builder{};node:=root;`)
	for i := 0; i < 12; i++ {
		source.WriteString(`node.next=&Builder{};node=node.next;`)
	}
	source.WriteString(`node.q=r.FormValue("q");db.Exec(walk(root,stop))}`)
	if flows := sourceTaintFlows(t, source.String()); len(flows) != 1 {
		t.Fatalf("deep source lost: %+v", flows)
	}
}
