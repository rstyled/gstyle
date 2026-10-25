package goquality

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestClosedReceiverLocalFieldCurrentSource(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), []string{"domain/quality/helper_purity.go"})
	if err != nil {
		t.Fatal(err)
	}
	function := findFunction(t, facts, "readOnlyHelpers")
	var pending int
	for _, write := range function.Mutations {
		if write.Root == "proof" && write.FieldPath == "pending" {
			pending++
			if write.Provenance != quality.Local {
				t.Errorf("current readOnlyHelpers loses closed pending storage: %+v", write)
			}
		}
	}
	if pending == 0 {
		t.Fatal("missing raw pending mutation")
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	result := resultNamed(t, report, "readOnlyHelpers")
	for _, effect := range result.Effects {
		if effect.Detail == "proof.pending" {
			t.Errorf("current source retains false pending effect: %+v", effect)
		}
	}
	// Returning pure from a receiver and owning map rows need separate proofs.
	for _, kind := range []quality.Kind{quality.MutationEffect, quality.UnknownEffect} {
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("current source loses remaining %s evidence", kind)
		}
	}
}

func TestClosedReceiverLocalFieldBoundaries(t *testing.T) {
	const step = `func(p *state)Step(){p.items=append(p.items,1)}`
	for _, tc := range []struct {
		name, method, body string
		local              bool
	}{
		{"value", step, `p:=state{items:[]int{0}};p.Step();p.items[0]=2`, true},
		{"pointer", step, `p:=&state{items:[]int{0}};p.Step();p.items[0]=2`, true},
		{"zero", step, `var p state;p.Step();p.items[0]=2`, true},
		{"new", step, `p:=new(state);p.Step();p.items[0]=2`, true},
		{"borrowed", step, `p:=state{items:input};p.Step();p.items[0]=2`, false},
		{"method replacement", `func(p *state)Step(v []int){p.items=v}`, `p:=state{items:[]int{0}};p.Step(input);p.items[0]=2`, false},
		{"method publication", `func(p *state)Step(){shared=p.items}`, `p:=state{items:[]int{0}};p.Step();p.items[0]=2`, false},
		{"method return", `func(p *state)Step()[]int{return p.items}`, `p:=state{items:[]int{0}};alias:=p.Step();alias[0]=1;p.items[0]=2`, false},
		{"field alias", step, `p:=state{items:[]int{0}};p.Step();alias:=p.items;shared=alias;p.items[0]=2`, false},
		{"root alias", step, `p:=&state{items:[]int{0}};alias:=p;alias.items=input;p.Step();p.items[0]=2`, false},
		{"field address", step, `p:=state{items:[]int{0}};slot:=&p.items;*slot=input;p.Step();p.items[0]=2`, false},
		{"root publication", step, `p:=&state{items:[]int{0}};published=p;p.Step();p.items[0]=2`, false},
		{"callback", step, `p:=&state{items:[]int{0}};callback(p);p.Step();p.items[0]=2`, false},
		{"capture", step, `p:=state{items:[]int{0}};saved=func(){p.items=input};p.Step();p.items[0]=2`, false},
		{"async", step, `p:=state{items:[]int{0}};go p.Step();p.items[0]=2`, false},
		{"deferred method", step, `p:=state{items:[]int{0}};defer p.Step();p.items[0]=2`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `package repro
type state struct{items []int}
var shared []int
var published any
var saved func()
` + tc.method + `
func Run(input []int,callback func(any)){` + tc.body + `}`
			root := qualitySourceModule(t, []byte(source))
			facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			writes := findFunction(t, facts, "Run").Mutations
			write := writes[len(writes)-1]
			if write.Root != "p" || write.FieldPath != "items" || (write.Provenance == quality.Local) != tc.local {
				t.Errorf("local=%t, raw write=%+v", tc.local, write)
			}
		})
	}
}

func TestClosedReceiverLocalFieldKeepsIndependentEffects(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
type state struct{items []int;borrowed map[string]int;children []*int}
func(p *state)Step(callback func()){p.items=append(p.items,1);p.borrowed["method"]=2;*p.children[0]=3;println("step");callback()}
func Run(input map[string]int,child *int,callback func()){
 p:=state{items:[]int{0},borrowed:input,children:[]*int{child}}
 p.Step(callback)
 p.items[0]=4
 p.borrowed["caller"]=5
 *p.children[0]=6
}
type clean struct{items []int}
func(p *clean)Step(){p.items=append(p.items,1)}
func PureRun(){p:=clean{items:[]int{0}};p.Step();p.items[0]=2}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	writes := findFunction(t, facts, "Run").Mutations
	if len(writes) != 3 || writes[0].Provenance != quality.Local || writes[1].Provenance == quality.Local || writes[2].Provenance == quality.Local {
		t.Fatalf("owned field hides borrowed or child writes: %+v", writes)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if result := resultNamed(t, report, "PureRun"); !result.Complete || len(result.Effects) != 0 {
		t.Errorf("closed local field retains false effects: %+v", result)
	}
	for _, kind := range []quality.Kind{quality.MutationEffect, quality.IO, quality.UnknownEffect} {
		if !slices.ContainsFunc(resultNamed(t, report, "Run").Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("local field hides independent %s effect", kind)
		}
	}
	data, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	var restored []quality.Function
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	replay, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil || !reflect.DeepEqual(report, replay) {
		t.Fatal("local closed field facts change after JSON replay", err)
	}
	for _, kind := range []quality.Kind{quality.MutationEffect, quality.IO, quality.UnknownEffect} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.clean.Step", Kind: kind})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(resultNamed(t, overridden, "PureRun").Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("local field hides explicit %s policy", kind)
		}
	}
}

func TestClosedReceiverLocalFieldTestVariant(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
type state struct{items []int}
func(p *state)Step(){p.items=append(p.items,1)}
func Run(){p:=state{items:[]int{0}};p.Step();p.items[0]=2}
`))
	if err := os.WriteFile(filepath.Join(root, "input_test.go"), []byte(`package repro
var shared []int
func(p *state)publish(){shared=p.items}
`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tests := range []bool{false, true} {
		facts, err := (Analyzer{Root: root, Tests: tests}).Extract(context.Background(), []string{"input.go"})
		if err != nil {
			t.Fatal(err)
		}
		writes := findFunction(t, facts, "Run").Mutations
		if len(writes) != 1 || (writes[0].Provenance == quality.Local) == tests {
			t.Errorf("tests=%t, local field writes=%+v", tests, writes)
		}
	}
}

func BenchmarkClosedReceiverLocalFieldExtraction(b *testing.B) {
	root := os.Getenv("GSTYLE_BENCH_ROOT")
	if root == "" {
		b.Skip("set GSTYLE_BENCH_ROOT to a fixed source corpus")
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := (Analyzer{Root: root}).Extract(context.Background(), []string{"domain/quality/helper_purity.go"}); err != nil {
			b.Fatal(err)
		}
	}
}
