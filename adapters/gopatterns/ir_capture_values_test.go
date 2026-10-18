package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"sync"
	"testing"
)

func TestCapturedCellInvariantsComposeExitEffects(t *testing.T) {
	for _, tc := range []struct {
		name, body, helpers string
		want                callflow.Outcomes
	}{
		{"false", `flag:=false;func(){if flag{panic("bad")}}()`, "", callflow.Returns},
		{"true", `flag:=true;func(){if flag{panic("yes")}}()`, "", callflow.Panics},
		{"nil", `var pointer *int;func(){if pointer!=nil{panic("bad")}}()`, "", callflow.Returns},
		{"boxed nil", `var pointer *int;var boxed any=pointer;func(){if boxed==nil{panic("bad")}}()`, "", callflow.Returns},
		{"returned value before capture", `flag:=value();func(){if flag{panic("bad")}}()`, `func value()bool{return false}`, callflow.Returns},
		{"stored copy", `flag:=false;fn:=func(){if flag{panic("bad")}};copy:=fn;copy()`, "", callflow.Returns},
		{"same write", `flag:=false;flag=false;func(){if flag{panic("bad")}}()`, "", callflow.Returns},
		{"write before capture", `flag:=false;flag=true;func(){if flag{panic("yes")}}()`, "", callflow.Panics},
		{"live after construction", `flag:=true;fn:=func(){if flag{panic("maybe")}};flag=false;fn()`, "", callflow.Returns},
		{"parameter", `outer(false)`, `func outer(flag bool){func(){if flag{panic("bad")}}()}`, callflow.Returns},
		{"nested relay", `flag:=false;func(){func(){if flag{panic("bad")}}()}()`, "", callflow.Returns},
		{"captured return", `flag:=false;if func()bool{return flag}(){panic("bad")}`, "", callflow.Returns},
		{"ordinary args", `flag:=false;func(value bool){if value&&flag{panic("bad")}}(true)`, "", callflow.Returns},
		{"indirect write", `flag:=false;pointer:=&flag;*pointer=true;func(){if flag{panic("maybe")}}()`, "", callflow.Returns | callflow.Panics},
		{"unknown escape", `flag:=false;foreign(&flag);func(){if flag{panic("maybe")}}()`, `var foreign func(*bool)`, callflow.UnknownExit},
		{"deferred live cell", `flag:=false;defer func(){if flag{panic("maybe")}}();flag=true`, "", callflow.Panics},
		{"deferred saved arg", `flag:=false;defer func(value bool){if value&&flag{panic("bad")}}(false);flag=true`, "", callflow.Returns},
		{"deferred true arg live cell", `flag:=false;defer func(value bool){if value&&flag{panic("maybe")}}(true);flag=true`, "", callflow.Panics},
		{"loop retained older cell", `saved:=func(){};again,first:=true,true;for again{flag:=false;if first{saved=func(){if flag{panic("old cell")}};flag=true;first=false}else{again=false}};saved()`, "", callflow.Returns | callflow.Panics},
		{"history state at join", `saved:=func(){};again,first:=true,true;for again{flag:=false;if first{saved=func(){if flag{panic("old cell")}};if !choose{}else{flag=true};first=false}else{again=false}};choose=false;saved()`, "", callflow.Returns | callflow.Panics},
		{"loop uniform cells", `saved:=func(){};again,first:=true,true;for again{flag:=false;if first{saved=func(){if flag{panic("bad")}};first=false}else{again=false}};saved()`, "", callflow.Returns},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;var choose bool;func f(){`+tc.body+`};`+tc.helpers)
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

func TestCapturedObservationsAreReadOnlyAfterSolving(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example;func f(){flag:=false;func(){if flag{panic("bad")}}()}`)
	graphs := map[string]*pdg.Graph{}
	for _, graph := range extraction.Callables {
		graphs[graph.Tables.Text(graph.Function.ID)] = graph
	}
	summaries, err := callflow.ExitEffects(graphs)
	if err != nil {
		t.Fatal(err)
	}
	// A new graph view deliberately misses the solver's graph-pointer cache.
	owner := *graphs["example.f"]
	queries, err := callflow.NewQueries(&owner)
	if err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	failures := make(chan string, 16)
	for range 16 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			observed := callflow.OrdinaryObservations(queries, summaries)
			if len(observed.Calls) != 1 {
				failures <- "missing captured call"
				return
			}
			for _, outcome := range observed.Calls {
				if outcome != callflow.Returns {
					failures <- "captured outcome loses its invariant"
				}
			}
		}()
	}
	wait.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
}
