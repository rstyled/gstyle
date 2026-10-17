package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestResourceCleanupUsesRecoveryContext(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"older close after recovery", `defer file.Close();defer absorb();panic("fault")`, 0},
		{"recovery alone leaks", `defer absorb();panic("fault")`, 1},
		{"builtin defer does not block older close", `defer file.Close();defer recover();panic("fault")`, 0},
		{"indirect recover does not block older close", `defer file.Close();defer indirect();panic("fault")`, 0},
		{"abort skips cleanup", `defer file.Close();defer absorb();os.Exit(1)`, 1},
		{"older abort still blocks close", `defer file.Close();defer os.Exit(1);defer absorb();panic("fault")`, 1},
		{"goexit runs older close", `defer file.Close();defer absorb();runtime.Goexit()`, 0},
		{"recovered panic keeps goexit", `defer file.Close();defer absorb();defer panic("new");runtime.Goexit()`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;import("io/fs";"os";"runtime");var _=os.Exit;var _=runtime.Goexit;func absorb(){recover()};func indirect(){absorb()};func caller(files fs.FS){file,err:=files.Open("file");if err!=nil{return};`+tc.body+`}`)
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

func TestResourceCleanupUsesRecoveredValueGuards(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"guarded recursion lets panic cleanup run", `defer file.Close();defer guardedRecursive();panic("fault")`, 0},
		{"real ordinary recursion blocks goexit cleanup", `defer file.Close();defer guardedRecursive();runtime.Goexit()`, 1},
		{"panic-only handler lets older close run", `defer file.Close();defer panicOnly();panic("fault")`, 0},
		{"panic-only handler aborts goexit", `defer file.Close();defer panicOnly();runtime.Goexit()`, 1},
		{"inactive-only handler aborts panic", `defer file.Close();defer inactiveOnly();panic("fault")`, 1},
		{"inactive-only handler lets goexit close", `defer file.Close();defer inactiveOnly();runtime.Goexit()`, 0},
		{"double recover retains first value", `defer file.Close();defer twice();panic("fault")`, 0},
		{"ordinary guarded recover aborts", `panicOnly();file.Close()`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;import("io/fs";"os";"runtime");var _=runtime.Goexit
func guardedRecursive(){if recover()!=nil{return};guardedRecursive()};func panicOnly(){if recover()==nil{os.Exit(1)}};func inactiveOnly(){if recover()!=nil{os.Exit(1)}};func twice(){first:=recover();second:=recover();if first==nil||second!=nil{os.Exit(1)}}
func caller(files fs.FS){file,err:=files.Open("file");if err!=nil{return};`+tc.body+`}`)
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
