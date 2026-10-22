package gopatterns

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestConcreteReturnCacheInvalidation(t *testing.T) {
	root, _ := sourceFixModule(t, map[string]string{
		"helper.go": `package sample;type Target struct{};func factory()any{return &Target{}}`,
		"caller.go": `package sample;func check(){value:=factory();_=value.(*Target)}`,
	})
	cache := &FixExtractor{}
	before, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{})
	if err != nil {
		t.Fatal(err)
	}
	if hits := findCachedFunction(before, ".check").ForceTypeAssertHits; len(hits) != 0 {
		t.Fatalf("safe contract lost: %+v", hits)
	}
	helper := filepath.Join(root, "helper.go")
	cache.Invalidate([]string{helper})
	revision := SourceOverlay{Files: map[string][]byte{helper: []byte(`package sample;type Target struct{};func factory()any{return nil}`)}, Disk: before.Sources.EvidenceOnly()}
	next, err := cache.Extract(context.Background(), root, []string{"."}, revision)
	if err != nil {
		t.Fatal(err)
	}
	full, err := ExtractFixOverlay(context.Background(), root, []string{"."}, revision)
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range []*Extraction{next, full} {
		if hits := findCachedFunction(result, ".check").ForceTypeAssertHits; len(hits) != 1 {
			t.Fatalf("stale contract hides nil return: %+v", hits)
		}
	}
}

func TestConcreteReturnAssertions(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example
type Target struct{}
func (*Target) M(){}
type Other struct{}
type Base struct{}
func (Base) Make()any{return &Target{}}
type Maker interface{Make()any}
var global any
func build()any{copy:=Target{};return &copy}
func branches(flag bool)any{if flag{return &Target{}};return &Target{}}
func typedNil()any{return (*Target)(nil)}
func stringResult()any{return "safe"}
func empty()any{return nil}
func nilPath(flag bool)any{if flag{return nil};return &Target{}}
func mixed(flag bool)any{if flag{return &Other{}};return &Target{}}
func errorPath(flag bool)(any,error){if flag{return nil,problem{}};return &Target{},nil}
type problem struct{}
func(problem)Error()string{return "bad"}
func named()(result any){return &Target{}}
func deferred()(result any){defer func(){result=nil}();return &Target{}}
func recovering(flag bool)any{defer func(){recover()}();if flag{panic("bad")};return &Target{}}
func passthrough(value any)any{return value}
func generic[T any]()any{var value T;return value}
func reversed()(error,any){return nil,&Target{}}
func replace(value *any){*value=nil}
func checkDirect(){value:=build();_=value.(*Target)}
func checkExpression(){_=build().(*Target)}
func checkAlias(){value:=build();alias:=value;_=alias.(*Target)}
func checkBranches(flag bool){value:=branches(flag);_=value.(*Target)}
func checkTypedNil(){value:=typedNil();_=value.(*Target)}
func checkMethod(){value:=(Base{}).Make();_=value.(*Target)}
func checkMethodExpression(){value:=Base.Make(Base{});_=value.(*Target)}
func checkInterface(){value:=build();_=value.(interface{M()})}
func checkJoin(flag bool){var value any;if flag{value=build()}else{value=typedNil()};_=value.(*Target)}
func checkString(){value:=stringResult();_=value.(string)}
func checkAny(){value:=typedNil();_=value.(any)}
func checkNil(flag bool){value:=nilPath(flag);_=value.(*Target)}
func checkMixed(flag bool){value:=mixed(flag);_=value.(*Target)}
func checkError(flag bool){value,err:=errorPath(flag);_=err;_=value.(*Target)}
func checkWrong(){value:=build();_=value.(*Other)}
func checkOverwrite(){value:=build();value=nil;_=value.(*Target)}
func checkMixedJoin(flag bool){value:=build();if flag{value=nil};_=value.(*Target)}
func checkEscape(){value:=build();replace(&value);_=value.(*Target)}
func checkCapture(){value:=build();func(){value=nil}();_=value.(*Target)}
func checkGlobal(){global=build();_=global.(*Target)}
func checkField(){box:=struct{value any}{build()};_=box.value.(*Target)}
func checkDynamic(m Maker){value:=m.Make();_=value.(*Target)}
func checkUnknown(value any){_=value.(*Target)}
func checkNamed(){value:=named();_=value.(*Target)}
func checkDeferred(){value:=deferred();_=value.(*Target)}
func checkRecovery(flag bool){value:=recovering(flag);_=value.(*Target)}
func checkPassthrough(){value:=passthrough(nil);_=value.(*Target)}
func checkGeneric(){value:=generic[*Target]();_=value.(*Target)}
func checkSecondResult(){_,value:=reversed();_=value.(*Target)}
func checkEmpty(){value:=empty();_=value.(any)}
func checkPartialJoin(flag bool){var value any;if flag{value=build()};_=value.(*Target)}
func checkFunctionValue(){makeValue:=build;value:=makeValue();_=value.(*Target)}
`)
	want := map[string]int{
		"checkDirect": 0, "checkExpression": 0, "checkAlias": 0, "checkBranches": 0,
		"checkTypedNil": 0, "checkMethod": 0, "checkMethodExpression": 0, "checkInterface": 0, "checkJoin": 0,
		"checkString": 0, "checkAny": 0,
		"checkNil": 1, "checkMixed": 1, "checkError": 1, "checkWrong": 1, "checkOverwrite": 1,
		"checkMixedJoin": 1, "checkEscape": 1, "checkCapture": 1, "checkGlobal": 1, "checkField": 1,
		"checkDynamic": 1, "checkUnknown": 1, "checkNamed": 1, "checkDeferred": 1, "checkRecovery": 1,
		"checkPassthrough": 1, "checkGeneric": 1,
		"checkSecondResult": 1, "checkEmpty": 1, "checkPartialJoin": 1, "checkFunctionValue": 1,
	}
	seen := 0
	for _, function := range extraction.Funcs {
		name := function.Name[strings.LastIndex(function.Name, ".")+1:]
		expected, selected := want[name]
		if !selected {
			continue
		}
		seen++
		if len(function.ForceTypeAssertHits) != expected {
			t.Errorf("%s: got %+v, want %d warnings", name, function.ForceTypeAssertHits, expected)
		}
	}
	if seen != len(want) {
		t.Fatalf("checked %d functions, want %d", seen, len(want))
	}
}

func TestConcreteReturnAcrossFiles(t *testing.T) {
	extraction, _ := proofExtractionWithExtra(t,
		`package example;func check(){value:=factory();_=value.(*Target)}`,
		`package example;type Target struct{};func factory()any{return &Target{}}`)
	for _, function := range extraction.Funcs {
		if strings.HasSuffix(function.Name, ".check") && len(function.ForceTypeAssertHits) != 0 {
			t.Fatalf("cross-file contract lost: %+v", function.ForceTypeAssertHits)
		}
	}
}

func TestConcreteReturnRequiresCanonicalEvidence(t *testing.T) {
	for _, missing := range []string{"return operand", "helper flow", "caller operand", "caller flow"} {
		t.Run(missing, func(t *testing.T) {
			pkg := loadTestPackage(t, `package p;type Target struct{};func factory()any{return &Target{}};func check(){value:=factory();_=value.(*Target)}`)
			root := filepath.Dir(pkg.Fset.PositionFor(pkg.Syntax[0].Pos(), false).Filename)
			analysis := packageAnalysis{pkg: pkg, root: root, pool: pdg.NewBuilder()}
			if err := analysis.build(); err != nil {
				t.Fatal(err)
			}
			for _, source := range analysis.graphs {
				selected := strings.HasPrefix(missing, "caller") == (source.declaration.Name.Name == "check")
				if !selected {
					continue
				}
				graph := source.graph
				if strings.HasSuffix(missing, "flow") {
					graph.NormalFlow, graph.ValueFlow, graph.Extensions = nil, nil, nil
					continue
				}
				kept := graph.Edges[:0]
				removed := 0
				for _, edge := range graph.Edges {
					target := graph.Nodes[edge.Target-1]
					assertion := graph.Tables.Text(graph.NodeAttributes(target).Operator.Value) == "type-assert"
					if edge.Kind == pdg.Data && edge.Position == 1 && (target.Category == pdg.Return || assertion) {
						removed++
						continue
					}
					kept = append(kept, edge)
				}
				graph.Edges = kept
				if removed == 0 {
					t.Fatal("fixture lacks the operand under test")
				}
			}
			if err := analysis.summarize(); err != nil {
				t.Fatal(err)
			}
			if err := analysis.analyze(); err != nil {
				t.Fatal(err)
			}
			for _, function := range analysis.result {
				if strings.HasSuffix(function.Name, ".check") && len(function.ForceTypeAssertHits) != 1 {
					t.Fatalf("missing %s hides warning: %+v", missing, function.ForceTypeAssertHits)
				}
			}
		})
	}
}
