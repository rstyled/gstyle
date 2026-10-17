package gopatterns

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func extractedFrameEffects(t *testing.T, extraction *Extraction) map[string]callflow.FrameEffect {
	t.Helper()
	graphs := map[string]*pdg.Graph{}
	for i := range extraction.Funcs {
		function := &extraction.Funcs[i]
		graphs[function.Name] = &function.Pdg
	}
	effects, err := callflow.FrameEffects(graphs)
	if err != nil {
		t.Fatal(err)
	}
	return effects
}
func TestFrameEffectsFollowLocalCallAndValueDependencies(t *testing.T) {
	const source = `package example
import "runtime"
func First(x int)int{return left(x)}
func Second(x int)int{return right(x)}
func left(x int)int{return middle(x)}
func right(x int)int{return middle(x)}
func middle(x int)int{target:=observe;alias:=target;return alias(x)}
func observe(x int)int{_,_,line,_:=runtime.Caller(x);return line}
func plain(x int)int{return x+1}
func unused(x int)int{target:=observe;target=plain;return target(x)}
func branch(x int,b bool)int{target:=plain;if b{target=observe};return target(x)}
func closure(){_=func(){runtime.Caller(0)}}
func invokeClosure(){target:=func(){runtime.Caller(0)};target()}
func callback(target func()int)int{return target()}
`
	extraction, report := proofExtraction(t, source)
	effects := extractedFrameEffects(t, extraction)
	for _, name := range []string{"First", "Second", "left", "right", "middle", "observe", "branch"} {
		if got := effects["example."+name]; !got.Observes || got.Unknown {
			t.Errorf("%s=%+v", name, got)
		}
	}
	for _, name := range []string{"plain", "unused", "closure"} {
		if got := effects["example."+name]; got != (callflow.FrameEffect{}) {
			t.Errorf("%s=%+v", name, got)
		}
	}
	for _, name := range []string{"invokeClosure", "callback"} {
		if got := effects["example."+name]; !got.Unknown {
			t.Errorf("%s=%+v", name, got)
		}
	}
	if err := VerifyParameterizations(&report, extraction); err != nil {
		t.Fatal(err)
	}
	candidate := report.Candidates[0]
	if candidate.FixSpec != nil || !strings.Contains(strings.Join(candidate.CounterEvidence, " "), "stack-inspection") {
		t.Fatalf("transitive frame-dependent extraction accepted: %+v", candidate)
	}
}
func TestFrameSummaryChainsHaveNoPassLimit(t *testing.T) {
	source := `package example;import "runtime";func First()int{return left()};func Second()int{return right()};func left()int{return hop24()};func right()int{return hop24()};func hop0()int{_,_,line,_:=runtime.Caller(0);return line};`
	for i := 1; i < 25; i++ {
		source += fmt.Sprintf("func hop%d()int{return hop%d()};", i, i-1)
	}
	extraction, report := proofExtraction(t, source)
	if err := VerifyParameterizations(&report, extraction); err != nil {
		t.Fatal(err)
	}
	if report.Candidates[0].FixSpec != nil {
		t.Fatal("long local call chain hides frame dependency")
	}
}

func TestFrameEffectsIncludeDeferredInvocations(t *testing.T) {
	source := `package example
import "runtime"
func First()int{return left()}
func Second()int{return right()}
func left()int{defer observe();return 1}
func right()int{defer observe();return 2}
func observe(){runtime.Caller(0)}
func neutral(){defer plain()}
func plain(){}
`
	extraction, report := proofExtraction(t, source)
	effects := extractedFrameEffects(t, extraction)
	for _, name := range []string{"First", "Second", "left", "right"} {
		if effect := effects["example."+name]; !effect.Observes || effect.Unknown {
			t.Fatalf("deferred frame effect %s=%+v", name, effect)
		}
	}
	if effects["example.neutral"] != (callflow.FrameEffect{}) {
		t.Fatal("neutral defer has invented frame effects")
	}
	if err := VerifyParameterizations(&report, extraction); err != nil {
		t.Fatal(err)
	}
	if report.Candidates[0].FixSpec != nil {
		t.Fatal("deferred stack inspection permits extraction")
	}
}
func TestFrameSummaryUnknownCallsCannotAuthorizeExtraction(t *testing.T) {
	for _, body := range []string{"return cb(x)", "return external(x)"} {
		source := `package example;var external func(int)int;func First(x int,cb func(int)int)int{left();` + body + `};func Second(x int,cb func(int)int)int{right();` + body + `};func left(){};func right(){}`
		extraction, report := proofExtraction(t, source)
		if err := VerifyParameterizations(&report, extraction); err != nil {
			t.Fatal(err)
		}
		candidate := report.Candidates[0]
		if candidate.FixSpec != nil || !strings.Contains(strings.Join(candidate.CounterEvidence, " "), "unknown") {
			t.Fatalf("unknown call granted extraction: %+v", candidate)
		}
	}
}
func TestFrameSensitiveHelperChangesNativeCaller(t *testing.T) {
	source := `package example
import "runtime"
func First()string{return left()}
func Second()string{return right()}
//go:noinline
func left()string{return observe()}
//go:noinline
func right()string{return observe()}
//go:noinline
func observe()string{pc,_,_,_:=runtime.Caller(2);return runtime.FuncForPC(pc).Name()}
//go:noinline
func helper(fn func()string)string{return fn()}
func manualFirst()string{return helper(left)}
`
	extraction, report := proofExtraction(t, source)
	test := `package example
import("strings";"testing")
func TestNativeFrame(t *testing.T){
 if got:=First();!strings.HasSuffix(got,".First"){t.Fatal(got)}
 if got:=Second();!strings.HasSuffix(got,".Second"){t.Fatal(got)}
 if got:=manualFirst();!strings.HasSuffix(got,".helper"){t.Fatal(got)}
}
`
	if err := os.WriteFile("sample_test.go", []byte(test), 0600); err != nil {
		t.Fatal(err)
	}
	assertParameterizationModule(t)
	if err := VerifyParameterizations(&report, extraction); err != nil {
		t.Fatal(err)
	}
	candidate := report.Candidates[0]
	if candidate.FixSpec != nil || !strings.Contains(strings.Join(candidate.CounterEvidence, " "), "stack-inspection") {
		t.Fatalf("observable helper frame accepted: %+v", candidate)
	}
}
