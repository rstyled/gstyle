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

func TestCursorNodeCurrentSource(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), []string{"adapters/gopatterns/primitivefix.go"})
	if err != nil {
		t.Fatal(err)
	}
	function := findFunction(t, facts, "callWrapper.visit")
	var calls int
	for _, call := range function.Calls {
		if call.Callee == "golang.org/x/tools/go/ast/astutil.Cursor.Node" {
			calls++
			if call.Dynamic || call.Local {
				t.Fatalf("expected concrete external method: %+v", call)
			}
		}
	}
	if calls != 1 || len(function.Mutations) != 1 || function.Mutations[0].Provenance != quality.External {
		t.Fatalf("lost raw query or receiver write: %+v", function)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	result := resultNamed(t, report, "callWrapper.visit")
	for _, effect := range result.Effects {
		if strings.Contains(effect.Detail, "golang.org/x/tools/go/ast/astutil.Cursor.Node") {
			t.Errorf("read-only Node retains false effect: %+v", effect)
		}
	}
	if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.MutationEffect }) {
		t.Fatal("lost real wrapper mutation")
	}
}

func TestCursorNodeBoundaries(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("go/ast";"golang.org/x/tools/go/ast/astutil")
func Query(c *astutil.Cursor)ast.Node{return c.Node()}
func Expression(c *astutil.Cursor)ast.Node{return (*astutil.Cursor).Node(c)}
func Forward(c *astutil.Cursor)ast.Node{return Query(c)}
func Shared(c *astutil.Cursor){id:=c.Node().(*ast.Ident);id.Name="changed"}
func Alias(c *astutil.Cursor){id:=c.Node().(*ast.Ident);alias:=id;alias.Name="changed"}
func Child(c *astutil.Cursor){call:=c.Node().(*ast.CallExpr);call.Args[0]=nil}
var saved ast.Node
func Publish(c *astutil.Cursor){saved=c.Node()}
func Escape(c *astutil.Cursor,keep func(ast.Node)){keep(c.Node())}
func Replace(c *astutil.Cursor){c.Replace(c.Node())}
func Apply(node ast.Node,visit func(*astutil.Cursor)bool){astutil.Apply(node,visit,nil)}
func Callback(c *astutil.Cursor,run func()){c.Node();run()}
func IO(c *astutil.Cursor){c.Node();println("retained")}
func Indirect(c *astutil.Cursor){query:=c.Node;query()}
func Dynamic(c interface{Node()ast.Node}){c.Node()}
type custom struct{count int}
func(c *custom)Node()ast.Node{c.count++;return nil}
func Other(c *custom){c.Node()}
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
				t.Errorf("%s retains false query effect: %+v", name, result)
			}
		}
	})
	t.Run("controls", func(t *testing.T) {
		for name, kind := range map[string]quality.Kind{"Shared": quality.UnknownEffect, "Alias": quality.UnknownEffect, "Child": quality.UnknownEffect, "Publish": quality.Global, "Escape": quality.UnknownEffect, "Replace": quality.UnknownEffect, "Apply": quality.UnknownEffect, "Callback": quality.UnknownEffect, "IO": quality.IO, "Indirect": quality.UnknownEffect, "Dynamic": quality.UnknownEffect, "Other": quality.MutationEffect} {
			result := resultNamed(t, report, name)
			if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
				t.Errorf("%s loses %s: %+v", name, kind, result)
			}
		}
		for _, name := range []string{"Shared", "Alias", "Child"} {
			function := findFunction(t, facts, name)
			if len(function.Mutations) != 1 || function.Mutations[0].Provenance == quality.Local {
				t.Errorf("%s loses raw shared write: %+v", name, function.Mutations)
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
		t.Fatal("saved cursor facts change evaluation", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "golang.org/x/tools/go/ast/astutil.Cursor.Node", Kind: quality.IO})
	overridden, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Query", "Expression", "Forward", "Shared"} {
		result := resultNamed(t, overridden, name)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.IO }) {
			t.Errorf("%s loses saved-fact query policy: %+v", name, result)
		}
	}
}
