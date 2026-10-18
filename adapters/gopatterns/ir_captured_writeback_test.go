package gopatterns

import (
	"context"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestCapturedWritesComposeNormalState(t *testing.T) {
	for _, tc := range []struct {
		name, body, helper string
		want               callflow.Outcomes
	}{
		{"unknown entry overwritten", `flag:=choose;func(){flag=true}();if flag{panic("certain")}`, "", callflow.Panics},
		{"true store", `flag:=false;func(){flag=true}();if flag{panic("certain")}`, "", callflow.Panics},
		{"false store", `flag:=true;func(){flag=false}();if flag{panic("impossible")}`, "", callflow.Returns},
		{"unordered sibling read", `flag:=false;value:=func()bool{flag=true;return false};if flag==value(){panic("maybe")}`, "", callflow.Returns | callflow.Panics},
		{"reverse unordered read", `flag:=false;value:=func()bool{flag=true;return false};if value()==flag{panic("maybe")}`, "", callflow.Returns | callflow.Panics},
		{"required argument snapshot", `flag:=false;result:=func(before bool)bool{flag=true;return before}(flag);if result||!flag{panic("bad")}`, "", callflow.Returns},
		{"stored writer", `flag:=false;write:=func(){flag=true};write();if flag{panic("certain")}`, "", callflow.Panics},
		{"nested relay", `flag:=false;func(){func(){flag=true}()}();if flag{panic("certain")}`, "", callflow.Panics},
		{"selected argument", `flag:=false;func(set bool){if set{flag=true}}(true);if flag{panic("certain")}`, "", callflow.Panics},
		{"unselected argument", `flag:=false;func(set bool){if set{flag=true}}(false);if flag{panic("bad")}`, "", callflow.Returns},
		{"multiple cells", `a,b:=false,true;func(){a,b=b,a}();if !a||b{panic("bad")}`, "", callflow.Returns},
		{"nil store", `value:=0;p:=&value;func(){p=nil}();if p!=nil{panic("bad")}`, "", callflow.Returns},
		{"non nil store", `var p *int;value:=0;func(){p=&value}();if p==nil{panic("bad")}`, "", callflow.Returns},
		{"boxed nil store", `var box any;var p *int;func(){box=p}();if box==nil{panic("bad")}`, "", callflow.Returns},
		{"joint result and store", `flag:=false;if func()bool{flag=true;return false}(){panic("bad")};if !flag{panic("bad")}`, "", callflow.Returns},
		{"read after write in body", `flag:=false;func(){flag=true;func(){if !flag{panic("bad")}}()}();if !flag{panic("bad")}`, "", callflow.Returns},
		{"conflicting targets", `flag:=false;write:=func(){flag=true};if choose{write=func(){flag=false}};write();if flag{panic("maybe")}`, "", callflow.Returns | callflow.Panics},
		{"agreeing targets", `flag:=false;write:=func(){flag=true};if choose{write=func(){flag=true}};write();if flag{panic("certain")}`, "", callflow.Panics},
		{"preserving alternative", `flag:=false;write:=func(){flag=true};if choose{write=func(){}};write();if flag{panic("maybe")}`, "", callflow.Returns | callflow.Panics},
		{"nonreturning alternative", `flag:=false;write:=func(){flag=true;panic("certain")};if choose{write=func(){flag=false}};write();if flag{panic("bad")}`, "", callflow.Returns | callflow.Panics},
		{"unconfined global writer", `flag:=false;func(){global=true;flag=true}();if flag{panic("maybe")}`, "", callflow.Returns | callflow.Panics},
		{"global storage cannot be private", `global=false;func(){global=true}();if !global{panic("bad")}`, "", callflow.Returns | callflow.Panics},
		{"unconfined indirect writer", `flag:=false;func(){p:=&flag;*p=true}();if flag{panic("maybe")}`, "", callflow.Returns | callflow.Panics},
		{"deferred state remains unknown", `flag:=false;func(){defer func(){flag=false}();flag=true}();if flag{panic("maybe")}`, "", callflow.Returns | callflow.Panics},
		{"ordinary named output", `if !value(){panic("bad")}`, `func value()(flag bool){func(){flag=true}();return}`, callflow.Returns},
		{"outer parameter cell", `outer(false)`, `func outer(flag bool){func(){flag=true}();if flag{panic("certain")}}`, callflow.Panics},
		{"private nested cell", `flag:=false;helper();if flag{panic("bad")}`, `func helper(){local:=false;func(){local=true}();_=local}`, callflow.Returns},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;var choose,global bool;func f(){`+tc.body+`};`+tc.helper)
			graphs := map[string]*pdg.Graph{}
			for _, graph := range extraction.Callables {
				graphs[graph.Tables.Text(graph.Function.ID)] = graph
			}
			summaries, err := callflow.ExitEffects(graphs)
			if err != nil {
				t.Fatal(err)
			}
			if got := summaries["example.f"].Ordinary; got != tc.want {
				t.Fatalf("exit=%v want %v", got, tc.want)
			}
		})
	}
}

func TestCapturedWritebackRefinesSharedGraph(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		reachable  bool
	}{
		{"false postcondition", `flag:=true;func(){flag=false}();if flag{println("arm")}`, false},
		{"true postcondition", `flag:=false;func(){flag=true}();if !flag{println("arm")}`, false},
		{"mixed postcondition", `flag:=false;write:=func(){flag=true};if choose{write=func(){flag=false}};write();if flag{println("arm")}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;var choose bool;func f(){`+tc.body+`}`)
			graph := continuationGraph(t, extraction, "example.f")
			found, reachable := false, false
			for _, block := range graph.ValueFlow.Blocks {
				for _, ref := range block.Operations {
					attrs := graph.NodeAttributes(graph.Nodes[ref-1])
					if graph.Tables.Text(attrs.Callee.Value) != "println" {
						continue
					}
					found = true
					reachable = reachable || block.Reachable
					if !block.Reachable {
						assertNoResultExecutionEdges(t, graph, ref)
					}
				}
			}
			if !found {
				t.Fatal("missing source operation")
			}
			if reachable != tc.reachable {
				t.Fatalf("arm reachable=%v want %v", reachable, tc.reachable)
			}
		})
	}
}

func TestCapturedWritebackNeedsSingleCellEvidence(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example;func f(){flag:=true;func(){flag=false}();if flag{panic("maybe")}}`)
	graphs := map[string]*pdg.Graph{}
	for _, graph := range extraction.Callables {
		graphs[graph.Tables.Text(graph.Function.ID)] = graph
	}
	owner := *graphs["example.f"]
	owner.Symbols = slices.Clone(owner.Symbols)
	for i := range owner.Symbols {
		owner.Symbols[i].Extensions = slices.DeleteFunc(slices.Clone(owner.Symbols[i].Extensions), func(extension pdg.Extension) bool {
			return owner.Tables.Text(extension.Name) == "gstyle.memory:single-cell"
		})
	}
	graphs["example.f"] = &owner
	summaries, err := callflow.ExitEffects(graphs)
	if err != nil {
		t.Fatal(err)
	}
	if got := summaries["example.f"].Ordinary; got != callflow.Returns|callflow.Panics {
		t.Fatalf("missing cell proof asserts exit=%v", got)
	}
}

func TestCapturedWritebackAcrossCachedRevisions(t *testing.T) {
	root := crossPackageModule(t, `package stop;func Now(flag bool){}`)
	caller := filepath.Join(root, "caller.go")
	firstSource := `package example;func f(){flag:=false;func(){flag=true}();if flag{panic("certain")}}`
	secondSource := `package example;func f(){flag:=false;func(){flag=false}();if flag{panic("bad")}}`
	if err := os.WriteFile(caller, []byte(firstSource), 0644); err != nil {
		t.Fatal(err)
	}
	cache := &FixExtractor{}
	first, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{})
	if err != nil {
		t.Fatal(err)
	}
	checkMemoryRevision(t, first, callflow.Panics)
	files := map[string][]byte{caller: []byte(secondSource)}
	second, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{Files: files, Disk: first.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	checkMemoryRevision(t, second, callflow.Returns)
	files[caller] = []byte(firstSource)
	third, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{Files: files, Disk: second.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	checkMemoryRevision(t, third, callflow.Panics)
}
