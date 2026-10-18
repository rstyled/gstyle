package gopatterns

import (
	"fmt"
	"strings"
	"testing"
)

// The model stores concrete safe/unsafe values in two independent cells. It
// does not use the analyzer's bindings, aliases, summaries, or join functions.
func TestTaintReturnedClosuresAgainstTwoCellOracle(t *testing.T) {
	var source strings.Builder
	source.WriteString(`func pair(q string)(func()string,func(string)){return func()string{return q},func(v string){q=v}};`)
	expected := map[string]int{}
	id := 0
	for first := 0; first < 4; first++ {
		for second := 0; second < 4; second++ {
			for read := 0; read < 2; read++ {
				cells := [2]bool{}
				fmt.Fprintf(&source, `func trace%d(r *http.Request){readA,setA:=pair("SELECT 1");readB,setB:=pair("SELECT 1");_=readA;_=readB;_=setA;_=setB;input:=r.FormValue("q");_=input;`, id)
				for _, step := range []int{first, second} {
					target, tainted := step/2, step%2 == 1
					cells[target] = tainted
					argument := `"SELECT 1"`
					if tainted {
						argument = "input"
					}
					fmt.Fprintf(&source, "%s(%s);", []string{"setA", "setB"}[target], argument)
				}
				fmt.Fprintf(&source, "db.Exec(%s());};", []string{"readA", "readB"}[read])
				name := fmt.Sprintf("example.trace%d", id)
				expected[name] = 0
				if cells[read] {
					expected[name] = 1
				}
				id++
			}
		}
	}
	actual := map[string]int{}
	for _, flow := range sourceTaintFlows(t, source.String()) {
		actual[flow.FuncID]++
	}
	for name, want := range expected {
		if actual[name] != want {
			t.Errorf("%s: findings=%d want %d", name, actual[name], want)
		}
		delete(actual, name)
	}
	if id != 32 || len(actual) != 0 {
		t.Fatalf("oracle coverage=%d unexpected=%v", id, actual)
	}
}

func TestTaintRepeatedClosureAllocationRetainsOldCell(t *testing.T) {
	flows := sourceTaintFlows(t, `func pair(q string)(func()string,func(string)){return func()string{return q},func(v string){q=v}};func handler(r *http.Request){var old func()string;for i:=0;i<2;i++{read,set:=pair("SELECT 1");if i==0{old=read;set(r.FormValue("q"))}else{set("SELECT 1")}};db.Exec(old())}`)
	if len(flows) != 1 {
		t.Fatalf("old allocation family source lost: %+v", flows)
	}
}
