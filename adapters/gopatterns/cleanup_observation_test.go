package gopatterns

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
)

func TestCleanupPresenceKeepsUnresolvedOwnership(t *testing.T) {
	pkg := loadTestPackage(t, `package p
import("time";"os")
var choose bool
func direct(){t:=time.NewTimer(time.Second);defer t.Stop();<-t.C}
func alias(){t:=time.NewTimer(time.Second);a:=t;defer a.Stop();<-t.C}
func saved(other *time.Timer){t:=time.NewTimer(time.Second);defer t.Stop();t=other;<-t.C}
func overwritten(other *time.Timer){t:=time.NewTimer(time.Second);t=other;defer t.Stop();<-t.C}
func wrong(other *time.Timer){t:=time.NewTimer(time.Second);defer other.Stop();<-t.C}
func shadow(){t:=time.NewTimer(time.Second);{t:=time.NewTimer(time.Second);defer t.Stop()};<-t.C}
func method(){t:=time.NewTimer(time.Second);defer (*time.Timer).Stop(t);<-t.C}
func conditional(){t:=time.NewTimer(time.Second);if choose{defer t.Stop()};<-t.C}
func closure(){t:=time.NewTimer(time.Second);_=func(){defer t.Stop()};<-t.C}
func unreachable(){t:=time.NewTimer(time.Second);<-t.C;return;defer t.Stop()}
func mixed(other *time.Timer){t:=time.NewTimer(time.Second);a:=t;if choose{a=other};defer a.Stop();<-t.C}
func copied(){t:=time.NewTimer(time.Second);v:=*t;defer v.Stop();<-t.C}
func two(){t:=time.NewTimer(time.Second);other:=time.NewTimer(time.Second);defer other.Stop();<-t.C}
func indirect(other *time.Timer){t:=time.NewTimer(time.Second);ptr:=&t;*ptr=other;defer t.Stop();<-t.C}
func captured(other *time.Timer){t:=time.NewTimer(time.Second);func(){t=other}();defer t.Stop();<-t.C}
func fileFactory(name string)*os.File{f,_:=os.Open(name);return f}
func file(){f:=fileFactory("x");defer f.Close();os.Stat("x")}
`)
	root := filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename)
	extraction, err := Extract(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"direct": {"t.Stop()"}, "alias": {"a.Stop()"}, "saved": {"t.Stop()"},
		"overwritten": nil, "wrong": nil, "shadow": nil, "method": {"(*time.Timer).Stop(t)"},
		"conditional": {"t.Stop()"}, "closure": nil, "unreachable": nil, "mixed": nil, "file": {"f.Close()"}, "indirect": nil, "captured": nil, "copied": nil, "two": nil,
	}
	for _, function := range extraction.Funcs {
		name := function.Name[strings.LastIndex(function.Name, ".")+1:]
		expected, selected := want[name]
		if !selected {
			continue
		}
		facts := &patterns.MiningFacts{ID: function.Name, Pdg: patterns.MiningView(&function.Pdg),
			ResourceLifecycles: function.ResourceLifecycles, FileObligationSites: function.FileObligationSites}
		violations := patterns.FindObligationViolations([]*patterns.MiningFacts{facts})
		candidates := patterns.ObligationCandidates(violations, []*patterns.MiningFacts{facts})
		if len(candidates) == 0 {
			t.Errorf("%s: warning was suppressed", name)
			continue
		}
		candidate := candidates[0]
		if name == "shadow" || name == "two" {
			if len(candidates) != 2 || len(candidates[1].CounterEvidence) != 1 {
				t.Errorf("%s: cleanup was not linked to the second acquisition: %+v", name, candidates)
			}
		}
		if candidate.ResourceLifecycle != nil && candidate.ResourceLifecycle.State != patterns.ResourceUnresolved {
			t.Errorf("%s: changed ownership state: %+v", name, candidate.ResourceLifecycle)
		}
		if len(candidate.CounterEvidence) != len(expected) {
			t.Errorf("%s: evidence=%v, want %v", name, candidate.CounterEvidence, expected)
			continue
		}
		for i, call := range expected {
			if !strings.Contains(candidate.CounterEvidence[i], "defer "+call+" observed at line ") || !strings.Contains(candidate.CounterEvidence[i], "remain unproved") {
				t.Errorf("%s: misleading evidence: %q", name, candidate.CounterEvidence[i])
			}
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Fatalf("unchecked functions: %v", want)
	}
}
