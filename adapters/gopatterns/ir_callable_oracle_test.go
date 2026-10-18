package gopatterns

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"strings"
	"testing"
)

func TestCallableContextsAgainstTargetChoiceOracle(t *testing.T) {
	var source strings.Builder
	source.WriteString(`package example;var choose bool;func cb0(x bool)bool{return false};func cb1(x bool)bool{return true};func cb2(x bool)bool{return x};func cb3(x bool)bool{return !x};func cb4(x bool)bool{panic("callback")};
 func apply(cb func(bool)bool,x bool)bool{return cb(x)};
 func forward(cb func(bool)bool,x,again bool)bool{if again{return forward(cb,x,false)};return apply(cb,x)};
 func saved(cb func(bool)bool,x bool)(result bool){defer func(fn func(bool)bool,v bool){result=fn(v)}(cb,x);cb=cb0;x=!x;return false};`)
	want := map[string]callflow.Outcomes{}
	index := 0
	for a := 0; a < 5; a++ {
		for b := 0; b < 5; b++ {
			pick := fmt.Sprintf("pick%d_%d", a, b)
			fmt.Fprintf(&source, "func %s(v bool)func(bool)bool{if v{return cb%d};return cb%d};", pick, a, b)
			for _, input := range []bool{false, true} {
				for _, expected := range []bool{false, true} {
					for _, wrapper := range []string{"apply", "forward", "saved"} {
						name := fmt.Sprintf("f%d", index)
						index++
						args := fmt.Sprintf("%s(choose),%t", pick, input)
						if wrapper == "forward" {
							args += ",true"
						}
						fmt.Fprintf(&source, "func %s(){if %s(%s)!=%t{panic(\"result\")}};", name, wrapper, args, expected)
						// Independent concrete interpretation of each possible selected callback.
						var outcomes callflow.Outcomes
						for _, target := range []int{a, b} {
							truth := input
							switch target {
							case 0:
								truth = false
							case 1:
								truth = true
							case 3:
								truth = !input
							case 4:
								outcomes |= callflow.Panics
								continue
							}
							if truth == expected {
								outcomes |= callflow.Returns
							} else {
								outcomes |= callflow.Panics
							}
						}
						want["example."+name] = outcomes
					}
				}
			}
		}
	}
	checkCallableSourceOracle(t, source.String(), want)
	if len(want) != 300 {
		t.Fatal("incomplete target choice oracle")
	}
}
func checkCallableSourceOracle(t *testing.T, source string, want map[string]callflow.Outcomes) {
	t.Helper()
	extraction, _ := proofExtraction(t, source)
	graphs := map[string]*pdg.Graph{}
	for _, graph := range extraction.Callables {
		graphs[graph.Tables.Text(graph.Function.ID)] = graph
	}
	summaries, err := callflow.ExitEffects(graphs)
	if err != nil {
		t.Fatal(err)
	}
	for name, expected := range want {
		if got := summaries[name].Ordinary; got != expected {
			t.Errorf("%s: got %v want %v", name, got, expected)
		}
	}
}
func TestReturnedCallableSetRetainsThreeHundredIdentities(t *testing.T) {
	var source strings.Builder
	source.WriteString(`package example;func f(){read:=pick(0);if read(){panic("bad")}};`)
	for i := 0; i < 300; i++ {
		fmt.Fprintf(&source, "func cb%d()bool{return false};", i)
	}
	source.WriteString(`func pick(index int)func()bool{switch index{`)
	for i := 0; i < 299; i++ {
		fmt.Fprintf(&source, "case %d:return cb%d;", i, i)
	}
	source.WriteString(`};return cb299}`)
	checkCallableSourceOracle(t, source.String(), map[string]callflow.Outcomes{"example.f": callflow.Returns})
}
