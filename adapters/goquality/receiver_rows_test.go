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

func TestClosedReceiverRowsCurrentSource(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), []string{"domain/quality/helper_purity.go"})
	if err != nil {
		t.Fatal(err)
	}
	function := findFunction(t, facts, "index")
	rows := 0
	for _, write := range function.Mutations {
		if write.Root == "proof" && write.FieldPath == "callers" {
			rows++
			if write.Provenance != quality.Local {
				t.Errorf("current index loses private callers row: %+v", write)
			}
		}
	}
	if rows != 2 {
		t.Fatalf("want both raw row writes, got %d", rows)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index", "readOnlyHelpers"} {
		result := resultNamed(t, report, name)
		if slices.ContainsFunc(result.Effects, func(e quality.Effect) bool {
			return e.Kind == quality.UnknownEffect && e.Line == 31 || e.Detail == "github.com/rstyled/gstyle/domain/quality.helperPurity.index → proof.callers"
		}) {
			t.Errorf("%s retains false callers row effect: %+v", name, result)
		}
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.MutationEffect }) {
			t.Errorf("%s loses separate pure map mutation", name)
		}
	}
}

func TestClosedReceiverRowsBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, seed, before, after, extra string
		local                            bool
	}{
		{"empty", `map[int][]int{}`, ``, ``, ``, true},
		{"allocated", `make(map[int][]int)`, ``, ``, ``, true},
		{"zero", `nil`, `p.rows=make(map[int][]int)`, ``, ``, true},
		{"fresh row", `map[int][]int{}`, `p.rows[0]=[]int{1}`, ``, ``, true},
		{"allocated row", `map[int][]int{}`, `p.rows[0]=make([]int,1)`, ``, ``, true},
		{"nil row", `map[int][]int{}`, `p.rows[0]=nil`, ``, ``, true},
		{"observation", `map[int][]int{}`, `_ = len(p.rows);_ = cap(p.rows[0]);if p.rows[0]==nil{}`, ``, ``, true},
		{"copy elements", `map[int][]int{}`, ``, `_ = append(input,p.rows[0]...)`, ``, true},
		{"seeded", `map[int][]int{0:input}`, ``, ``, ``, false},
		{"borrowed outer", `seed`, ``, ``, ``, false},
		{"injected", `map[int][]int{}`, `p.rows[0]=input`, ``, ``, false},
		{"rebound", `map[int][]int{}`, `p.rows=seed`, ``, ``, false},
		{"seeded rebound", `map[int][]int{}`, `p.rows=map[int][]int{0:input}`, ``, ``, false},
		{"outer alias", `map[int][]int{}`, `alias:=p.rows;_ = alias`, ``, ``, false},
		{"row alias", `map[int][]int{}`, `alias:=p.rows[0];_ = alias`, ``, ``, false},
		{"row return", `map[int][]int{}`, ``, ``, `func(p *state)Row()[]int{return p.rows[0]}`, false},
		{"outer return", `map[int][]int{}`, ``, ``, `func(p *state)Rows()map[int][]int{return p.rows}`, false},
		{"outer publication", `map[int][]int{}`, `shared=p.rows`, ``, ``, false},
		{"row publication", `map[int][]int{}`, `saved=p.rows[0]`, ``, ``, false},
		{"row callback", `map[int][]int{}`, `callback(p.rows[0])`, ``, ``, false},
		{"root callback", `map[int][]int{}`, `callback(p)`, ``, ``, false},
		{"outer address", `map[int][]int{}`, `callback(&p.rows)`, ``, ``, false},
		{"element address", `map[int][]int{}`, `callback(&p.rows[0][0])`, ``, ``, false},
		{"row range alias", `map[int][]int{}`, `for _,row:=range p.rows{saved=row}`, ``, ``, false},
		{"capture", `map[int][]int{}`, `closure=func(){saved=p.rows[0]}`, ``, ``, false},
		{"defer", `map[int][]int{}`, `defer func(){saved=p.rows[0]}()`, ``, ``, false},
		{"async", `map[int][]int{}`, `go func(){saved=p.rows[0]}()`, ``, ``, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `package repro
type state struct{rows map[int][]int}
var shared map[int][]int
var saved []int
var closure func()
func(p *state)Step(input []int,seed map[int][]int,callback func(any)){
` + tc.before + `
p.rows[0]=append(p.rows[0],1)
` + tc.after + `
}
func Run(input []int,seed map[int][]int,callback func(any)){p:=state{rows:` + tc.seed + `};p.Step(input,seed,callback)}
` + tc.extra
			root := qualitySourceModule(t, []byte(source))
			facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			writes := findFunction(t, facts, "Step").Mutations
			var rowWrite quality.Mutation
			for _, write := range writes {
				if write.FieldPath == "rows" {
					rowWrite = write
				}
			}
			if rowWrite.Root != "p" || (rowWrite.Provenance == quality.Local) != tc.local {
				t.Errorf("local=%t, final row storage write=%+v; all=%+v", tc.local, rowWrite, writes)
			}
		})
	}
}

func TestClosedReceiverRowsEffectsAndReplay(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
type state struct{rows map[int][]*int}
func(p *state)Step(input *int){p.rows[0]=append(p.rows[0],input)}
func Run(input *int){p:=state{rows:map[int][]*int{}};p.Step(input)}
type child struct{rows map[int][]*int}
func(p *child)Write(input *int){p.rows[0]=append(p.rows[0],input);*p.rows[0][0]=2}
func Child(input *int){p:=child{rows:map[int][]*int{}};p.Write(input)}
type noisy struct{rows map[int][]int}
func(p *noisy)Step(callback func()){p.rows[0]=append(p.rows[0],1);println("step");callback()}
func Noisy(callback func()){p:=noisy{rows:map[int][]int{}};p.Step(callback)}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if result := resultNamed(t, report, "Run"); !result.Complete || len(result.Effects) != 0 {
		t.Errorf("private row append retains false effect: %+v", result)
	}
	writes := findFunction(t, facts, "Write").Mutations
	if writes[len(writes)-1].Provenance == quality.Local || len(resultNamed(t, report, "Child").Effects) == 0 {
		t.Errorf("row proof hides borrowed child write: %+v", writes)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.UnknownEffect} {
		if !slices.ContainsFunc(resultNamed(t, report, "Noisy").Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("row proof hides independent %s", kind)
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
		t.Fatal("row facts change after JSON replay", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.MutationEffect, quality.UnknownEffect} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.state.Step", Kind: kind})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(resultNamed(t, overridden, "Run").Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("row proof hides explicit %s policy", kind)
		}
	}
}

func TestClosedReceiverRowsLoadedTestVariant(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
type state struct{rows map[int][]int}
func(p *state)Step(){p.rows[0]=append(p.rows[0],1)}
func Run(){p:=state{rows:map[int][]int{}};p.Step()}
`))
	if err := os.WriteFile(filepath.Join(root, "input_test.go"), []byte(`package repro
var shared []int
func(p *state)Publish(){shared=p.rows[0]}
`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tests := range []bool{false, true} {
		facts, err := (Analyzer{Root: root, Tests: tests}).Extract(context.Background(), []string{"input.go"})
		if err != nil {
			t.Fatal(err)
		}
		writes := findFunction(t, facts, "Step").Mutations
		if (writes[len(writes)-1].Provenance == quality.Local) == tests {
			t.Errorf("tests=%t, publication boundary lost: %+v", tests, writes)
		}
	}
}

func TestClosedReceiverRowsShadowedAppendKeepsUnknown(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
type state struct{rows map[int][]int}
func(p *state)Step(input []int){
 append:=func([]int,...int)[]int{return input}
 p.rows[0]=append(p.rows[0],1)
}
func Run(input []int){p:=state{rows:map[int][]int{}};p.Step(input)}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(resultNamed(t, report, "Run").Effects, func(e quality.Effect) bool { return e.Kind == quality.UnknownEffect }) {
		t.Fatal("shadowed append loses its unknown call effect")
	}
}
