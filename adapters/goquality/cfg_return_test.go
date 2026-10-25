package goquality

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestCFGReturnCurrentSource(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), []string{"adapters/gopatterns/ir_implicit_return.go"})
	if err != nil {
		t.Fatal(err)
	}
	function := findFunction(t, facts, "implicitReturnFacts")
	var calls int
	for _, call := range function.Calls {
		if call.Callee == "golang.org/x/tools/go/cfg.Block.Return" {
			calls++
			if call.Dynamic || call.Local {
				t.Fatalf("expected concrete external method: %+v", call)
			}
		}
	}
	if calls != 1 {
		t.Fatalf("want one raw Return call, got %d", calls)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	result := resultNamed(t, report, "implicitReturnFacts")
	var mutations int
	for _, effect := range result.Effects {
		if strings.Contains(effect.Detail, "golang.org/x/tools/go/cfg.Block.Return") {
			t.Errorf("read-only Return retains false effect: %+v", effect)
		}
		if effect.Kind == quality.MutationEffect {
			mutations++
		}
	}
	if mutations == 0 {
		t.Fatal("lost builder mutation evidence")
	}
}

func TestCFGReturnBoundaries(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("go/ast";"golang.org/x/tools/go/cfg")
func Query(b *cfg.Block)*ast.ReturnStmt{return b.Return()}
func Expression(b *cfg.Block)*ast.ReturnStmt{return (*cfg.Block).Return(b)}
func Forward(b *cfg.Block)*ast.ReturnStmt{return Query(b)}
func WriteBlock(b *cfg.Block){b.Return();b.Nodes=nil}
func WriteResult(b *cfg.Block){r:=b.Return();r.Results=nil}
func Alias(b *cfg.Block){r:=b.Return();alias:=r;alias.Results=nil}
func Child(b *cfg.Block){r:=b.Return();r.Results[0]=nil}
var saved *ast.ReturnStmt
func Publish(b *cfg.Block){saved=b.Return()}
func Callback(b *cfg.Block,f func()){b.Return();f()}
func IO(b *cfg.Block){b.Return();println("real")}
func Indirect(b *cfg.Block){f:=b.Return;f()}
func Dynamic(b interface{Return()*ast.ReturnStmt}){b.Return()}
func New(body *ast.BlockStmt,mayReturn func(*ast.CallExpr)bool){cfg.New(body,mayReturn)}
`))
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/quality-repro\n\ngo 1.26.0\n\nrequire golang.org/x/tools v0.50.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	sums, err := os.ReadFile("../../go.sum")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.sum"), sums, 0600); err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Run("queries", func(t *testing.T) {
		for _, name := range []string{"Query", "Expression", "Forward"} {
			result := resultNamed(t, report, name)
			if !result.Complete || len(result.Effects) != 0 {
				t.Errorf("%s retains false effect: %+v", name, result)
			}
		}
	})
	t.Run("controls", func(t *testing.T) {
		for name, kind := range map[string]quality.Kind{"WriteBlock": quality.MutationEffect, "WriteResult": quality.UnknownEffect, "Alias": quality.UnknownEffect, "Child": quality.UnknownEffect, "Publish": quality.Global, "Callback": quality.UnknownEffect, "IO": quality.IO, "Indirect": quality.UnknownEffect, "Dynamic": quality.UnknownEffect, "New": quality.UnknownEffect} {
			result := resultNamed(t, report, name)
			if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
				t.Errorf("%s loses %s: %+v", name, kind, result)
			}
		}
	})
	data, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	var restored []quality.Function
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	replay, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil || !reflect.DeepEqual(report, replay) {
		t.Fatal("saved CFG call facts changed", err)
	}
}
