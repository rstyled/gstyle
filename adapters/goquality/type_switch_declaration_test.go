package goquality

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestCurrentSourceTypeSwitchGuardDoesNotMutate(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), []string{"adapters/gopatterns/ir_branch_cfg.go"})
	if err != nil {
		t.Fatal(err)
	}
	f := findFunction(t, facts, "branchCFGLoop")
	if len(f.Mutations) != 4 {
		t.Errorf("type-switch declaration adds a write: %+v", f.Mutations)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if result := resultNamed(t, report, "branchCFGLoop"); result.MutatedTargets != 3 {
		t.Errorf("type-switch declaration adds a target: got %d, want 3", result.MutatedTargets)
	}
	for _, d := range report.Diagnostics {
		if d.Name == f.Name && d.RuleID == "mutated_targets" {
			t.Errorf("false target diagnostic: %+v", d)
		}
	}
	for _, name := range []string{"branchCFGStatement", "branchCFGSelection"} {
		if result := resultNamed(t, report, name); result.MutatedTargets != 4 {
			t.Errorf("%s targets: got %d, want 4", name, result.MutatedTargets)
		}
	}
}

func TestTypeSwitchDeclarationsKeepAssignmentsAndEffects(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
func observe(value any) any { println("observed"); return value }
func Declare(value any) { switch item := observe(value).(type) { case *int: *item=1; case []int: item[0]=2; case int, string: _=item; default: _=item } }
func Assign(value any) { var item any; switch item = observe(value); item.(type) { case int: _=item }; switch item = value; item.(type) { default: _=item }; switch item = value; item.(type) { default: _=item }; switch item = value; item.(type) { default: _=item } }
func Init(value any) { var old any; switch old = observe(value); item := old.(type) { default: _=item } }
func Nested(value any) { _=func(){switch item:=value.(type){case *int:*item=3}} }
func CaseAssign(value any) { switch item := value.(type) { case int: item=1; _=item; default: item=nil; _=item } }
func NoBinding(value any) { switch observe(value).(type) { default: } }
func Mixed(value any) { old:=0; old, fresh:=1,2; _, fresh=0,3; _=old; _=fresh }
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		writes int
	}{{"Declare", 2}, {"Assign", 4}, {"Init", 1}, {"Nested", 1}, {"CaseAssign", 2}, {"NoBinding", 0}, {"Mixed", 2}} {
		f := findFunction(t, facts, tc.name)
		if len(f.Mutations) != tc.writes {
			t.Errorf("%s writes: got %+v, want %d", tc.name, f.Mutations, tc.writes)
		}
	}
	for _, name := range []string{"Declare", "Nested"} {
		for _, m := range findFunction(t, facts, name).Mutations {
			if m.Root != "item" || m.Provenance != quality.Unknown {
				t.Errorf("%s loses case write: %+v", name, m)
			}
		}
	}
	data, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	var replay []quality.Function
	if err := json.Unmarshal(data, &replay); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.Network, quality.None} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.observe", Kind: kind})
		report, err := quality.Evaluate(facts, config)
		if err != nil {
			t.Fatal(err)
		}
		restored, err := quality.Evaluate(replay, config)
		if err != nil || !reflect.DeepEqual(report, restored) {
			t.Fatal("type-switch facts change after replay", err)
		}
		for _, name := range []string{"Declare", "Assign", "Init", "NoBinding"} {
			result := resultNamed(t, report, name)
			calls := 0
			for _, e := range result.Effects {
				if e.Detail == "example.com/quality-repro.observe" && e.Kind == kind {
					calls++
				}
			}
			if kind != quality.None && calls != 1 {
				t.Errorf("%s loses guard/init call policy %s: %+v", name, kind, result.Effects)
			}
		}
		result := resultNamed(t, report, "Assign")
		if result.Mutations != 4 || result.MutatedTargets != 1 {
			t.Errorf("assignment guard loses writes: %+v", result)
		}
		found := false
		for _, d := range report.Diagnostics {
			if d.Name == result.Name && d.RuleID == "mutation_per_target" && d.Actual == 4 {
				found = true
			}
		}
		if !found {
			t.Error("four assignment guards lose mutation budget diagnostic")
		}
	}
}
