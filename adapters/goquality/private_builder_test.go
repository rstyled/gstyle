package goquality

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestPrivateBuilderCurrentSource(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), []string{"examples/quality-hunt/corpus.go"})
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"program", "oracle"} {
		function := findFunction(t, facts, name)
		found := false
		for _, call := range function.Calls {
			found = found || call.Callee == "fmt.Fprintf" && call.MemoryWriter
		}
		if !found || len(function.Mutations) == 0 {
			t.Fatalf("%s loses raw call or write records: %+v", name, function)
		}
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s retains private builder effect: %+v", name, result)
		}
	}
}

func TestPrivateBuilderWrites(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("fmt";"io";"strings")
func Local()string{var out strings.Builder;out.Grow(10);out.WriteString("start");fmt.Fprintf(&out,"%d",1);io.WriteString(&out,"end");return out.String()}
func Loop(values []string)string{var out strings.Builder;for _,value:=range values{fmt.Fprintln(&out,value)};return out.String()}
func Forward()string{return Local()}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Local", "Loop", "Forward"} {
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s retains private builder effect: %+v", name, result)
		}
	}
}

func TestPrivateBuilderBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, body string
	}{
		{"borrowed", `fmt.Fprint(input,"x")`},
		{"value parameter", `fmt.Fprint(&copy,"x")`},
		{"global", `fmt.Fprint(&global,"x")`},
		{"alias", `var out strings.Builder;alias:=&out;fmt.Fprint(&out,"x");_ = alias`},
		{"replacement", `var out strings.Builder;out=copy;fmt.Fprint(&out,"x")`},
		{"address escape", `var out strings.Builder;published=&out;fmt.Fprint(&out,"x")`},
		{"value escape", `var out strings.Builder;published=out;fmt.Fprint(&out,"x")`},
		{"callback", `var out strings.Builder;callback(&out);fmt.Fprint(&out,"x")`},
		{"return", `var out strings.Builder;fmt.Fprint(&out,"x");return &out`},
		{"capture", `var out strings.Builder;saved=func(){out.Reset()};fmt.Fprint(&out,"x")`},
		{"nested write", `var out strings.Builder;func(){fmt.Fprint(&out,"x")}()`},
		{"go", `var out strings.Builder;go fmt.Fprint(&out,"x")`},
		{"defer", `var out strings.Builder;defer fmt.Fprint(&out,"x")`},
		{"deferred observation", `var out strings.Builder;defer out.Reset();fmt.Fprint(&out,"x")`},
		{"method value", `var out strings.Builder;write:=out.WriteString;_,_=write("x");fmt.Fprint(&out,"x")`},
		{"method expression", `var out strings.Builder;_,_=(*strings.Builder).WriteString(&out,"x");fmt.Fprint(&out,"x")`},
		{"unproved initializer", `out:=strings.Builder{};fmt.Fprint(&out,"x")`},
		{"formatter callback", `var out strings.Builder;fmt.Fprint(&out,custom(1))`},
		{"open value", `var out strings.Builder;fmt.Fprint(&out,value)`},
		{"spread", `var out strings.Builder;fmt.Fprint(&out,values...)`},
		{"borrowed buffer", `out:=bytes.NewBuffer(data[:0]);fmt.Fprint(out,"x")`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := qualitySourceModule(t, []byte(`package repro
import("bytes";"fmt";"strings")
var _ = bytes.NewBuffer
var global strings.Builder
var published any
var saved func()
type custom int
func(custom)String()string{println("callback");return "x"}
func Run(input *strings.Builder,copy strings.Builder,callback func(any),value any,values []any,data []byte)*strings.Builder{
`+tc.body+`
return nil
}`))
			facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			report, err := quality.Evaluate(facts, quality.DefaultConfig())
			if err != nil {
				t.Fatal(err)
			}
			result := resultNamed(t, report, "Run")
			if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool {
				return e.Detail == "fmt.Fprint" && (e.Kind == quality.MutationEffect || e.Kind == quality.IO)
			}) {
				t.Errorf("unsupported builder loses write evidence: %+v", result)
			}
		})
	}
}

func TestPrivateBuilderEffectsReplayAndPolicy(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("fmt";"io";"strings")
type Alias=strings.Builder
var counter int
func next()int{counter++;return counter}
func Run(input *int,callback func())string{
 var out Alias
 out.Write([]byte("a"));out.WriteByte('b');out.WriteRune('c');_ = out.Len();_ = out.Cap();out.Reset()
 fmt.Fprintf((&out),"%d",next());io.WriteString(&out,"x")
 *input=3;println("io");callback()
 return out.String()
}
func Forward(input *int,callback func())string{return Run(input,callback)}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Run", "Forward"} {
		result := resultNamed(t, report, name)
		for _, kind := range []quality.Kind{quality.Global, quality.IO, quality.UnknownEffect, quality.MutationEffect} {
			if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
				t.Errorf("%s loses independent %s: %+v", name, kind, result)
			}
		}
		if slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Detail == "fmt.Fprintf" || e.Detail == "io.WriteString" }) {
			t.Errorf("%s retains private writer effect: %+v", name, result)
		}
	}
	writes := findFunction(t, facts, "Run").Mutations
	if !slices.ContainsFunc(writes, func(m quality.Mutation) bool { return m.Root == "input" && m.Provenance == quality.External }) {
		t.Fatalf("missing raw borrowed write: %+v", writes)
	}
	var restored []quality.Function
	if err := json.Unmarshal(before, &restored); err != nil {
		t.Fatal(err)
	}
	replay, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil || !reflect.DeepEqual(report, replay) {
		t.Fatal("private builder replay changes report", err)
	}
	for _, callee := range []string{"fmt.Fprintf", "io.WriteString"} {
		for _, kind := range []quality.Kind{quality.IO, quality.MutationEffect, quality.UnknownEffect} {
			config := quality.DefaultConfig()
			config.Prefixes = append(config.Prefixes, quality.Prefix{Path: callee, Kind: kind})
			overridden, err := quality.Evaluate(restored, config)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.ContainsFunc(resultNamed(t, overridden, "Run").Effects, func(e quality.Effect) bool { return e.Detail == callee && e.Kind == kind }) {
				t.Errorf("private builder hides explicit %s policy for %s", kind, callee)
			}
		}
	}
	after, err := json.Marshal(facts)
	if err != nil || string(before) != string(after) {
		t.Fatal("evaluation changes immutable input facts", err)
	}
}
