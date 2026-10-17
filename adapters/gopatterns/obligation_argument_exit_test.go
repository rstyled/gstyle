package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestResourceCleanupUsesCallArgumentContexts(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"false reaches close", `stop(false);file.Close()`, 0},
		{"true aborts before close", `stop(true);file.Close()`, 1},
		{"saved false permits older close", `defer file.Close();defer stop(false)`, 0},
		{"saved true skips older close", `defer file.Close();defer stop(true)`, 1},
		{"terminating recursion reaches close", `walk(true);file.Close()`, 0},
		{"true recursion blocks close", `loop(true);file.Close()`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;import("io/fs";"os");func stop(flag bool){if flag{os.Exit(1)}};func walk(flag bool){if !flag{return};walk(false)};func loop(flag bool){if flag{loop(true)}};func caller(files fs.FS){file,err:=files.Open("file");if err!=nil{return};`+tc.body+`}`)
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
