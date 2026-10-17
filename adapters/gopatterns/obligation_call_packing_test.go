package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestResourceCleanupUsesVariadicReceivingValues(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"empty pack permits close", `stop();file.Close()`, 0},
		{"nonempty pack aborts", `stop(0);file.Close()`, 1},
		{"saved empty pack permits older close", `defer file.Close();defer stop()`, 0},
		{"saved nonempty pack blocks older close", `defer file.Close();defer stop(0)`, 1},
		{"spread nil permits close", `stop([]int(nil)...);file.Close()`, 0},
		{"empty nonnil spread aborts", `stop([]int{}...);file.Close()`, 1},
		{"argument-state recursion reaches close", `walk(0);file.Close()`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;import("io/fs";"os");func stop(values ...int){if values!=nil{os.Exit(1)}};func walk(values ...int){if values==nil{return};walk()};func caller(files fs.FS){file,err:=files.Open("file");if err!=nil{return};`+tc.body+`}`)
			summaries, err := packageResourceSummaries(pkg)
			if err != nil {
				t.Fatal(err)
			}
			fn := findFunc(pkg.Syntax[0], "caller")
			graph, err := buildGraph(pkg, fn, "p.go", pdg.NewBuilder())
			if err != nil {
				t.Fatal(err)
			}
			ctx := extractCtx{fileSources: summaries.sources, fileFailures: summaries.failures, helpers: summaries.helpers, exits: summaries.exits}
			_, sites, err := fileObligationFacts(pkg, fn, ctx, &graph)
			if err != nil {
				t.Fatal(err)
			}
			if len(sites) != tc.want {
				t.Fatalf("obligations=%v want %d", sites, tc.want)
			}
		})
	}
}
