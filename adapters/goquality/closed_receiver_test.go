package goquality

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestClosedReceiverStorageBoundaries(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "encoding/json"
type cursor struct{ n int }
func(p *cursor) ClosedCursor(){p.n++}
func cursorRun(){p:=new(cursor);p.ClosedCursor()}
type problem struct{ message string }
func(p *problem) ClosedError(){p.message="error"}
func errorRun(){p:=&problem{};p.ClosedError()}
type constructed struct{n int}
func(p *constructed)ConstructorLocal(){p.n++}
func newConstructed()*constructed{p:=&constructed{};p.n=1;return p}
func constructedRun(){p:=newConstructed();p.ConstructorLocal()}
type literalConstructed struct{n int}
func(p *literalConstructed)LiteralConstructor(){p.n++}
func newLiteralConstructed()*literalConstructed{return &literalConstructed{}}
func literalConstructedRun(){p:=newLiteralConstructed();p.LiteralConstructor()}
type escapedConstructor struct{n int}
func(p *escapedConstructor)EscapedConstructor(){p.n++}
func newEscapedConstructor()*escapedConstructor{return new(escapedConstructor)}
func escapedConstructorRun()any{p:=newEscapedConstructor();p.EscapedConstructor();return p}
type indirectConstructor struct{n int}
func(p *indirectConstructor)IndirectConstructor(){p.n++}
func newIndirectConstructor()*indirectConstructor{return &indirectConstructor{}}
var indirectFactory=newIndirectConstructor
func indirectConstructorRun(){p:=indirectFactory();p.IndirectConstructor()}
type interfaceConstructor struct{n int}
func(p *interfaceConstructor)InterfaceConstructor(){p.n++}
func newInterfaceConstructor()*interfaceConstructor{return &interfaceConstructor{}}
var published any
func interfaceConstructorRun(){var p any=newInterfaceConstructor();published=p}
type interfaceLiteral struct{n int}
func(p *interfaceLiteral)InterfaceLiteral(){p.n++}
func interfaceLiteralRun(){var p any=&interfaceLiteral{};published=p}
type returned struct{ n int }
func(p *returned) Returned(){p.n++}
func returnedRun()any{p:=new(returned);return p}
type holder struct{ n int }
func(p *holder) PointerHolder(){p.n++}
func exposeHolder()any{p:=new(*holder);json.Unmarshal([]byte("{}"),p);return *p}
type global struct{ n int }
var globalValue=new(global)
func(p *global) InferredGlobal(){p.n++}
type fresh struct{ n int }
func(p *fresh) FreshEscape(){p.n++}
func freshRun()any{return &fresh{}}
type parameter struct{ n int }
func(p *parameter) PublicParameter(){p.n++}
func ExposeParameter(p *parameter){}
type result struct{ n int }
func(p *result) PublicResult(){p.n++}
func ExposeResult()*result{return new(result)}
type methodValue struct{ n int }
func(p *methodValue) MethodValue(){p.n++}
func methodRun()func(){p:=new(methodValue);return p.MethodValue}
type captured struct{ n int }
func(p *captured) Captured(){p.n++}
func captureRun()func(){p:=new(captured);return func(){p.Captured()}}
type concurrent struct{ n int }
func(p *concurrent) Concurrent(){p.n++}
func concurrentRun(){p:=new(concurrent);go p.Concurrent()}
type deferred struct{ n int }
func(p *deferred) Deferred(){p.n++}
func deferredRun(){p:=new(deferred);defer p.Deferred()}
type addressed struct{ n int }
var escapedInt *int
func(p *addressed) FieldAddress(){p.n++}
func addressRun(){p:=new(addressed);escapedInt=&p.n}
type arrayView struct{ n int; values [2]int }
var escapedSlice []int
func(p *arrayView) ArrayView(){p.n++}
func viewRun(){p:=new(arrayView);escapedSlice=p.values[:]}
type base struct{}
var escapedBase *base
func(p *base) Expose(){escapedBase=p}
type promoted struct{base;n int}
func(p *promoted) Promoted(){p.n++}
func promotedRun(){p:=new(promoted);p.Expose()}
type scalar int
var escapedScalar *scalar
func(p *scalar) Save(){escapedScalar=p}
type scalarField struct{ value scalar; n int }
func(p *scalarField) ScalarPointerMethod(){p.n++}
func scalarRun(){p:=new(scalarField);p.value.Save()}
type aliasScalar int
type aliasPointer = *aliasScalar
var escapedAlias aliasPointer
func(p aliasPointer) SaveAlias(){escapedAlias=p}
type aliasField struct{ value aliasScalar; n int }
func(p *aliasField) AliasedPointerMethod(){p.n++}
func aliasRun(){p:=new(aliasField);p.value.SaveAlias()}
type sharedBase struct{ count int }
type embeddedPointer struct{*sharedBase}
func(p *embeddedPointer) PromotedWrite(){p.count++}
func promotedWriteRun(shared *sharedBase){p:=&embeddedPointer{shared};p.PromotedWrite()}
type aliasTarget struct{ n int }
type publicAlias = aliasTarget
func(p *aliasTarget) TypeAlias(){p.n++}
func aliasTargetRun(){p:=new(aliasTarget);p.TypeAlias()}
type borrowed struct{ n int; values map[string]int }
func(p *borrowed) BorrowedMap(){p.values["key"]=1}
func borrowedRun(values map[string]int){p:=new(borrowed);p.values=values;p.BorrowedMap()}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		want quality.Provenance
	}{
		{"ClosedCursor", quality.Local}, {"ClosedError", quality.Local},
		{"ConstructorLocal", quality.Local}, {"LiteralConstructor", quality.Local},
		{"EscapedConstructor", quality.External}, {"IndirectConstructor", quality.External},
		{"InterfaceConstructor", quality.External}, {"InterfaceLiteral", quality.External},
		{"Returned", quality.External}, {"PointerHolder", quality.External}, {"InferredGlobal", quality.External}, {"FreshEscape", quality.External},
		{"PublicParameter", quality.External}, {"PublicResult", quality.External}, {"MethodValue", quality.External},
		{"Captured", quality.External}, {"Concurrent", quality.External}, {"Deferred", quality.External},
		{"FieldAddress", quality.External}, {"ArrayView", quality.External}, {"Promoted", quality.External},
		{"ScalarPointerMethod", quality.External}, {"AliasedPointerMethod", quality.External}, {"PromotedWrite", quality.External}, {"TypeAlias", quality.External}, {"BorrowedMap", quality.External},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fact := findFunction(t, facts, tc.name)
			if len(fact.Mutations) != 1 || fact.Mutations[0].Provenance != tc.want {
				t.Fatalf("mutations=%+v, want one %s write", fact.Mutations, tc.want)
			}
		})
	}
}

func TestClosedReceiverTestVariantEscape(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
type testCursor struct{n int}
func(p *testCursor) TestVariantWrite(){p.n++}
func newTestCursor()*testCursor{return &testCursor{}}
func run(){p:=newTestCursor();p.TestVariantWrite()}
`))
	if err := os.WriteFile(filepath.Join(root, "input_test.go"), []byte(`package repro
var escapedTest any
func init(){p:=newTestCursor();escapedTest=p}
`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tests := range []bool{false, true} {
		facts, err := (Analyzer{Root: root, Tests: tests}).Extract(context.Background(), []string{"input.go"})
		if err != nil {
			t.Fatal(err)
		}
		want := quality.Local
		if tests {
			want = quality.External
		}
		fact := findFunction(t, facts, "TestVariantWrite")
		if len(fact.Mutations) != 1 || fact.Mutations[0].Provenance != want {
			t.Fatalf("tests=%t mutations=%+v, want %s", tests, fact.Mutations, want)
		}
	}
}

func TestClosedReceiverFreshFields(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
type ownedMap struct{values map[string]int}
func(p *ownedMap)OwnedMap(){p.values["x"]=1;delete(p.values,"y");clear(p.values)}
func ownedMapRun(){p:=ownedMap{values:make(map[string]int)};p.OwnedMap()}
type ownedSlice struct{values []int}
func(p *ownedSlice)OwnedSlice(){p.values=append(p.values,1);p.values[0]=2}
func ownedSliceRun(){var p ownedSlice;p.OwnedSlice()}
type resetMap struct{values map[string]int}
func(p *resetMap)ResetMap(){p.values=map[string]int{};p.values["x"]=1}
func resetMapRun(){p:=new(resetMap);p.ResetMap()}
type finishedSlice struct{values []int}
func(p *finishedSlice)FinishedSlice(){p.values=append(p.values,1)}
func finishedSliceRun()[]int{p:=finishedSlice{};p.FinishedSlice();return p.values}
type finishedMap struct{values map[string]int}
func(p *finishedMap)FinishedMap(){p.values["x"]=1}
func finishedMapRun()map[string]int{var p finishedMap;p.values=make(map[string]int);p.FinishedMap();return p.values}
type deferredSlice struct{values []int}
var sharedSlice []int
func(p *deferredSlice)DeferredPublication(){p.values=append(p.values,1)}
func deferredSliceRun()(out []int){p:=deferredSlice{};p.DeferredPublication();defer func(){sharedSlice=out;out[0]=2}();return p.values}
type borrowedMap struct{values map[string]int}
func(p *borrowedMap)BorrowedField(){p.values["x"]=1}
func borrowedMapRun(values map[string]int)map[string]int{p:=borrowedMap{values:values};p.BorrowedField();return p.values}
type replacedMap struct{values map[string]int}
var sharedMap=map[string]int{}
func(p *replacedMap)ReplacedField(){p.values=sharedMap;p.values["x"]=1}
func replacedMapRun(){p:=replacedMap{values:map[string]int{}};p.ReplacedField()}
type publishedMap struct{values map[string]int}
func(p *publishedMap)PublishedField(){sharedMap=p.values;p.values["x"]=1}
func publishedMapRun(){p:=publishedMap{values:map[string]int{}};p.PublishedField()}
type returnedMap struct{values map[string]int}
func(p *returnedMap)ReturnedField()map[string]int{p.values["x"]=1;return p.values}
func returnedMapRun(){p:=returnedMap{values:map[string]int{}};sharedMap=p.ReturnedField();p.ReturnedField()}
type aliasedMap struct{values map[string]int}
func(p *aliasedMap)AliasedField(){alias:=p.values;delete(alias,"y");p.values["x"]=1}
func aliasedMapRun(){p:=aliasedMap{values:map[string]int{}};p.AliasedField()}
func consume(values map[string]int){}
type passedMap struct{values map[string]int}
func(p *passedMap)PassedField(){consume(p.values);p.values["x"]=1}
func passedMapRun(){p:=passedMap{values:map[string]int{}};p.PassedField()}
var element *int
type addressedSlice struct{values []int}
func(p *addressedSlice)AddressedElement(){element=&p.values[0];p.values[0]=1}
func addressedRun(){p:=addressedSlice{values:[]int{0}};p.AddressedElement()}
type elementBox struct{value int}
var exposedBox *elementBox
func(p *elementBox)Expose(){exposedBox=p}
type methodSlice struct{values []elementBox}
func(p *methodSlice)ElementMethod(){p.values[0].Expose();p.values[0].value=1}
func methodSliceRun(){p:=methodSlice{values:[]elementBox{{}}};p.ElementMethod()}
type shallowSlice struct{values []*int}
func(p *shallowSlice)BorrowedElement(){*p.values[0]=1}
func shallowRun(input *int)[]*int{p:=shallowSlice{values:[]*int{input}};p.BorrowedElement();return p.values}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		local bool
	}{
		{"OwnedMap", true}, {"OwnedSlice", true}, {"ResetMap", true},
		{"FinishedSlice", true}, {"FinishedMap", true}, {"DeferredPublication", false},
		{"BorrowedField", false}, {"ReplacedField", false}, {"PublishedField", false}, {"ReturnedField", false},
		{"AliasedField", false}, {"PassedField", false}, {"AddressedElement", false}, {"ElementMethod", false}, {"BorrowedElement", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fact := findFunction(t, facts, tc.name)
			if len(fact.Mutations) == 0 {
				t.Fatal("missing mutation")
			}
			got := fact.Mutations[len(fact.Mutations)-1].Provenance
			if (got == quality.Local) != tc.local {
				t.Fatalf("mutation=%s, want local=%t: %+v", got, tc.local, fact.Mutations)
			}
		})
	}
}

func TestClosedReceiverFieldTestVariant(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
type privateCache struct{values map[string]int}
func(p *privateCache)CacheWrite(){p.values["x"]=1}
func run(){p:=privateCache{values:map[string]int{}};p.CacheWrite()}
type otherCache struct{values map[string]int}
func(p *otherCache)OtherCacheWrite(){p.values["x"]=1}
func otherRun(){p:=otherCache{values:map[string]int{}};p.OtherCacheWrite()}
`))
	if err := os.WriteFile(filepath.Join(root, "input_test.go"), []byte(`package repro
var published map[string]int
func(p *privateCache)publish(){published=p.values}
`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tests := range []bool{false, true} {
		facts, err := (Analyzer{Root: root, Tests: tests}).Extract(context.Background(), []string{"input.go"})
		if err != nil {
			t.Fatal(err)
		}
		fact := findFunction(t, facts, "CacheWrite")
		got := fact.Mutations[0].Provenance
		if (got == quality.Local) == tests {
			t.Fatalf("tests=%t mutation=%s", tests, got)
		}
		if other := findFunction(t, facts, "OtherCacheWrite"); other.Mutations[0].Provenance != quality.Local {
			t.Fatalf("same-named field loses independent proof: %+v", other.Mutations)
		}
	}
}
