package gopatterns

import (
	"context"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"os"
	"path/filepath"
	"testing"
)

func TestClosureBodiesComposeExitEffects(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       callflow.Outcomes
	}{
		{"deferred noop", `defer func(){}()`, callflow.Returns},
		{"immediate panic", `func(){panic("stop")}()`, callflow.Panics},
		{"deferred panic", `defer func(){panic("stop")}()`, callflow.Panics},
		{"deferred recovery", `defer func(){recover()}();panic("caught")`, callflow.Returns},
		{"ordinary recovery is nil", `func(){if recover()!=nil{panic("bad")}}()`, callflow.Returns},
		{"nested ordinary recovery", `defer func(){func(){recover()}()}();panic("uncaught")`, callflow.Panics},
		{"nested bodies", `func(){func(){panic("stop")}()}()`, callflow.Panics},
		{"stored value", `stop:=func(){panic("stop")};stop()`, callflow.Panics},
		{"stored copy", `stop:=func(){panic("stop")};copy:=stop;copy()`, callflow.Panics},
		{"argument", `func(flag bool){if flag{panic("stop")}}(true)`, callflow.Panics},
		{"deferred argument snapshot", `flag:=true;defer func(value bool){if value{panic("stop")}}(flag);flag=false;_=flag`, callflow.Panics},
		{"Boolean result", `if func()bool{return false}(){panic("bad")}`, callflow.Returns},
		{"tuple result", `a,b:=func()(bool,bool){return false,true}();if a||!b{panic("bad")}`, callflow.Returns},
		{"construction only", `stop:=func(){panic("not called")};_=stop`, callflow.Returns},
		{"constant captured input", `flag:=true;func(){if flag{panic("maybe")}}()`, callflow.Panics},
		{"captured write updates", `flag:=true;func(){flag=false}();if flag{panic("maybe")}`, callflow.Returns},
		{"target join", `stop:=func(){};if choose{stop=func(){panic("maybe")}};stop()`, callflow.Returns | callflow.Panics},
		{"recursive stored closure", `var again func();again=func(){again()};again()`, callflow.Diverges},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;var choose bool;func f(){`+tc.body+`}`)
			graphs := map[string]*pdg.Graph{}
			for _, graph := range extraction.Callables {
				if err := pdg.Validate(graph); err != nil {
					t.Fatal(err)
				}
				name := graph.Tables.Text(graph.Function.ID)
				if graphs[name] != nil {
					t.Fatal("duplicate body identity")
				}
				graphs[name] = graph
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

func TestClosureExitPrunesCanonicalContinuation(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example;func f(){func(){panic("stop")}();println("after")}`)
	graph := continuationGraph(t, extraction, "example.f")
	found := false
	for _, block := range graph.ValueFlow.Blocks {
		for _, ref := range block.Operations {
			attrs := graph.NodeAttributes(graph.Nodes[ref-1])
			if graph.Tables.Text(attrs.Callee.Value) != "println" {
				continue
			}
			found = true
			if block.Reachable {
				t.Fatal("nonreturning closure retains reachable continuation")
			}
			assertNoResultExecutionEdges(t, graph, ref)
		}
	}
	if !found {
		t.Fatal("missing continuation source operation")
	}
}

func TestClosureInventoryKeepsCompilerInfoAndDeclarationGraphs(t *testing.T) {
	pkg := loadTestPackage(t, `package example;func f(){func(){func(){}()}()}`)
	original := pkg.TypesInfo
	definitions := len(original.Defs)
	p := packageAnalysis{pkg: pkg, root: filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename), pool: pdg.NewBuilder()}
	if _, err := p.run(); err != nil {
		t.Fatal(err)
	}
	if pkg.TypesInfo != original || len(original.Defs) != definitions {
		t.Fatal("closure lowering mutates loader type information")
	}
	if len(p.graphs) != 3 || len(p.result) != 1 {
		t.Fatalf("callables=%d declaration reports=%d", len(p.graphs), len(p.result))
	}
	source := p.graphs[0].graph
	if &source.Nodes[0] != &p.result[0].Pdg.Nodes[0] {
		t.Fatal("declaration report replaces canonical storage")
	}
	for _, body := range p.graphs {
		if body.inventory != nil {
			t.Fatal("compiler graph inventory is retained after publication")
		}
	}
}

func TestLoadedClosureBodiesComposeAcrossPackages(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":     "module example\n\ngo 1.26\n",
		"lib/lib.go": `package lib;func Stop(){func(){panic("stop")}()}`,
		"main.go":    `package example;import "example/lib";func f(){lib.Stop();println("after")}`,
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	extraction, err := Extract(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	graphs := map[string]*pdg.Graph{}
	for _, graph := range extraction.Callables {
		graphs[graph.Tables.Text(graph.Function.ID)] = graph
	}
	summaries, err := callflow.ExitEffects(graphs)
	if err != nil {
		t.Fatal(err)
	}
	if summaries["example.f"].Ordinary != callflow.Panics {
		t.Fatalf("loaded closure exit=%v", summaries["example.f"].Ordinary)
	}
	graph := continuationGraph(t, extraction, "example.f")
	for _, block := range graph.ValueFlow.Blocks {
		for _, ref := range block.Operations {
			attrs := graph.NodeAttributes(graph.Nodes[ref-1])
			if graph.Tables.Text(attrs.Callee.Value) == "println" && block.Reachable {
				t.Fatal("loaded closure retains caller continuation")
			}
		}
	}
}

func TestClosureCatalogAcrossExtractionModes(t *testing.T) {
	for _, mode := range []string{"normal", "bounded", "cached"} {
		t.Run(mode, func(t *testing.T) {
			root := crossPackageModule(t, `package stop;func Now(flag bool){func(value bool){if value{panic("stop")}}(flag)}`)
			var extraction *Extraction
			var err error
			switch mode {
			case "normal":
				extraction, err = Extract(context.Background(), root, []string{"."})
			case "bounded":
				extraction, err = ExtractForFix(context.Background(), root, []string{"."})
			default:
				extraction, err = (&FixExtractor{}).Extract(context.Background(), root, []string{"."}, SourceOverlay{})
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(extraction.Callables) <= len(extraction.Funcs) {
				t.Fatal("lexical closure catalog is missing")
			}
			if afterCallReachable(t, extraction, "example.f") || !afterCallReachable(t, extraction, "example.g") {
				t.Fatal("loaded closure loses its argument-specific continuation")
			}
		})
	}
}

func TestClosureCatalogSurvivesCallerAndDependencyRevisions(t *testing.T) {
	root := crossPackageModule(t, `package stop;func Now(flag bool){func(value bool){if value{panic("stop")}}(flag)}`)
	cache := &FixExtractor{}
	first, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{})
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{filepath.Join(root, "caller.go"): []byte(`package example;import "example/middle";func f(){middle.Stop(true);println("after");_=0};func g(){middle.Stop(false);println("after")}`)}
	second, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{Files: files, Disk: first.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Callables) <= len(second.Funcs) || afterCallReachable(t, second, "example.f") {
		t.Fatal("caller revision loses unaffected closure bodies")
	}
	files[filepath.Join(root, "stop/stop.go")] = []byte(`package stop;func Now(flag bool){func(value bool){}(flag)}`)
	third, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{Files: files, Disk: second.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	if !afterCallReachable(t, third, "example.f") {
		t.Fatal("dependency revision retains a stale closure panic")
	}
}
