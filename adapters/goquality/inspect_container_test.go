package goquality

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestInspectContainerCurrentSource(t *testing.T) {
	source := `package repro
import("fmt";"go/ast";"go/token";"go/types";"unicode")
type parameterizationSource struct {
 file *ast.File
 packageNames []string
 snapshot struct{Imports []struct{Name string}}
}
type parameterizationPair struct{second *parameterizationSource;a,b *ast.FuncDecl}
`
	source += inspectSourceFunctions(t, "../gopatterns/parameterize_hygiene.go", "identifierNames", "freshIdentifier", "freshHelperName", "validHelperIdentifier")
	source += inspectSourceFunctions(t, "../gopatterns/paramfix.go", "helperName", "commonPrefix", "commonSuffix")
	root := qualitySourceModule(t, []byte(source))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	writes := findFunction(t, facts, "freshHelperName").Mutations
	var used int
	for _, write := range writes {
		if write.Root == "used" {
			used++
			if write.Provenance != quality.Local {
				t.Errorf("current freshHelperName loses private Inspect map: %+v", write)
			}
		}
	}
	if used != 3 {
		t.Fatalf("want three raw used writes, got %+v", writes)
	}
	result := resultNamed(t, report, "freshHelperName")
	if !result.Complete || slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.UnknownEffect }) {
		t.Errorf("current freshHelperName keeps false unknown: %+v", result)
	}
	if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool {
		return e.Kind == quality.Global && strings.Contains(e.Detail, "go/types.Universe")
	}) {
		t.Fatal("Inspect ownership hides independent global evidence")
	}
}

func TestInspectContainerBoundaries(t *testing.T) {
	source := `package repro
import "go/ast"
var saved map[string]bool
var savedVisit func(ast.Node)bool
func retain(_ ast.Node, visit func(ast.Node)bool){savedVisit=visit}
func Local(node ast.Node)map[string]bool{out:=make(map[string]bool);ast.Inspect(node,func(n ast.Node)bool{out["x"]=true;return false});return out}
func Owned(node ast.Node){out:=Local(node);out["caller"]=true}
func Nested(node ast.Node)map[string]bool{out:=make(map[string]bool);ast.Inspect(node,func(n ast.Node)bool{ast.Inspect(n,func(child ast.Node)bool{out["x"]=true;return false});return false});return out}
func NestedCaller(node ast.Node){out:=Nested(node);out["caller"]=true}
func borrowed(out map[string]bool,node ast.Node)map[string]bool{ast.Inspect(node,func(n ast.Node)bool{out["x"]=true;return false});return out}
func Borrowed(out map[string]bool,node ast.Node){result:=borrowed(out,node);result["caller"]=true}
func published(node ast.Node)map[string]bool{out:=make(map[string]bool);ast.Inspect(node,func(n ast.Node)bool{saved=out;out["x"]=true;return false});return out}
func Published(node ast.Node){out:=published(node);out["caller"]=true}
func header(node ast.Node,input map[string]bool)map[string]bool{out:=make(map[string]bool);ast.Inspect(node,func(n ast.Node)bool{out=input;out["x"]=true;return false});return out}
func Header(node ast.Node,input map[string]bool){out:=header(node,input);out["caller"]=true}
func address(node ast.Node,input map[string]bool)map[string]bool{out:=make(map[string]bool);ast.Inspect(node,func(n ast.Node)bool{slot:=&out;*slot=input;out["x"]=true;return false});return out}
func Address(node ast.Node,input map[string]bool){out:=address(node,input);out["caller"]=true}
func alias(node ast.Node)map[string]bool{out:=make(map[string]bool);ast.Inspect(node,func(n ast.Node)bool{other:=out;saved=other;out["x"]=true;return false});return out}
func Alias(node ast.Node){out:=alias(node);out["caller"]=true}
func named(node ast.Node)map[string]bool{out:=make(map[string]bool);visit:=func(n ast.Node)bool{out["x"]=true;return false};ast.Inspect(node,visit);return out}
func Named(node ast.Node){out:=named(node);out["caller"]=true}
func indirect(node ast.Node)map[string]bool{out:=make(map[string]bool);inspect:=ast.Inspect;inspect(node,func(n ast.Node)bool{out["x"]=true;return false});return out}
func Indirect(node ast.Node){out:=indirect(node);out["caller"]=true}
func escaping(node ast.Node)map[string]bool{out:=make(map[string]bool);retain(node,func(n ast.Node)bool{out["x"]=true;return false});return out}
func Escaping(node ast.Node){out:=escaping(node);out["caller"]=true}
func async(node ast.Node)map[string]bool{out:=make(map[string]bool);go ast.Inspect(node,func(n ast.Node)bool{out["x"]=true;return false});return out}
func Async(node ast.Node){out:=async(node);out["caller"]=true}
func deferred(node ast.Node)map[string]bool{out:=make(map[string]bool);defer ast.Inspect(node,func(n ast.Node)bool{out["x"]=true;return false});return out}
func Deferred(node ast.Node){out:=deferred(node);out["caller"]=true}
func outerAsync(node ast.Node)map[string]bool{out:=make(map[string]bool);go func(){ast.Inspect(node,func(n ast.Node)bool{out["x"]=true;return false})}();return out}
func OuterAsync(node ast.Node){out:=outerAsync(node);out["caller"]=true}
func child(node ast.Node,input *int)map[string]*int{out:=make(map[string]*int);ast.Inspect(node,func(n ast.Node)bool{out["x"]=input;return false});return out}
func Child(node ast.Node,input *int){out:=child(node,input);*out["x"]=2}
func unknown(node ast.Node,callback func())map[string]bool{out:=make(map[string]bool);ast.Inspect(node,func(n ast.Node)bool{callback();out["x"]=true;return false});return out}
func Unknown(node ast.Node,callback func()){out:=unknown(node,callback);out["caller"]=true}
func ioMap(node ast.Node)map[string]bool{out:=make(map[string]bool);ast.Inspect(node,func(n ast.Node)bool{println("visit");out["x"]=true;return false});return out}
func IO(node ast.Node){out:=ioMap(node);out["caller"]=true}
func mutatingNode(node ast.Node)map[string]bool{out:=make(map[string]bool);ast.Inspect(node,func(n ast.Node)bool{n.(*ast.Ident).Name="changed";out["x"]=true;return false});return out}
func MutatingNode(node ast.Node){out:=mutatingNode(node);out["caller"]=true}
`
	root := qualitySourceModule(t, []byte(source))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Owned", "NestedCaller"} {
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s loses private Inspect map: %+v", name, result)
		}
	}
	for _, name := range []string{"Owned", "NestedCaller", "Unknown", "IO", "MutatingNode"} {
		writes := findFunction(t, facts, name).Mutations
		if len(writes) != 1 || writes[0].Provenance != quality.Local {
			t.Errorf("%s loses raw local write: %+v", name, writes)
		}
	}
	for _, name := range []string{"Borrowed", "Published", "Header", "Address", "Alias", "Named", "Indirect", "Escaping", "Async", "Deferred", "OuterAsync", "Child"} {
		writes := findFunction(t, facts, name).Mutations
		if len(writes) != 1 || writes[0].Provenance == quality.Local || len(resultNamed(t, report, name).Effects) == 0 {
			t.Errorf("%s hides borrowed or unproved write: %+v", name, writes)
		}
	}
	for name, kind := range map[string]quality.Kind{"Borrowed": quality.MutationEffect, "Published": quality.Global, "Unknown": quality.UnknownEffect, "IO": quality.IO, "MutatingNode": quality.MutationEffect} {
		if !slices.ContainsFunc(resultNamed(t, report, name).Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("%s hides independent %s evidence", name, kind)
		}
	}
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
		t.Fatal("Inspect map facts change after JSON replay", err)
	}
	for _, path := range []string{"go/ast.Inspect", "example.com/quality-repro.Local"} {
		for _, kind := range []quality.Kind{quality.IO, quality.MutationEffect, quality.UnknownEffect} {
			config := quality.DefaultConfig()
			config.Prefixes = append(config.Prefixes, quality.Prefix{Path: path, Kind: kind})
			overridden, err := quality.Evaluate(restored, config)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.ContainsFunc(resultNamed(t, overridden, "Owned").Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
				t.Errorf("%s policy hides %s evidence", path, kind)
			}
		}
	}
}

// Copy actual declarations byte for byte. Support records only supply the fields
// those declarations read; no production application body is rewritten.
func inspectSourceFunctions(t *testing.T, path string, names ...string) string {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	var found int
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !slices.Contains(names, fn.Name.Name) {
			continue
		}
		out.Write(source[fset.Position(fn.Pos()).Offset:fset.Position(fn.End()).Offset])
		out.WriteByte('\n')
		found++
	}
	if found != len(names) {
		t.Fatalf("missing current source declarations in %s: %d of %d", path, found, len(names))
	}
	return out.String()
}

func BenchmarkInspectMapExtraction(b *testing.B) {
	root := os.Getenv("GSTYLE_BENCH_ROOT")
	if root == "" {
		b.Skip("set GSTYLE_BENCH_ROOT to a fixed source corpus")
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := (Analyzer{Root: root}).Extract(context.Background(), nil); err != nil {
			b.Fatal(err)
		}
	}
}
