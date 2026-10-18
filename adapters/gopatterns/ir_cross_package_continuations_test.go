package gopatterns

import (
	"context"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
	"github.com/rstyled/gstyle/domain/runtimeguard"
	"golang.org/x/tools/go/packages"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func crossPackageModule(t *testing.T, target string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod":           "module example\n\ngo 1.24\n",
		"caller.go":        `package example;import "example/middle";func f(){middle.Stop(true);println("after")};func g(){middle.Stop(false);println("after")}`,
		"middle/middle.go": `package middle;import "example/stop";func Stop(flag bool){stop.Now(flag)}`,
		"stop/stop.go":     target,
	}
	for name, source := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func afterCallReachable(t *testing.T, extraction *Extraction, name string) bool {
	t.Helper()
	graph := continuationGraph(t, extraction, name)
	if err := pdg.Validate(graph); err != nil {
		t.Fatal(err)
	}
	for _, block := range graph.ValueFlow.Blocks {
		if !block.Reachable {
			continue
		}
		for _, ref := range block.Operations {
			if graph.Tables.Text(graph.NodeAttributes(graph.Nodes[ref-1]).Callee.Value) == "println" {
				return true
			}
		}
	}
	return false
}

func TestCrossPackageCallContinuations(t *testing.T) {
	for _, mode := range []string{"normal", "bounded", "cached"} {
		t.Run(mode, func(t *testing.T) {
			root := crossPackageModule(t, `package stop;func Now(flag bool){if flag{panic("stop")}}`)
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
			if afterCallReachable(t, extraction, "example.f") {
				t.Fatal("true argument retains normal continuation through imported chain")
			}
			if !afterCallReachable(t, extraction, "example.g") {
				t.Fatal("false argument loses normal continuation through imported chain")
			}
		})
	}
}

func TestCrossPackageCallContinuationRevisions(t *testing.T) {
	root := crossPackageModule(t, `package stop;func Now(flag bool){if flag{panic("stop")}}`)
	cache := &FixExtractor{}
	first, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{})
	if err != nil {
		t.Fatal(err)
	}
	if afterCallReachable(t, first, "example.f") {
		t.Fatal("initial imported panic retains continuation")
	}
	// Changing only the caller must retain the unaffected imported body queries.
	caller := filepath.Join(root, "caller.go")
	files := map[string][]byte{caller: []byte(`package example;import "example/middle";func f(){middle.Stop(false);println("after")};func g(){middle.Stop(true);println("after")}`)}
	second, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{Files: files, Disk: first.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	if !afterCallReachable(t, second, "example.f") || afterCallReachable(t, second, "example.g") {
		t.Fatal("caller revision loses imported argument contexts")
	}
	// A changed dependency must replace its old summaries and refresh importers.
	files[filepath.Join(root, "stop/stop.go")] = []byte(`package stop;func Now(flag bool){}`)
	third, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{Files: files, Disk: second.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	if !afterCallReachable(t, third, "example.f") || !afterCallReachable(t, third, "example.g") {
		t.Fatal("dependency revision reuses a stale nonreturning summary")
	}
}

func TestCrossPackageUnloadedBodyStaysUnknown(t *testing.T) {
	root := crossPackageModule(t, `package stop;func Now(flag bool){if flag{panic("stop")}}`)
	extraction, err := Extract(context.Background(), root, []string{"caller.go"})
	if err != nil {
		t.Fatal(err)
	}
	if !afterCallReachable(t, extraction, "example.f") {
		t.Fatal("unloaded imported body acquires a nonreturning proof")
	}
}

func TestProgramCallClosureMatchesWholeInventory(t *testing.T) {
	root := crossPackageModule(t, `package stop;func Now(flag bool){if flag{panic("stop")}}`)
	extraction, err := Extract(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	graphs := map[string]*pdg.Graph{}
	program := callflow.NewProgram()
	for i := range extraction.Funcs {
		graph := &extraction.Funcs[i].Pdg
		name := graph.Tables.Text(graph.Function.ID)
		graphs[name] = graph
		if name != "example.f" {
			if err := program.Add(graph); err != nil {
				t.Fatal(err)
			}
		}
	}
	full, err := callflow.ExitEffects(graphs)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := program.ExitEffects(map[string]*pdg.Graph{"example.f": graphs["example.f"]})
	if err != nil {
		t.Fatal(err)
	}
	if selected["example.f"].Ordinary != full["example.f"].Ordinary {
		t.Fatalf("selected=%v full=%v", selected["example.f"], full["example.f"])
	}
	if _, found := selected["example.g"]; found {
		t.Fatal("unrelated root enters callee closure")
	}
}

func TestCrossPackageExitKinds(t *testing.T) {
	for _, tc := range []struct {
		name, target string
		returns      bool
	}{
		{"panic", `package stop;func Now(flag bool){if flag{panic("stop")}}`, false},
		{"divergence", `package stop;func Now(flag bool){if flag{for{}}}`, false},
		{"recursion", `package stop;func Now(flag bool){if flag{Now(flag)}}`, false},
		{"goexit", `package stop;import "runtime";func Now(flag bool){if flag{runtime.Goexit()}}`, false},
		{"abort", `package stop;import "os";func Now(flag bool){if flag{os.Exit(1)}}`, false},
		{"callee recovery", `package stop;func Now(flag bool){if flag{defer absorb();panic("stop")}};func absorb(){recover()}`, true},
		{"unknown", `package stop;var foreign func();func Now(flag bool){if flag{foreign()}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := crossPackageModule(t, tc.target)
			extraction, err := Extract(context.Background(), root, []string{"."})
			if err != nil {
				t.Fatal(err)
			}
			if got := afterCallReachable(t, extraction, "example.f"); got != tc.returns {
				t.Fatalf("true argument continuation=%v want %v", got, tc.returns)
			}
			if !afterCallReachable(t, extraction, "example.g") {
				t.Fatal("false argument loses its returning path")
			}
		})
	}
}

func TestCrossPackageRuntimeGuardUsesRefinedFlow(t *testing.T) {
	root := crossPackageModule(t, `package stop;func Now(flag bool){if flag{panic("stop")}}`)
	path := filepath.Join(root, "caller.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source = append(source, []byte(`;func safe(pointer *int){if pointer==nil{middle.Stop(true)};_=*pointer}`)...)
	if err := os.WriteFile(path, source, 0644); err != nil {
		t.Fatal(err)
	}
	extraction, err := Extract(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	report, err := runtimeguard.Analyze(continuationGraph(t, extraction, "example.safe"))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Accesses) != 1 || !report.Accesses[0].Value.NonNil {
		t.Fatalf("imported panic does not establish the guard: %+v", report.Accesses)
	}
}

func TestCrossPackageContinuationsAcrossCompilerBatches(t *testing.T) {
	root := crossPackageModule(t, `package stop;func Now(flag bool){if flag{panic("stop")}}`)
	// This indivisible root family exceeds the batch budget. Its dependencies
	// must finish in an earlier compiler batch, not merely earlier in one batch.
	padding := []byte("package example;/*" + strings.Repeat(" ", int(sourceBatchBytes)) + "*/")
	if err := os.WriteFile(filepath.Join(root, "padding.go"), padding, 0644); err != nil {
		t.Fatal(err)
	}
	extraction, err := ExtractForFix(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	if afterCallReachable(t, extraction, "example.f") {
		t.Fatal("imported panic is lost across compiler batches")
	}
	if !afterCallReachable(t, extraction, "example.g") {
		t.Fatal("returning argument is lost across compiler batches")
	}
}

func TestCrossPackageResourceConsumerSharesExitScope(t *testing.T) {
	root := crossPackageModule(t, `package stop;func Now(flag bool){if flag{panic("stop")}}`)
	path := filepath.Join(root, "caller.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Replace(string(source), `import "example/middle"`, `import ("example/middle";"os")`, 1) + `;func owned(file *os.File){defer file.Close();middle.Stop(true)}`
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	pkgs, err := packages.Load(&packages.Config{Dir: root, Mode: packages.LoadSyntax}, "./...")
	if err != nil {
		t.Fatal(err)
	}
	if err := patternPackageErrors(pkgs, root); err != nil {
		t.Fatal(err)
	}
	program := callflow.NewProgram()
	for _, pkg := range orderedSourcePackages(productionPatternPackages(pkgs)) {
		analysis := packageAnalysis{pkg: pkg, root: root, pool: pdg.NewBuilder(), program: program}
		if err := analysis.build(); err != nil {
			t.Fatal(err)
		}
		if err := analysis.summarize(); err != nil {
			t.Fatal(err)
		}
		for _, source := range analysis.graphs {
			graph := source.graph
			if graph.Tables.Text(graph.Function.ID) != "example.owned" {
				continue
			}
			query, err := resourceflow.New(graph)
			if err != nil {
				t.Fatal(err)
			}
			if !query.FromEntry(resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol, Exits: analysis.context.exits}) {
				t.Fatal("resource consumer loses imported panic/defer exit contract")
			}
			return
		}
	}
	t.Fatal("missing resource function")
}
