package gopatterns

import (
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

const sortTaintSourcePrefix = `package example
import("database/sql";"net/http";"sort")
var db *sql.DB
var _ = sort.Slice
`

func TestTaintSortPureComparators(t *testing.T) {
	for _, method := range []string{"Slice", "SliceStable"} {
		for _, tc := range []struct {
			name, body string
			want       int
		}{
			{"captured taint survives", `func handler(r *http.Request){q:=r.FormValue("q");SORT([]int{1,0},func(i,j int)bool{return len(q)+i<len(q)+j});db.Exec(q)}`, 1},
			{"clean capture stays separate", `func handler(r *http.Request){q:="SELECT 1";input:=r.FormValue("q");SORT([]int{1,0},func(i,j int)bool{_=q;_=input;return i<j});db.Exec(q)}`, 0},
			{"transitive reader", `func size(q string)int{return len(q)};func handler(r *http.Request){q:="SELECT 1";input:=r.FormValue("q");SORT([]int{1,0},func(i,j int)bool{return size(q)+size(input)+i<j});db.Exec(q)}`, 0},
			{"local comparator binding", `func handler(r *http.Request){q:="SELECT 1";input:=r.FormValue("q");less:=func(i,j int)bool{_=q;_=input;return i<j};SORT([]int{1,0},less);db.Exec(q)}`, 0},
			{"all alternatives read only", `func handler(r *http.Request,choose bool){q:="SELECT 1";input:=r.FormValue("q");less:=func(i,j int)bool{_=q;_=input;return i<j};if choose{less=func(i,j int)bool{_=q;_=input;return j<i}};SORT([]int{1,0},less);db.Exec(q)}`, 0},
			{"overwrite captured cell", `func handler(r *http.Request){q:=r.FormValue("q");SORT([]int{1,0},func(i,j int)bool{return len(q)+i<j});q="SELECT 1";db.Exec(q)}`, 0},
			{"later source reaches cell", `func handler(r *http.Request){q:="SELECT 1";SORT([]int{1,0},func(i,j int)bool{return len(q)+i<j});q=r.FormValue("q");db.Exec(q)}`, 1},
			{"shared reader closure stays clean", `func handler(r *http.Request){q:="SELECT 1";input:=r.FormValue("q");read:=func()string{return q};SORT([]int{1,0},func(i,j int)bool{_=q;_=input;return i<j});db.Exec(read())}`, 0},
			{"shared reader closure keeps taint", `func handler(r *http.Request){q:=r.FormValue("q");read:=func()string{return q};SORT([]int{1,0},func(i,j int)bool{return len(q)+i<j});db.Exec(read())}`, 1},
			{"shared reader observes overwrite", `func handler(r *http.Request){q:=r.FormValue("q");read:=func()string{return q};SORT([]int{1,0},func(i,j int)bool{return len(q)+i<j});q="SELECT 1";db.Exec(read())}`, 0},
			{"caller input remains tainted", `func ordered(q string)string{SORT([]int{1,0},func(i,j int)bool{return len(q)+i<j});return q};func handler(r *http.Request){db.Exec(ordered(r.FormValue("q")))}`, 1},
			{"caller inputs stay separate", `func ordered(q,input string)string{SORT([]int{1,0},func(i,j int)bool{_=q;_=input;return i<j});return q};func handler(r *http.Request){db.Exec(ordered("SELECT 1",r.FormValue("q")))}`, 0},
		} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				source := sortTaintSourcePrefix + strings.ReplaceAll(tc.body, "SORT", "sort."+method)
				assertSortTaintFlows(t, extractedSortTaintFlows(t, source), tc.want)
			})
		}
	}
}

func TestTaintSortWritersStayConservative(t *testing.T) {
	for _, method := range []string{"Slice", "SliceStable"} {
		for _, tc := range []struct {
			name, body string
		}{
			{"direct captured writer", `func handler(r *http.Request){q:="SELECT 1";input:=r.FormValue("q");SORT([]int{1,0},func(i,j int)bool{q=input;return i<j});db.Exec(q)}`},
			{"transitive pointer writer", `func write(q *string,input string){*q=input};func relay(q *string,input string){write(q,input)};func handler(r *http.Request){q:="SELECT 1";input:=r.FormValue("q");SORT([]int{1,0},func(i,j int)bool{relay(&q,input);return i<j});db.Exec(q)}`},
			{"unknown writer", `func handler(r *http.Request,write func(*string,string)){q:="SELECT 1";input:=r.FormValue("q");SORT([]int{1,0},func(i,j int)bool{write(&q,input);return i<j});db.Exec(q)}`},
			{"deferred captured writer", `func handler(r *http.Request){q:="SELECT 1";input:=r.FormValue("q");SORT([]int{1,0},func(i,j int)bool{defer func(){q=input}();return i<j});db.Exec(q)}`},
			{"ambiguous reader and writer", `func handler(r *http.Request,choose bool){q:="SELECT 1";input:=r.FormValue("q");less:=func(i,j int)bool{_=q;_=input;return i<j};if choose{less=func(i,j int)bool{q=input;return i<j}};SORT([]int{1,0},less);db.Exec(q)}`},
			{"ambiguous reader and unknown", `func handler(r *http.Request,choose bool,less func(int,int)bool){q:="SELECT 1";input:=r.FormValue("q");if choose{less=func(i,j int)bool{_=q;_=input;return i<j}};SORT([]int{1,0},less);db.Exec(q)}`},
			{"writer cannot prove overwrite", `func handler(r *http.Request){q:=r.FormValue("q");SORT([]int{1,0},func(i,j int)bool{q="SELECT 1";return i<j});db.Exec(q)}`},
		} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				source := sortTaintSourcePrefix + strings.ReplaceAll(tc.body, "SORT", "sort."+method)
				assertSortTaintFlows(t, extractedSortTaintFlows(t, source), 1)
			})
		}
	}
}

func TestTaintSortFirstArgumentKeepsEnvironmentAliases(t *testing.T) {
	for _, method := range []string{"Slice", "SliceStable"} {
		for _, tc := range []struct {
			name, body string
		}{
			{"callback also occupies first argument", `func handler(r *http.Request){q:="SELECT 1";input:=r.FormValue("q");less:=func(i,j int)bool{_=q;_=input;return i<j};SORT(less,less);db.Exec(q)}`},
			{"interface alias occupies first argument", `func handler(r *http.Request){q:="SELECT 1";input:=r.FormValue("q");less:=func(i,j int)bool{_=q;_=input;return i<j};var values any=less;SORT(values,less);db.Exec(q)}`},
		} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				source := sortTaintSourcePrefix + strings.ReplaceAll(tc.body, "SORT", "sort."+method)
				assertSortTaintFlows(t, extractedSortTaintFlows(t, source), 1)
			})
		}
	}
}

func TestTaintNonSortReadOnlyCallbacksStayConservative(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"pointer result", `func handler(r *http.Request,opaque func(func()*string,string)){q:="SELECT 1";opaque(func()*string{return &q},r.FormValue("q"));db.Exec(q)}`, 1},
		{"panic payload", `func handler(r *http.Request,opaque func(func(),string)){q:="SELECT 1";opaque(func(){panic(&q)},r.FormValue("q"));db.Exec(q)}`, 1},
		{"pointer result without source", `func handler(r *http.Request,opaque func(func()*string,string)){q:="SELECT 1";opaque(func()*string{return &q},"SELECT 2");db.Exec(q)}`, 0},
		{"panic payload without source", `func handler(r *http.Request,opaque func(func(),string)){q:="SELECT 1";opaque(func(){panic(&q)},"SELECT 2");db.Exec(q)}`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertSortTaintFlows(t, sourceTaintFlows(t, tc.body), tc.want)
		})
	}
}

func TestTaintSortUsesResolvedIdentity(t *testing.T) {
	for _, method := range []string{"Slice", "SliceStable"} {
		for _, tc := range []struct {
			name, source string
			want         int
		}{
			{"renamed import", `package example;import("database/sql";"net/http";sorting "sort");var db *sql.DB;func handler(r *http.Request){q:="SELECT 1";input:=r.FormValue("q");sorting.METHOD([]int{1,0},func(i,j int)bool{_=q;_=input;return i<j});db.Exec(q)}`, 0},
			{"shadowed selector", sortTaintSourcePrefix + `func handler(r *http.Request,opaque func(any,func(int,int)bool)){q:="SELECT 1";sort:=struct{METHOD func(any,func(int,int)bool)}{opaque};sort.METHOD(r.FormValue("q"),func(i,j int)bool{if i<j{panic(&q)};return false});db.Exec(q)}`, 1},
			{"local lookalike method", sortTaintSourcePrefix + `type sorter struct{};var opaque func(any,func(int,int)bool);func(sorter)METHOD(values any,less func(int,int)bool){opaque(values,less)};func handler(r *http.Request){q:="SELECT 1";sort:=sorter{};sort.METHOD(r.FormValue("q"),func(i,j int)bool{if i<j{panic(&q)};return false});db.Exec(q)}`, 1},
			{"local lookalike function", sortTaintSourcePrefix + `var opaque func(any,func(int,int)bool);func METHOD(values any,less func(int,int)bool){opaque(values,less)};func handler(r *http.Request){q:="SELECT 1";METHOD(r.FormValue("q"),func(i,j int)bool{if i<j{panic(&q)};return false});db.Exec(q)}`, 1},
		} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				assertSortTaintFlows(t, extractedSortTaintFlows(t, strings.ReplaceAll(tc.source, "METHOD", method)), tc.want)
			})
		}
	}
}

func extractedSortTaintFlows(t *testing.T, source string) []patterns.TaintFlow {
	t.Helper()
	extraction, _ := proofExtraction(t, source)
	var facts []*patterns.MiningFacts
	for _, graph := range extraction.Callables {
		name := graph.Tables.Text(graph.Function.ID)
		facts = append(facts, &patterns.MiningFacts{ID: name, Name: name, Canonical: graph})
	}
	flows, err := patterns.QueryTaintFlows(facts)
	if err != nil {
		t.Fatal(err)
	}
	return flows
}

func assertSortTaintFlows(t *testing.T, flows []patterns.TaintFlow, want int) {
	t.Helper()
	if len(flows) != want {
		t.Fatalf("got %+v, want %d flows", flows, want)
	}
	for _, flow := range flows {
		if flow.FuncID != "example.handler" || flow.SourceFuncID != "example.handler" {
			t.Fatalf("unexpected flow endpoints: %+v", flow)
		}
		if flow.SourceCall != "net/http.Request.FormValue" || flow.SinkCall != "database/sql.DB.Exec" {
			t.Fatalf("unexpected flow calls: %+v", flow)
		}
	}
}
