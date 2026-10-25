package goquality

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/rstyled/gstyle/domain/quality"
)

func TestNewViewRuntime(t *testing.T) {
	for _, content := range []string{"", "plain", "\x00\xff", "\x1b[31mstyled\x1b[0m", "日本語", strings.Repeat("x", 4096)} {
		view := tea.NewView(content)
		if !reflect.DeepEqual(view, tea.View{Content: content}) {
			t.Fatalf("constructor changes more than local content: %+v", view)
		}
		other := tea.NewView(content)
		view.Content = "changed"
		if other.Content != content {
			t.Fatal("constructed values share writable content")
		}
	}
}

func TestNewViewCurrentSource(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"application/view.go", "application/bugreducer/dashboard_view.go"} {
		facts, err := (Analyzer{Root: root}).Extract(context.Background(), []string{path})
		if err != nil {
			t.Fatal(err)
		}
		var calls int
		for _, function := range facts {
			for _, call := range function.Calls {
				if call.Callee == "charm.land/bubbletea/v2.NewView" {
					calls++
					t.Logf("%s raw call=%+v writes=%+v", function.Name, call, function.Mutations)
					if call.Dynamic || call.Local {
						t.Errorf("expected a concrete external constructor: %+v", call)
					}
					want := 5
					if path == "application/bugreducer/dashboard_view.go" {
						want = 3
					}
					if len(function.Mutations) != want {
						t.Errorf("lost construction writes: %+v", function.Mutations)
					}
					for _, write := range function.Mutations {
						if write.Provenance != quality.Local {
							t.Errorf("local view construction loses ownership: %+v", write)
						}
					}
				}
			}
		}
		if calls != 1 {
			t.Fatalf("%s: expected one concrete NewView call, got %d", path, calls)
		}
		report, err := quality.Evaluate(facts, quality.DefaultConfig())
		if err != nil {
			t.Fatal(err)
		}
		for _, function := range report.Functions {
			for _, effect := range function.Effects {
				if strings.Contains(effect.Detail, "charm.land/bubbletea/v2.NewView") {
					t.Errorf("local view construction retains false effect: %+v", effect)
				}
			}
		}
	}
}

func TestNewViewBoundaries(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import tea "charm.land/bubbletea/v2"
func Build(s string)tea.View{return tea.NewView(s)}
func Forward(s string)tea.View{return Build(s)}
func Local(s string)tea.View{v:=tea.NewView(s);v.AltScreen=true;return v}
func Shared(v *tea.View){_ =tea.NewView("local");v.Content="changed"}
func Alias(v *tea.View){alias:=v;_ =tea.NewView("local");alias.WindowTitle="changed"}
var saved tea.View
var count int
func Publish(){saved=tea.NewView("saved")}
func globalText()string{count++;return "global"}
func GlobalArgument()tea.View{return tea.NewView(globalText())}
func ioText()string{println("retained");return "io"}
func IOArgument()tea.View{return tea.NewView(ioText())}
func panicText()string{panic("retained")}
func PanicArgument()tea.View{return tea.NewView(panicText())}
func DynamicArgument(text func()string)tea.View{return tea.NewView(text())}
func Callback(run func()){tea.NewView("local");run()}
func Indirect(){build:=tea.NewView;build("indirect")}
func Program(){tea.NewProgram(nil)}
func Set(v *tea.View){v.SetContent("changed")}
func Custom(){NewView("custom")}
func NewView(string){count++}
`))
	module, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	module = []byte(strings.Replace(string(module), "module github.com/rstyled/gstyle", "module example.com/quality-repro", 1))
	if err := os.WriteFile(filepath.Join(root, "go.mod"), module, 0600); err != nil {
		t.Fatal(err)
	}
	sums, err := os.ReadFile("../../go.sum")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.sum"), sums, 0600); err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Build", "Forward", "Local"} {
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s retains false construction effects: %+v", name, result)
		}
	}
	for name, kind := range map[string]quality.Kind{"Shared": quality.MutationEffect, "Alias": quality.MutationEffect, "Publish": quality.Global, "GlobalArgument": quality.Global, "IOArgument": quality.IO, "PanicArgument": quality.Panic, "DynamicArgument": quality.UnknownEffect, "Callback": quality.UnknownEffect, "Indirect": quality.UnknownEffect, "Program": quality.UnknownEffect, "Set": quality.UnknownEffect, "Custom": quality.Global} {
		result := resultNamed(t, report, name)
		t.Logf("%s raw=%+v result=%+v", name, findFunction(t, facts, name).Mutations, result)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("%s loses %s: %+v", name, kind, result)
		}
	}
	for _, name := range []string{"Shared", "Alias"} {
		writes := findFunction(t, facts, name).Mutations
		if len(writes) != 1 || writes[0].Provenance != quality.External {
			t.Errorf("%s loses raw external write: %+v", name, writes)
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
		t.Fatal("saved view facts change evaluation", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.Global, quality.UnknownEffect} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "charm.land/bubbletea/v2.NewView", Kind: kind})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"Build", "Forward", "Shared"} {
			result := resultNamed(t, overridden, name)
			if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
				t.Errorf("%s loses %s saved-fact policy", name, kind)
			}
		}
	}
}
