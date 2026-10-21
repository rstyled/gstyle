package gopatterns

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestResourceLifecycleContracts(t *testing.T) {
	pkg := loadTestPackage(t, `package p
import("time";"net";"database/sql")
var choose bool
type owner struct{ticker *time.Ticker}
func factory()*time.Ticker{return time.NewTicker(time.Second)}
func returned()*time.Ticker{t:=factory();return t}
func local(){t:=time.NewTicker(time.Second);t.Stop()}
func deferred(){t:=factory();defer t.Stop()}
func leak(){t:=factory();_=t}
func wrong(){t:=factory();other:=factory();other.Stop();_=t}
func conditional(){t:=factory();if choose{t.Stop()}}
func helper(t *time.Ticker){t.Stop()}
func helperCaller(){t:=factory();helper(t)}
func timer(){t:=time.NewTimer(time.Second);t.Stop()}
func conn(){c,err:=net.Dial("tcp","example.com:80");if err!=nil{return};c.Close()}
func db(){d,err:=sql.Open("driver","dsn");if err!=nil{return};d.Close()}
func dbFactory()(*sql.DB,error){return sql.Open("driver","dsn")}
func dbLeak(){d,err:=dbFactory();if err!=nil{return};_=d}
func object()*owner{t:=factory();return &owner{ticker:t}}
func inlineObject()*owner{return &owner{ticker:time.NewTicker(time.Second)}}
func nestedObject()[]*owner{return []*owner{{ticker:factory()},{ticker:factory()}}}
func boundObject()*owner{t:=factory();out:=&owner{ticker:t};return out}
func overwritten()*owner{t:=factory();out:=&owner{ticker:t};out=&owner{};return out}
func aliasReturn(other *time.Ticker)*time.Ticker{t:=factory();alias:=t;t=other;_=t;return alias}
func wrongKind(){t:=factory();timer:=time.NewTimer(time.Second);timer.Stop();_=t}
func dynamicStop(stop func(*time.Ticker)){t:=factory();stop(t)}
func closedFactory()*time.Ticker{t:=factory();t.Stop();return t}
func discard(*time.Ticker){}
func mixed(){t:=time.NewTicker(time.Second);t.Stop();discard(time.NewTicker(time.Second))}
`)
	root := filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename)
	extraction, err := Extract(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"factory": {"returned"}, "returned": {"returned"}, "local": {"released"},
		"deferred": {"released"}, "leak": {"unresolved"}, "wrong": {"unresolved", "released"},
		"conditional": {"unresolved"}, "helperCaller": {"released"}, "timer": {"released"},
		"conn": {"released"}, "db": {"released"}, "dbFactory": {"returned"}, "dbLeak": {"unresolved"},
		"object": {"returned_owner"}, "inlineObject": {"returned_owner"}, "nestedObject": {"returned_owner", "returned_owner"},
		"boundObject": {"returned_owner"}, "overwritten": {"unresolved"},
		"aliasReturn": {"returned"}, "wrongKind": {"unresolved", "released"}, "dynamicStop": {"unresolved"}, "closedFactory": {"released"},
		"mixed": {"released"},
	}
	for _, function := range extraction.Funcs {
		name := function.Name[strings.LastIndex(function.Name, ".")+1:]
		expected, selected := want[name]
		if !selected {
			continue
		}
		var states []string
		for _, fact := range function.ResourceLifecycles {
			states = append(states, string(fact.State))
		}
		if strings.Join(states, ",") != strings.Join(expected, ",") {
			t.Errorf("%s states=%v, want %v; facts=%+v", name, states, expected, function.ResourceLifecycles)
		}
		wantViolations := 0
		for _, state := range expected {
			if state != "released" && state != "returned" {
				wantViolations++
			}
		}
		if name == "mixed" {
			wantViolations++ // The unsupported inline argument retains its raw warning.
		}
		facts := &patterns.MiningFacts{ID: function.Name, Pdg: patterns.MiningView(&function.Pdg), ResourceLifecycles: function.ResourceLifecycles}
		if got := patterns.FindObligationViolations([]*patterns.MiningFacts{facts}); len(got) != wantViolations {
			t.Errorf("%s obligations=%+v, want %d", name, got, wantViolations)
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Fatalf("missing functions: %v", want)
	}
}

func TestResourceAdviceDoesNotStopReturnedTicker(t *testing.T) {
	fact := &patterns.MiningFacts{ID: "owner", Pdg: &patterns.MiningGraph{}, ResourceLifecycles: []patterns.ResourceLifecycle{{ResourceAcquisition: patterns.ResourceAcquisition{Line: 3, Resource: "ticker", AcquireCall: "time.NewTicker"}, State: "unresolved", Reason: "owner lifetime is unresolved"}}}
	violations := patterns.FindObligationViolations([]*patterns.MiningFacts{fact})
	candidates := patterns.ObligationCandidates(violations, []*patterns.MiningFacts{fact})
	if len(candidates) != 1 || candidates[0].ResourceLifecycle == nil {
		t.Fatalf("missing lifecycle evidence: %+v", candidates)
	}
	if strings.Contains(candidates[0].PossibleRefactor, "right after") || !strings.Contains(candidates[0].PossibleRefactor, "owner") {
		t.Fatalf("unsafe constructor advice: %s", candidates[0].PossibleRefactor)
	}
}
